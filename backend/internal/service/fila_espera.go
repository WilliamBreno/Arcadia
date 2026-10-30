package service

import (
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Lista de espera de ingresso esgotado (item 7.3). Quando uma vaga volta
// (reserva expirada, cancelamento, cortesia revogada, mais estoque), ela fica
// guardada só para a primeira pessoa da fila, na quantidade que ela pediu,
// com prazo para comprar: 2 horas, ou 30 minutos quando faltam menos de 24 h
// para o evento (decisões do dono, 29/10/2026). Passou o prazo, a vez vai
// para a próxima pessoa.

const (
	PrazoFila         = 2 * time.Hour
	PrazoFilaPerto    = 30 * time.Minute
	PertoDoEventoFila = 24 * time.Hour
	MaxQuantidadeFila = 20
)

const (
	FilaAguardando = "aguardando"
	FilaConvidado  = "convidado"
	FilaReservou   = "reservou"
	FilaComprou    = "comprou"
	FilaExpirou    = "expirou"
	FilaSaiu       = "saiu"
)

var (
	ErrAindaHaVaga         = errors.New("ainda há ingressos à venda: é só comprar")
	ErrJaNaFila            = errors.New("você já está na lista de espera deste ingresso")
	ErrFilaIndisponivel    = errors.New("lista de espera indisponível para este ingresso")
	ErrQuantidadeFila      = errors.New("quantidade fora do permitido para este ingresso")
	ErrNaoEstaNaFila       = errors.New("você não está na lista de espera deste ingresso")
	statusFilaAtivos       = []string{FilaAguardando, FilaConvidado, FilaReservou}
	statusPedidoEncerrados = []domain.StatusPedido{domain.StatusPedidoExpirado, domain.StatusPedidoCancelado}
)

type FilaEspera struct {
	ID             int64 `gorm:"primaryKey"`
	EventoID       int64
	TipoIngressoID int64
	UsuarioID      int64
	Quantidade     int
	Status         string
	ConvidadoEm    *time.Time
	PrazoEm        *time.Time
	PedidoID       *int64
	CriadoEm       time.Time
	AtualizadoEm   time.Time
}

func (FilaEspera) TableName() string { return "fila_espera" }

// --- Vagas guardadas (usadas também no checkout) ---

// VagasGuardadas soma o que está guardado para convidados com prazo válido
// neste tipo, sem contar o próprio usuário (para ele a vaga é dele).
func VagasGuardadas(tx *gorm.DB, tipoID, excetoUsuarioID int64) (int64, error) {
	var n int64
	err := tx.Model(&FilaEspera{}).Select("COALESCE(SUM(quantidade),0)").
		Where("tipo_ingresso_id = ? AND status = ? AND prazo_em > ? AND usuario_id <> ?", tipoID, FilaConvidado, time.Now(), excetoUsuarioID).
		Scan(&n).Error
	// Item 10.3: ingressos segurados por grupos de compra abertos.
	return n + vagasDeGrupos(tx, tipoID), err
}

// VagasGuardadasPorTipos é a versão para a página pública (vários tipos).
func VagasGuardadasPorTipos(db *gorm.DB, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var linhas []struct {
		TipoIngressoID int64
		Total          int64
	}
	err := db.Model(&FilaEspera{}).Select("tipo_ingresso_id, SUM(quantidade) AS total").
		Where("tipo_ingresso_id IN ? AND status = ? AND prazo_em > ?", ids, FilaConvidado, time.Now()).
		Group("tipo_ingresso_id").Scan(&linhas).Error
	for _, l := range linhas {
		out[l.TipoIngressoID] = l.Total
	}
	for id, n := range vagasDeGruposPorTipos(db, ids) { // item 10.3
		out[id] += n
	}
	return out, err
}

// consumirConvite: a pessoa convidada reservou; a vaga guardada vira a reserva dela.
func consumirConvite(tx *gorm.DB, usuarioID, tipoID, pedidoID int64) error {
	return tx.Model(&FilaEspera{}).
		Where("usuario_id = ? AND tipo_ingresso_id = ? AND status = ? AND prazo_em > ?", usuarioID, tipoID, FilaConvidado, time.Now()).
		Updates(map[string]any{"status": FilaReservou, "pedido_id": pedidoID, "atualizado_em": time.Now()}).Error
}

// PrazoParaComprar: 2 h, ou 30 min se faltam menos de 24 h para o evento;
// nunca depois do início do evento nem do fim das vendas do ingresso. Testada.
func PrazoParaComprar(agora time.Time, inicioEvento, fimVendas *time.Time) time.Time {
	prazo := agora.Add(PrazoFila)
	if inicioEvento != nil && inicioEvento.Sub(agora) < PertoDoEventoFila {
		prazo = agora.Add(PrazoFilaPerto)
	}
	if inicioEvento != nil && inicioEvento.After(agora) && prazo.After(*inicioEvento) {
		prazo = *inicioEvento
	}
	if fimVendas != nil && fimVendas.After(agora) && prazo.After(*fimVendas) {
		prazo = *fimVendas
	}
	return prazo
}

// --- Serviço ---

type FilaEsperaService struct {
	db          *gorm.DB
	itens       *repository.ItemPedidoRepository
	mail        *mail.Cliente
	frontendURL string
	nome        string
}

func NovoFilaEsperaService(db *gorm.DB, itens *repository.ItemPedidoRepository, m *mail.Cliente, frontendURL, nome string) *FilaEsperaService {
	return &FilaEsperaService{db: db, itens: itens, mail: m, frontendURL: strings.TrimRight(frontendURL, "/"), nome: nome}
}

// tipoAVenda: o evento está publicado e não acabou, e o tipo ainda vende.
func tipoAVenda(e *domain.Evento, t *domain.TipoIngresso, agora time.Time) bool {
	if e.Status != domain.StatusEventoPublicado || !t.Ativo {
		return false
	}
	fim := e.InicioEm
	if e.FimEm != nil {
		fim = e.FimEm
	}
	if fim != nil && !fim.After(agora) {
		return false
	}
	return t.VendasFim == nil || t.VendasFim.After(agora)
}

// livres: vagas que qualquer pessoa poderia comprar agora (dentro da transação
// que trava o tipo).
func (s *FilaEsperaService) livres(tx *gorm.DB, t *domain.TipoIngresso) (int64, error) {
	ativos, err := s.itens.ContarAtivosPorTipo(tx, t.ID)
	if err != nil {
		return 0, err
	}
	guardadas, err := VagasGuardadas(tx, t.ID, 0)
	if err != nil {
		return 0, err
	}
	// Item 7.4: anúncios da revenda também são vagas para a fila.
	aVenda, err := RevendasAVenda(tx, t.ID)
	if err != nil {
		return 0, err
	}
	return VagasParaComprar(int64(t.Quantidade)-ativos, aVenda, guardadas), nil
}

// Entrar põe a pessoa na fila de um tipo esgotado.
func (s *FilaEsperaService) Entrar(usuarioID int64, evento *domain.Evento, tipoID int64, quantidade int) (*FilaEspera, error) {
	var entrada *FilaEspera
	err := s.db.Transaction(func(tx *gorm.DB) error {
		t, err := s.itens.BuscarTipoIngressoParaAtualizar(tx, tipoID)
		if err != nil || t.EventoID != evento.ID {
			return ErrFilaIndisponivel
		}
		if !tipoAVenda(evento, t, time.Now()) {
			return ErrFilaIndisponivel
		}
		max := t.MaxPorPedido
		if max <= 0 || max > MaxQuantidadeFila {
			max = MaxQuantidadeFila
		}
		if quantidade < max1(t.MinPorPedido) || quantidade > max {
			return ErrQuantidadeFila
		}
		livres, err := s.livres(tx, t)
		if err != nil {
			return err
		}
		if livres >= int64(quantidade) {
			return ErrAindaHaVaga
		}
		var ja int64
		if err := tx.Model(&FilaEspera{}).Where("tipo_ingresso_id = ? AND usuario_id = ? AND status IN ?", tipoID, usuarioID, statusFilaAtivos).Count(&ja).Error; err != nil {
			return err
		}
		if ja > 0 {
			return ErrJaNaFila
		}
		agora := time.Now()
		entrada = &FilaEspera{EventoID: evento.ID, TipoIngressoID: tipoID, UsuarioID: usuarioID, Quantidade: quantidade, Status: FilaAguardando, CriadoEm: agora, AtualizadoEm: agora}
		return tx.Create(entrada).Error
	})
	return entrada, err
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// Sair tira a pessoa da fila. Se a vaga estava guardada para ela, vai para a próxima.
func (s *FilaEsperaService) Sair(usuarioID, tipoID int64) error {
	var eraConvidado bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var e FilaEspera
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tipo_ingresso_id = ? AND usuario_id = ? AND status IN ?", tipoID, usuarioID, []string{FilaAguardando, FilaConvidado}).
			First(&e).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNaoEstaNaFila
		}
		if err != nil {
			return err
		}
		eraConvidado = e.Status == FilaConvidado
		return tx.Model(&e).Updates(map[string]any{"status": FilaSaiu, "atualizado_em": time.Now()}).Error
	})
	if err == nil && eraConvidado {
		s.promoverTipo(tipoID)
	}
	return err
}

// EntradaFila é o que a página do evento mostra para a pessoa.
type EntradaFila struct {
	TipoIngressoID int64      `json:"tipo_ingresso_id"`
	Quantidade     int        `json:"quantidade"`
	Status         string     `json:"status"`
	Posicao        int64      `json:"posicao,omitempty"`
	PrazoEm        *time.Time `json:"prazo_em,omitempty"`
}

func (s *FilaEsperaService) Minhas(usuarioID, eventoID int64) ([]EntradaFila, error) {
	var es []FilaEspera
	if err := s.db.Where("usuario_id = ? AND evento_id = ? AND status IN ?", usuarioID, eventoID, []string{FilaAguardando, FilaConvidado}).
		Order("id").Find(&es).Error; err != nil {
		return nil, err
	}
	out := make([]EntradaFila, 0, len(es))
	for _, e := range es {
		item := EntradaFila{TipoIngressoID: e.TipoIngressoID, Quantidade: e.Quantidade, Status: e.Status}
		if e.Status == FilaConvidado {
			item.PrazoEm = e.PrazoEm
		} else {
			var antes int64
			s.db.Model(&FilaEspera{}).Where("tipo_ingresso_id = ? AND status = ? AND (criado_em, id) < (?, ?)", e.TipoIngressoID, FilaAguardando, e.CriadoEm, e.ID).Count(&antes)
			item.Posicao = antes + 1
		}
		out = append(out, item)
	}
	return out, nil
}

// ResumoOrganizador: quantas pessoas e ingressos esperam em cada tipo.
type ResumoFila struct {
	TipoIngressoID int64 `json:"tipo_ingresso_id"`
	Pessoas        int64 `json:"pessoas"`
	Ingressos      int64 `json:"ingressos"`
}

func (s *FilaEsperaService) ResumoDoEvento(eventoID int64) ([]ResumoFila, error) {
	var out []ResumoFila
	err := s.db.Model(&FilaEspera{}).Select("tipo_ingresso_id, COUNT(*) AS pessoas, SUM(quantidade) AS ingressos").
		Where("evento_id = ? AND status IN ?", eventoID, []string{FilaAguardando, FilaConvidado}).
		Group("tipo_ingresso_id").Order("tipo_ingresso_id").Scan(&out).Error
	if out == nil {
		out = []ResumoFila{}
	}
	return out, err
}

// --- A fila anda ---

// PromoverListaEspera é o gancho de "uma vaga voltou" (mesmo do item 3.2).
func (s *FilaEsperaService) PromoverListaEspera(eventoID int64) {
	var tipos []int64
	s.db.Model(&FilaEspera{}).Distinct("tipo_ingresso_id").Where("evento_id = ? AND status = ?", eventoID, FilaAguardando).Pluck("tipo_ingresso_id", &tipos)
	for _, t := range tipos {
		s.promoverTipo(t)
	}
}

// convite é o e-mail a mandar depois que a transação fecha.
type convite struct {
	entrada FilaEspera
	evento  domain.Evento
	tipo    domain.TipoIngresso
}

// promoverTipo guarda as vagas livres para a fila, na ordem de chegada.
// Quem pediu mais do que há livre continua na vez; a vaga vai para o
// próximo que cabe (assim uma vaga solta não fica parada).
func (s *FilaEsperaService) promoverTipo(tipoID int64) int {
	var convites []convite
	err := s.db.Transaction(func(tx *gorm.DB) error {
		t, err := s.itens.BuscarTipoIngressoParaAtualizar(tx, tipoID)
		if err != nil {
			return err
		}
		var e domain.Evento
		if err := tx.First(&e, t.EventoID).Error; err != nil {
			return err
		}
		agora := time.Now()
		if !tipoAVenda(&e, t, agora) {
			return nil
		}
		livres, err := s.livres(tx, t)
		if err != nil || livres <= 0 {
			return err
		}
		var fila []FilaEspera
		if err := tx.Where("tipo_ingresso_id = ? AND status = ?", tipoID, FilaAguardando).Order("criado_em, id").Find(&fila).Error; err != nil {
			return err
		}
		for _, f := range fila {
			if livres <= 0 {
				break
			}
			if int64(f.Quantidade) > livres {
				continue
			}
			prazo := PrazoParaComprar(agora, e.InicioEm, t.VendasFim)
			if err := tx.Model(&FilaEspera{}).Where("id = ? AND status = ?", f.ID, FilaAguardando).
				Updates(map[string]any{"status": FilaConvidado, "convidado_em": agora, "prazo_em": prazo, "atualizado_em": agora}).Error; err != nil {
				return err
			}
			f.Status, f.PrazoEm = FilaConvidado, &prazo
			livres -= int64(f.Quantidade)
			convites = append(convites, convite{entrada: f, evento: e, tipo: *t})
		}
		return nil
	})
	if err != nil {
		slog.Warn("fila de espera: promover", "tipo_id", tipoID, "erro", err)
		return 0
	}
	for _, c := range convites {
		s.avisarVaga(&c)
	}
	return len(convites)
}

// Processar roda no job de expirar reservas (a cada 5 min): encerra convites
// vencidos, acompanha quem reservou e faz a fila andar.
func (s *FilaEsperaService) Processar(agora time.Time) (int, error) {
	var vencidos []FilaEspera
	if err := s.db.Where("status = ? AND prazo_em <= ?", FilaConvidado, agora).Find(&vencidos).Error; err != nil {
		return 0, err
	}
	for _, v := range vencidos {
		res := s.db.Model(&FilaEspera{}).Where("id = ? AND status = ?", v.ID, FilaConvidado).
			Updates(map[string]any{"status": FilaExpirou, "atualizado_em": agora})
		if res.Error == nil && res.RowsAffected == 1 {
			s.avisarPrazoAcabou(&v)
		}
	}
	// Reservou: pagou vira "comprou"; reserva vencida ou cancelada encerra a vez.
	s.db.Exec(`UPDATE fila_espera f SET status = ?, atualizado_em = ? FROM pedidos p
		WHERE f.pedido_id = p.id AND f.status = ? AND p.status = ?`, FilaComprou, agora, FilaReservou, domain.StatusPedidoPago)
	s.db.Exec(`UPDATE fila_espera f SET status = ?, atualizado_em = ? FROM pedidos p
		WHERE f.pedido_id = p.id AND f.status = ? AND p.status IN ?`, FilaExpirou, agora, FilaReservou, statusPedidoEncerrados)

	var tipos []int64
	if err := s.db.Model(&FilaEspera{}).Distinct("tipo_ingresso_id").Where("status = ?", FilaAguardando).Pluck("tipo_ingresso_id", &tipos).Error; err != nil {
		return 0, err
	}
	convidados := 0
	for _, t := range tipos {
		convidados += s.promoverTipo(t)
	}
	return convidados, nil
}

// --- E-mails ---

func (s *FilaEsperaService) destinatario(usuarioID int64) (nome, email string) {
	var u struct{ Nome, Email string }
	s.db.Table("usuarios").Select("nome, email").Where("id = ?", usuarioID).Scan(&u)
	return u.Nome, u.Email
}

// LinkConvite abre o evento com os ingressos já escolhidos. Testada.
func LinkConvite(frontendURL, slug string, tipoID int64, quantidade int) string {
	return frontendURL + "/e/" + slug + "?" + url.Values{"retomar": {strconv.FormatInt(tipoID, 10) + ":" + strconv.Itoa(quantidade)}}.Encode()
}

func (s *FilaEsperaService) avisarVaga(c *convite) {
	nome, email := s.destinatario(c.entrada.UsuarioID)
	if email == "" {
		return
	}
	prazo := c.entrada.PrazoEm.In(FusoDoEvento(&c.evento)).Format("02/01 às 15h04")
	link := LinkConvite(s.frontendURL, c.evento.Slug, c.tipo.ID, c.entrada.Quantidade)
	corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`+
		`<p>Olá, %s! Abriu vaga para você na lista de espera de <strong>%s</strong>.</p>`+
		`<p>Guardamos <strong>%d × %s</strong> só para você até <strong>%s</strong>. Depois disso, a vez passa para a próxima pessoa da fila.</p>`+
		`<p><a href="%s" style="background:#111;color:#fff;padding:10px 16px;border-radius:8px;text-decoration:none">Comprar agora</a></p>`+
		`<p style="color:#777;font-size:12px">Não quer mais? É só ignorar este e-mail ou sair da lista na página do evento.</p></div>`,
		html.EscapeString(primeiroNome(nome)), html.EscapeString(c.evento.Titulo), c.entrada.Quantidade, html.EscapeString(c.tipo.Nome), prazo, link)
	if err := s.mail.Enviar(email, "Abriu vaga: "+c.evento.Titulo+" — compre até "+prazo, corpo); err != nil {
		slog.Warn("fila de espera: e-mail de vaga", "erro", err)
	}
}

func (s *FilaEsperaService) avisarPrazoAcabou(f *FilaEspera) {
	nome, email := s.destinatario(f.UsuarioID)
	var e domain.Evento
	if email == "" || s.db.First(&e, f.EventoID).Error != nil {
		return
	}
	corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`+
		`<p>Olá, %s. O prazo para comprar o ingresso guardado de <strong>%s</strong> acabou, e a vaga foi para a próxima pessoa da fila.</p>`+
		`<p>Se ainda quiser ir, entre de novo na lista de espera: <a href="%s/e/%s">página do evento</a>.</p></div>`,
		html.EscapeString(primeiroNome(nome)), html.EscapeString(e.Titulo), s.frontendURL, e.Slug)
	_ = s.mail.Enviar(email, "O prazo da sua vaga acabou — "+e.Titulo, corpo)
}

// Promotores junta os ganchos de "vaga voltou" (fila da aprovação manual,
// item 3.2, e lista de espera de esgotado, item 7.3).
type Promotores []PromotorListaEspera

func (p Promotores) PromoverListaEspera(eventoID int64) {
	for _, x := range p {
		if x != nil {
			x.PromoverListaEspera(eventoID)
		}
	}
}
