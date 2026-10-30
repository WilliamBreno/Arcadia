package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Item 10.3: pacotes (preço fechado rateado nos itens) e compra em grupo
// (vagas seguradas até cada pessoa pagar a sua parte).

var (
	ErrAdicionalSemIngresso = errors.New("adicionais são vendidos junto com um ingresso")
	ErrPacoteIndisponivel   = errors.New("pacote indisponível")
	ErrGrupoInvalido        = errors.New("grupo de compra inválido")
	ErrGrupoEncerrado       = errors.New("este grupo de compra já fechou")
	ErrGrupoCompleto        = errors.New("todas as partes deste grupo já foram pagas ou estão sendo pagas")
	ErrGrupoJaParticipa     = errors.New("você já tem a sua parte neste grupo")
	ErrDadosPacote          = errors.New("dados do pacote inválidos")
)

const (
	PrazoGrupo         = 24 * time.Hour
	FolgaGrupoEvento   = 2 * time.Hour
	GrupoMin, GrupoMax = 2, 10
)

type Pacote struct {
	ID            int64 `gorm:"primaryKey"`
	EventoID      int64
	Nome          string
	Descricao     string
	PrecoCentavos int64
	Quantidade    int
	MaxPorPedido  int
	Ativo         bool
	CriadoEm      time.Time `gorm:"default:now()"`
}

func (Pacote) TableName() string { return "pacotes" }

type PacoteItem struct {
	PacoteID       int64 `gorm:"primaryKey"`
	TipoIngressoID int64 `gorm:"primaryKey"`
	Quantidade     int
}

func (PacoteItem) TableName() string { return "pacote_itens" }

type GrupoCompra struct {
	ID             int64 `gorm:"primaryKey"`
	EventoID       int64
	TipoIngressoID int64
	CriadorID      int64
	Quantidade     int
	Token          string
	ExpiraEm       time.Time
	Status         string
	CriadoEm       time.Time `gorm:"default:now()"`
}

func (GrupoCompra) TableName() string { return "grupos_compra" }

// ---------- pacote: rateio ----------

type parteRateio struct {
	TipoID     int64
	Quantidade int
	Preco      int64 // preço por unidade nesta parte
}

// RatearPacote divide o preço do pacote entre as unidades, proporcional
// ao preço de tabela (todos zero = partes iguais). O centavo que sobra vai
// numa unidade do primeiro item, que vira uma parte separada. A soma das
// partes é sempre o preço do pacote. Testada.
func RatearPacote(preco int64, itens []PacoteItem, tabela map[int64]int64) []parteRateio {
	var soma, unidades int64
	for _, it := range itens {
		soma += tabela[it.TipoIngressoID] * int64(it.Quantidade)
		unidades += int64(it.Quantidade)
	}
	partes := make([]parteRateio, 0, len(itens)+1)
	var usado int64
	for _, it := range itens {
		var unit int64
		if soma > 0 {
			unit = preco * tabela[it.TipoIngressoID] / soma
		} else if unidades > 0 {
			unit = preco / unidades
		}
		partes = append(partes, parteRateio{TipoID: it.TipoIngressoID, Quantidade: it.Quantidade, Preco: unit})
		usado += unit * int64(it.Quantidade)
	}
	if sobra := preco - usado; sobra != 0 && len(partes) > 0 {
		p := &partes[0]
		if p.Quantidade == 1 {
			p.Preco += sobra
		} else {
			p.Quantidade--
			partes = append(partes, parteRateio{TipoID: p.TipoID, Quantidade: 1, Preco: p.Preco + sobra})
		}
	}
	return partes
}

// expandirPacotes troca cada pacote pedido pelas suas partes (tipo,
// quantidade e preço rateado). Os demais itens passam como estão.
func expandirPacotes(db *gorm.DB, eventoID int64, itensReq []ItemRequisitado) ([]ItemRequisitado, error) {
	out := make([]ItemRequisitado, 0, len(itensReq))
	for _, ir := range itensReq {
		if ir.PacoteID == 0 {
			out = append(out, ir)
			continue
		}
		var p Pacote
		if db.Where("id = ? AND evento_id = ?", ir.PacoteID, eventoID).First(&p).Error != nil || !p.Ativo {
			return nil, ErrPacoteIndisponivel
		}
		if ir.Quantidade < 1 || ir.Quantidade > p.MaxPorPedido {
			return nil, fmt.Errorf("%w: até %d por pedido", ErrQuantidadeInvalida, p.MaxPorPedido)
		}
		if len(ir.LugarIDs) > 0 || ir.GrupoToken != "" {
			return nil, ErrPacoteIndisponivel
		}
		var itens []PacoteItem
		if err := db.Where("pacote_id = ?", p.ID).Order("tipo_ingresso_id").Find(&itens).Error; err != nil || len(itens) == 0 {
			return nil, ErrPacoteIndisponivel
		}
		tabela := map[int64]int64{}
		for _, it := range itens {
			var t domain.TipoIngresso
			if db.First(&t, it.TipoIngressoID).Error != nil {
				return nil, ErrPacoteIndisponivel
			}
			tabela[t.ID] = t.PrecoCentavos
		}
		pid := p.ID
		for _, parte := range RatearPacote(p.PrecoCentavos, itens, tabela) {
			preco := parte.Preco
			out = append(out, ItemRequisitado{
				TipoIngressoID: parte.TipoID, Quantidade: parte.Quantidade * ir.Quantidade,
				GarantiaContratada: ir.GarantiaContratada, precoPacote: &preco, pacoteID: &pid,
				pacotesPedidos: ir.Quantidade,
			})
		}
	}
	return out, nil
}

// conferirEstoquePacotes trava cada pacote e confere quantos ainda há.
func conferirEstoquePacotes(tx *gorm.DB, itensReq []ItemRequisitado) error {
	pedidos := map[int64]int{}
	for _, ir := range itensReq {
		if ir.pacoteID != nil {
			pedidos[*ir.pacoteID] = ir.pacotesPedidos
		}
	}
	for id, qtd := range pedidos {
		var p Pacote
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error; err != nil || !p.Ativo {
			return ErrPacoteIndisponivel
		}
		vendidos, err := PacotesVendidos(tx, id)
		if err != nil {
			return err
		}
		if vendidos+int64(qtd) > int64(p.Quantidade) {
			return fmt.Errorf("%w: %s (restam %d)", ErrEstoqueInsuficiente, p.Nome, max(int64(p.Quantidade)-vendidos, 0))
		}
	}
	return nil
}

// PacotesVendidos: unidades ativas do pacote ÷ unidades por pacote.
func PacotesVendidos(db *gorm.DB, pacoteID int64) (int64, error) {
	var r struct{ Ativos, PorPacote int64 }
	err := db.Raw(`SELECT
		(SELECT count(*) FROM itens_pedido WHERE pacote_id = ? AND status IN ('reservado', 'pago', 'utilizado')) AS ativos,
		(SELECT COALESCE(SUM(quantidade), 0) FROM pacote_itens WHERE pacote_id = ?) AS por_pacote`, pacoteID, pacoteID).Scan(&r).Error
	if err != nil || r.PorPacote == 0 {
		return 0, err
	}
	return (r.Ativos + r.PorPacote - 1) / r.PorPacote, nil
}

// ---------- grupo: vagas seguradas ----------

// vagasDeGrupos: ingressos segurados pelos grupos abertos do tipo (o que
// ainda não virou reserva ou pagamento de alguém do grupo).
func vagasDeGrupos(db *gorm.DB, tipoID int64) int64 {
	var n int64
	db.Raw(`SELECT COALESCE(SUM(GREATEST(g.quantidade - (SELECT count(*) FROM itens_pedido i
			WHERE i.grupo_id = g.id AND i.status IN ('reservado', 'pago', 'utilizado')), 0)), 0)
		FROM grupos_compra g WHERE g.tipo_ingresso_id = ? AND g.status = 'aberto' AND g.expira_em > now()`, tipoID).Scan(&n)
	return n
}

func vagasDeGruposPorTipos(db *gorm.DB, ids []int64) map[int64]int64 {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out
	}
	var linhas []struct {
		TipoIngressoID int64
		Total          int64
	}
	db.Raw(`SELECT g.tipo_ingresso_id, SUM(GREATEST(g.quantidade - (SELECT count(*) FROM itens_pedido i
			WHERE i.grupo_id = g.id AND i.status IN ('reservado', 'pago', 'utilizado')), 0)) AS total
		FROM grupos_compra g WHERE g.tipo_ingresso_id IN ? AND g.status = 'aberto' AND g.expira_em > now()
		GROUP BY g.tipo_ingresso_id`, ids).Scan(&linhas)
	for _, l := range linhas {
		out[l.TipoIngressoID] = l.Total
	}
	return out
}

// restanteDoGrupo: partes que ainda ninguém reservou nem pagou.
func restanteDoGrupo(db *gorm.DB, g *GrupoCompra) int64 {
	var ativos int64
	db.Raw(`SELECT count(*) FROM itens_pedido WHERE grupo_id = ? AND status IN ('reservado', 'pago', 'utilizado')`, g.ID).Scan(&ativos)
	return max(int64(g.Quantidade)-ativos, 0)
}

// grupoParaComprar trava o grupo e confere se a pessoa pode pegar uma parte.
func grupoParaComprar(tx *gorm.DB, token string, eventoID, tipoID, usuarioID int64, quantidade int, agora time.Time) (*GrupoCompra, error) {
	var g GrupoCompra
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token = ?", token).First(&g).Error; err != nil {
		return nil, ErrGrupoInvalido
	}
	if g.EventoID != eventoID || g.TipoIngressoID != tipoID || quantidade != 1 {
		return nil, fmt.Errorf("%w: cada pessoa paga uma parte (1 ingresso)", ErrGrupoInvalido)
	}
	if g.Status != "aberto" || !agora.Before(g.ExpiraEm) {
		return nil, ErrGrupoEncerrado
	}
	var ja int64
	tx.Raw(`SELECT count(*) FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
		WHERE i.grupo_id = ? AND p.usuario_id = ? AND i.status IN ('reservado', 'pago', 'utilizado')`, g.ID, usuarioID).Scan(&ja)
	if ja > 0 {
		return nil, ErrGrupoJaParticipa
	}
	if restanteDoGrupo(tx, &g) < 1 {
		return nil, ErrGrupoCompleto
	}
	return &g, nil
}

func tokenGrupo() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CriarGrupo segura N ingressos de um tipo por até 24 h (ou até 2 h antes
// do evento). Quem cria também paga a sua parte pelo link.
func (s *CheckoutService) CriarGrupo(usuarioID, eventoID, tipoID int64, quantidade int) (*GrupoCompra, error) {
	if quantidade < GrupoMin || quantidade > GrupoMax {
		return nil, fmt.Errorf("%w: de %d a %d pessoas", ErrGrupoInvalido, GrupoMin, GrupoMax)
	}
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || evento.Status != domain.StatusEventoPublicado {
		return nil, ErrEventoNaoDisponivelParaCompra
	}
	if s.plateia != nil && !s.plateia.PodeComprar(evento, usuarioID) {
		return nil, ErrPrecisaAprovacao
	}
	agora := time.Now()
	expira := agora.Add(PrazoGrupo)
	if evento.InicioEm != nil && evento.InicioEm.Add(-FolgaGrupoEvento).Before(expira) {
		expira = evento.InicioEm.Add(-FolgaGrupoEvento)
	}
	if expira.Sub(agora) < 30*time.Minute {
		return nil, fmt.Errorf("%w: o evento está muito perto; comprem direto", ErrGrupoInvalido)
	}
	token, err := tokenGrupo()
	if err != nil {
		return nil, err
	}
	var g *GrupoCompra
	err = s.db.Transaction(func(tx *gorm.DB) error {
		tipo, err := s.itensPedido.BuscarTipoIngressoParaAtualizar(tx, tipoID)
		if err != nil || tipo.EventoID != eventoID || tipo.Interno() || !tipo.Ativo {
			return ErrTipoIndisponivel
		}
		if (tipo.VendasInicio != nil && agora.Before(*tipo.VendasInicio)) || (tipo.VendasFim != nil && agora.After(*tipo.VendasFim)) {
			return fmt.Errorf("%w: %s", ErrVendasEncerradas, tipo.Nome)
		}
		if com, _ := TiposComLugar(tx, []int64{tipo.ID}); com[tipo.ID] {
			return fmt.Errorf("%w: ingresso com lugar marcado não tem compra em grupo", ErrGrupoInvalido)
		}
		if tipo.MaxPorPedido > 0 && quantidade > tipo.MaxPorPedido*GrupoMax {
			return ErrQuantidadeInvalida
		}
		ativos, err := s.itensPedido.ContarAtivosPorTipo(tx, tipo.ID)
		if err != nil {
			return err
		}
		guardadas, err := VagasGuardadas(tx, tipo.ID, 0)
		if err != nil {
			return err
		}
		if livre := int64(tipo.Quantidade) - ativos - guardadas; int64(quantidade) > livre {
			return fmt.Errorf("%w: %s (restam %d)", ErrEstoqueInsuficiente, tipo.Nome, max(livre, 0))
		}
		g = &GrupoCompra{EventoID: eventoID, TipoIngressoID: tipo.ID, CriadorID: usuarioID, Quantidade: quantidade,
			Token: token, ExpiraEm: expira, Status: "aberto"}
		return tx.Create(g).Error
	})
	return g, err
}

type GrupoView struct {
	Token          string     `json:"token"`
	EventoID       int64      `json:"evento_id"`
	EventoTitulo   string     `json:"evento_titulo"`
	EventoSlug     string     `json:"evento_slug"`
	EventoInicioEm *time.Time `json:"evento_inicio_em"`
	TipoID         int64      `json:"tipo_ingresso_id"`
	TipoNome       string     `json:"tipo_nome"`
	PrecoCentavos  int64      `json:"preco_centavos"`
	TotalCentavos  int64      `json:"total_por_pessoa_centavos"`
	Quantidade     int        `json:"quantidade"`
	Pagas          int64      `json:"pagas"`
	EmPagamento    int64      `json:"em_pagamento"`
	Restantes      int64      `json:"restantes"`
	ExpiraEm       time.Time  `json:"expira_em"`
	Status         string     `json:"status"`
	Criador        string     `json:"criador"`
	SouCriador     bool       `json:"sou_criador"`
	MinhaParte     string     `json:"minha_parte,omitempty"` // "", reservado, pago
	MeuPedidoID    *int64     `json:"meu_pedido_id,omitempty"`
}

// VerGrupo: a página do link. Só o primeiro nome de quem criou.
func (s *CheckoutService) VerGrupo(token string, usuarioID int64) (*GrupoView, error) {
	var g GrupoCompra
	if s.db.Where("token = ?", token).First(&g).Error != nil {
		return nil, ErrGrupoInvalido
	}
	return s.grupoView(&g, usuarioID)
}

func (s *CheckoutService) grupoView(g *GrupoCompra, usuarioID int64) (*GrupoView, error) {
	evento, err := s.eventos.BuscarPorID(g.EventoID)
	if err != nil {
		return nil, ErrGrupoInvalido
	}
	var tipo domain.TipoIngresso
	if s.db.First(&tipo, g.TipoIngressoID).Error != nil {
		return nil, ErrGrupoInvalido
	}
	v := &GrupoView{Token: g.Token, EventoID: evento.ID, EventoTitulo: evento.Titulo, EventoSlug: evento.Slug, EventoInicioEm: evento.InicioEm,
		TipoID: tipo.ID, TipoNome: tipo.Nome, PrecoCentavos: tipo.PrecoCentavos, Quantidade: g.Quantidade, ExpiraEm: g.ExpiraEm,
		Status: g.Status, SouCriador: usuarioID > 0 && usuarioID == g.CriadorID}
	if taxas, err := s.taxas.ParaEvento(evento); err == nil {
		v.TotalCentavos = taxas.Calcular(tipo.PrecoCentavos, false).TotalCentavos
	}
	if g.Status == "aberto" && !time.Now().Before(g.ExpiraEm) {
		v.Status = "encerrado"
	}
	var nome string
	s.db.Table("usuarios").Where("id = ?", g.CriadorID).Pluck("nome", &nome)
	v.Criador = primeiroNome(nome)
	var cont struct{ Pagas, Reservadas int64 }
	s.db.Raw(`SELECT count(*) FILTER (WHERE status IN ('pago', 'utilizado')) AS pagas, count(*) FILTER (WHERE status = 'reservado') AS reservadas
		FROM itens_pedido WHERE grupo_id = ?`, g.ID).Scan(&cont)
	v.Pagas, v.EmPagamento = cont.Pagas, cont.Reservadas
	if v.Status == "aberto" {
		v.Restantes = max(int64(g.Quantidade)-cont.Pagas-cont.Reservadas, 0)
	}
	if usuarioID > 0 {
		var minha struct {
			Status   string
			PedidoID int64
		}
		if res := s.db.Raw(`SELECT i.status, i.pedido_id FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
			WHERE i.grupo_id = ? AND p.usuario_id = ? AND i.status IN ('reservado', 'pago', 'utilizado') LIMIT 1`, g.ID, usuarioID).Scan(&minha); res.RowsAffected > 0 {
			v.MinhaParte = "pago"
			if minha.Status == "reservado" {
				v.MinhaParte = "reservado"
			}
			v.MeuPedidoID = &minha.PedidoID
		}
	}
	return v, nil
}

// MeusGrupos: grupos que a pessoa criou.
func (s *CheckoutService) MeusGrupos(usuarioID int64) ([]GrupoView, error) {
	var grupos []GrupoCompra
	if err := s.db.Where("criador_id = ?", usuarioID).Order("criado_em DESC").Limit(50).Find(&grupos).Error; err != nil {
		return nil, err
	}
	out := make([]GrupoView, 0, len(grupos))
	for i := range grupos {
		if v, err := s.grupoView(&grupos[i], usuarioID); err == nil {
			out = append(out, *v)
		}
	}
	return out, nil
}

// CancelarGrupo: quem criou solta as partes ainda não pagas (quem pagou
// fica com o ingresso).
func (s *CheckoutService) CancelarGrupo(usuarioID int64, token string) error {
	res := s.db.Model(&GrupoCompra{}).Where("token = ? AND criador_id = ? AND status = 'aberto'", token, usuarioID).Update("status", "cancelado")
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrGrupoInvalido
	}
	return res.Error
}

// ---------- pacotes: organizador ----------

type DadosPacote struct {
	Nome          string
	Descricao     string
	PrecoCentavos int64
	Quantidade    int
	MaxPorPedido  int
	Ativo         bool
	Itens         []PacoteItem
}

func validarPacote(tx *gorm.DB, eventoID int64, d *DadosPacote) error {
	d.Nome = strings.TrimSpace(d.Nome)
	d.Descricao = strings.TrimSpace(d.Descricao)
	if d.Nome == "" || utf8.RuneCountInString(d.Nome) > 80 || utf8.RuneCountInString(d.Descricao) > 300 {
		return fmt.Errorf("%w: nome até 80 letras e descrição até 300", ErrDadosPacote)
	}
	if d.PrecoCentavos <= 0 || d.Quantidade <= 0 || d.MaxPorPedido < 1 || d.MaxPorPedido > 20 {
		return fmt.Errorf("%w: preço e quantidade maiores que zero; até 20 por pedido", ErrDadosPacote)
	}
	if len(d.Itens) == 0 || len(d.Itens) > 10 {
		return fmt.Errorf("%w: de 1 a 10 itens no pacote", ErrDadosPacote)
	}
	vistos := map[int64]bool{}
	temIngresso := false
	var tabela int64
	for _, it := range d.Itens {
		if vistos[it.TipoIngressoID] || it.Quantidade < 1 || it.Quantidade > 20 {
			return fmt.Errorf("%w: itens repetidos ou quantidade fora de 1 a 20", ErrDadosPacote)
		}
		vistos[it.TipoIngressoID] = true
		var t domain.TipoIngresso
		if tx.Where("id = ? AND evento_id = ? AND NOT inscricao_talento AND NOT espaco_expositor", it.TipoIngressoID, eventoID).First(&t).Error != nil {
			return fmt.Errorf("%w: item não encontrado neste evento", ErrDadosPacote)
		}
		if com, _ := TiposComLugar(tx, []int64{t.ID}); com[t.ID] {
			return fmt.Errorf("%w: ingresso com lugar marcado não entra em pacote", ErrDadosPacote)
		}
		temIngresso = temIngresso || !t.Adicional
		tabela += t.PrecoCentavos * int64(it.Quantidade)
	}
	if !temIngresso {
		return fmt.Errorf("%w: o pacote precisa ter pelo menos um ingresso", ErrDadosPacote)
	}
	if tabela > 0 && d.PrecoCentavos > tabela {
		return fmt.Errorf("%w: o pacote não pode custar mais que os itens separados", ErrDadosPacote)
	}
	return nil
}

func (s *CheckoutService) eventoDoOrganizador(organizadorID, eventoID int64) error {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != organizadorID {
		return ErrEventoNaoPertenceAoOrganizador
	}
	return nil
}

func (s *CheckoutService) SalvarPacote(organizadorID, eventoID, pacoteID int64, d DadosPacote) (int64, error) {
	if err := s.eventoDoOrganizador(organizadorID, eventoID); err != nil {
		return 0, err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := validarPacote(tx, eventoID, &d); err != nil {
			return err
		}
		p := Pacote{ID: pacoteID, EventoID: eventoID, Nome: d.Nome, Descricao: d.Descricao, PrecoCentavos: d.PrecoCentavos,
			Quantidade: d.Quantidade, MaxPorPedido: d.MaxPorPedido, Ativo: d.Ativo}
		if pacoteID > 0 {
			var atual Pacote
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND evento_id = ?", pacoteID, eventoID).First(&atual).Error != nil {
				return ErrPacoteIndisponivel
			}
			vendidos, _ := PacotesVendidos(tx, pacoteID)
			if vendidos > 0 {
				// Vendido: só nome, descrição, estoque, limite e ativo mudam.
				p.PrecoCentavos = atual.PrecoCentavos
				p.CriadoEm = atual.CriadoEm
				if int64(p.Quantidade) < vendidos {
					return fmt.Errorf("%w: já foram vendidos %d", ErrDadosPacote, vendidos)
				}
				return tx.Model(&atual).Updates(map[string]any{"nome": p.Nome, "descricao": p.Descricao, "quantidade": p.Quantidade,
					"max_por_pedido": p.MaxPorPedido, "ativo": p.Ativo}).Error
			}
			p.CriadoEm = atual.CriadoEm
			if err := tx.Save(&p).Error; err != nil {
				return err
			}
			if err := tx.Where("pacote_id = ?", pacoteID).Delete(&PacoteItem{}).Error; err != nil {
				return err
			}
		} else if err := tx.Create(&p).Error; err != nil {
			return err
		}
		pacoteID = p.ID
		for _, it := range d.Itens {
			if err := tx.Create(&PacoteItem{PacoteID: p.ID, TipoIngressoID: it.TipoIngressoID, Quantidade: it.Quantidade}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return pacoteID, err
}

func (s *CheckoutService) ExcluirPacote(organizadorID, eventoID, pacoteID int64) error {
	if err := s.eventoDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	if v, _ := PacotesVendidos(s.db, pacoteID); v > 0 {
		return fmt.Errorf("%w: o pacote já tem vendas; desative em vez de apagar", ErrDadosPacote)
	}
	res := s.db.Where("id = ? AND evento_id = ?", pacoteID, eventoID).Delete(&Pacote{})
	if res.Error == nil && res.RowsAffected == 0 {
		return ErrPacoteIndisponivel
	}
	return res.Error
}

type PacoteView struct {
	ID             int64            `json:"id"`
	Nome           string           `json:"nome"`
	Descricao      string           `json:"descricao"`
	PrecoCentavos  int64            `json:"preco_centavos"`
	TabelaCentavos int64            `json:"tabela_centavos"`
	Quantidade     int              `json:"quantidade"`
	Vendidos       int64            `json:"vendidos"`
	MaxPorPedido   int              `json:"max_por_pedido"`
	Ativo          bool             `json:"ativo"`
	Itens          []PacoteItemView `json:"itens"`
}

type PacoteItemView struct {
	TipoIngressoID int64  `json:"tipo_ingresso_id"`
	Nome           string `json:"nome"`
	Quantidade     int    `json:"quantidade"`
	Adicional      bool   `json:"adicional"`
}

// Pacotes do evento (só ativos, para a página pública).
func (s *CheckoutService) Pacotes(eventoID int64, soAtivos bool) ([]PacoteView, error) {
	var ps []Pacote
	q := s.db.Where("evento_id = ?", eventoID).Order("preco_centavos, id")
	if soAtivos {
		q = q.Where("ativo")
	}
	if err := q.Find(&ps).Error; err != nil {
		return nil, err
	}
	out := make([]PacoteView, 0, len(ps))
	for _, p := range ps {
		v := PacoteView{ID: p.ID, Nome: p.Nome, Descricao: p.Descricao, PrecoCentavos: p.PrecoCentavos, Quantidade: p.Quantidade,
			MaxPorPedido: p.MaxPorPedido, Ativo: p.Ativo, Itens: []PacoteItemView{}}
		v.Vendidos, _ = PacotesVendidos(s.db, p.ID)
		s.db.Raw(`SELECT pi.tipo_ingresso_id, t.nome, pi.quantidade, t.adicional FROM pacote_itens pi
			JOIN tipos_ingresso t ON t.id = pi.tipo_ingresso_id WHERE pi.pacote_id = ? ORDER BY t.adicional, t.ordem, t.id`, p.ID).Scan(&v.Itens)
		s.db.Raw(`SELECT COALESCE(SUM(t.preco_centavos * pi.quantidade), 0) FROM pacote_itens pi
			JOIN tipos_ingresso t ON t.id = pi.tipo_ingresso_id WHERE pi.pacote_id = ?`, p.ID).Scan(&v.TabelaCentavos)
		out = append(out, v)
	}
	return out, nil
}
