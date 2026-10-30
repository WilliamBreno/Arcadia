package service

// Item 8.5: circuitos com ranking por temporada e seletivas. Decisões do
// dono (30/10/2026):
//   - qualquer organizador cria um circuito (com temporada) e aceita ou
//     recusa os eventos que outros organizadores pedem para incluir;
//   - pontos fixos do 1º ao 10º de cada categoria (25-18-15-12-10-8-6-4-2-1),
//     contados quando o organizador libera o resultado; desempate: mais
//     vitórias, depois mais 2º lugares;
//   - menor aparece só pelo primeiro nome (ou nome artístico), sem link;
//   - seletiva: os N primeiros de cada categoria entram aprovados na final
//     (e pagam a inscrição da final, se houver).

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrCircuitoNaoEncontrado = errors.New("circuito não encontrado")
	ErrCircuitoInvalido      = errors.New("dados do circuito inválidos")
	ErrEventoJaEmCircuito    = errors.New("este evento já está (ou pediu para entrar) em um circuito")
	ErrForaDaTemporada       = errors.New("a data do evento está fora da temporada do circuito")
	ErrSeletivaInvalida      = errors.New("seletiva inválida")
	ErrPedidoNaoEncontrado8  = errors.New("pedido não encontrado")
)

// PontosPorPosicao: 1º 25 ... 10º 1.
var PontosPorPosicao = []int{25, 18, 15, 12, 10, 8, 6, 4, 2, 1}

// Pontos de uma colocação (0 fora do top 10).
func Pontos(posicao int) int {
	if posicao < 1 || posicao > len(PontosPorPosicao) {
		return 0
	}
	return PontosPorPosicao[posicao-1]
}

type Circuito struct {
	ID              int64 `gorm:"primaryKey"`
	OrganizadorID   int64
	Nome            string
	Slug            string
	Descricao       string
	TemporadaInicio time.Time
	TemporadaFim    time.Time
	CriadoEm        time.Time
}

func (Circuito) TableName() string { return "circuitos" }

type CircuitoEvento struct {
	ID         int64 `gorm:"primaryKey"`
	CircuitoID int64
	EventoID   int64
	Status     string
	CriadoEm   time.Time
	DecididoEm *time.Time
}

func (CircuitoEvento) TableName() string { return "circuito_eventos" }

type ResultadoEvento struct {
	ID          int64 `gorm:"primaryKey"`
	EventoID    int64
	FichaID     int64
	UsuarioID   int64
	Tipo        string
	Posicao     int
	NomePublico string
	Menor       bool
	CriadoEm    time.Time
}

func (ResultadoEvento) TableName() string { return "resultados_evento" }

type Seletiva struct {
	ID               int64 `gorm:"primaryKey"`
	FinalEventoID    int64
	SeletivaEventoID int64
	Vagas            int
	Status           string
	CriadoEm         time.Time
}

func (Seletiva) TableName() string { return "seletivas" }

type CircuitoService struct {
	db        *gorm.DB
	avaliacao *AvaliacaoService
	fichas    *FichaService
	eventos   *repository.EventoRepository
	perfis    *PerfilTalentoService
}

func NovoCircuitoService(db *gorm.DB, a *AvaliacaoService, f *FichaService, e *repository.EventoRepository, p *PerfilTalentoService) *CircuitoService {
	return &CircuitoService{db: db, avaliacao: a, fichas: f, eventos: e, perfis: p}
}

func (s *CircuitoService) doOrganizador(organizadorID, eventoID int64) (*domain.Evento, error) {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return e, nil
}

// ---------- resultado gravado ----------

// GravarResultado regrava o resultado do evento (chamado ao liberar) e
// classifica os primeiros nas finais das seletivas.
func (s *CircuitoService) GravarResultado(eventoID int64) {
	if err := s.gravar(eventoID); err != nil {
		slog.Error("gravar resultado do evento", "evento_id", eventoID, "erro", err)
		return
	}
	s.Classificar(eventoID)
}

func (s *CircuitoService) gravar(eventoID int64) error {
	ranking, err := s.avaliacao.calcularRanking(eventoID)
	if err != nil {
		return err
	}
	var fichaIDs []int64
	for _, ls := range ranking {
		for _, l := range ls {
			fichaIDs = append(fichaIDs, l.FichaID)
		}
	}
	donos := map[int64]int64{}
	if len(fichaIDs) > 0 {
		var fs []domain.FichaParticipacao
		if err := s.db.Select("id, usuario_id").Where("id IN ?", fichaIDs).Find(&fs).Error; err != nil {
			return err
		}
		for _, f := range fs {
			donos[f.ID] = f.UsuarioID
		}
	}
	agora := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("evento_id = ?", eventoID).Delete(&ResultadoEvento{}).Error; err != nil {
			return err
		}
		var linhas []ResultadoEvento
		for tipo, ls := range ranking {
			for _, l := range ls {
				if l.Posicao < 1 {
					continue
				}
				linhas = append(linhas, ResultadoEvento{EventoID: eventoID, FichaID: l.FichaID, UsuarioID: donos[l.FichaID],
					Tipo: tipo, Posicao: l.Posicao, NomePublico: NomePublico(l), Menor: l.Menor, CriadoEm: agora})
			}
		}
		if len(linhas) == 0 {
			return nil
		}
		return tx.Create(&linhas).Error
	})
}

// garantirGravados grava o resultado de eventos liberados antes do 8.5.
func (s *CircuitoService) garantirGravados(eventoIDs []int64) {
	for _, id := range eventoIDs {
		var n int64
		s.db.Model(&ResultadoEvento{}).Where("evento_id = ?", id).Count(&n)
		if n == 0 {
			_ = s.gravar(id)
		}
	}
}

// ---------- circuitos ----------

type DadosCircuito struct {
	Nome            string
	Descricao       string
	TemporadaInicio time.Time
	TemporadaFim    time.Time
}

func (s *CircuitoService) Criar(organizadorID int64, d DadosCircuito) (*Circuito, error) {
	d.Nome = strings.TrimSpace(d.Nome)
	if len([]rune(d.Nome)) < 3 || len([]rune(d.Nome)) > 120 || d.TemporadaFim.Before(d.TemporadaInicio) ||
		d.TemporadaFim.Sub(d.TemporadaInicio) > 2*366*24*time.Hour {
		return nil, ErrCircuitoInvalido
	}
	base := Slugify(d.Nome)
	if base == "" {
		base = "circuito"
	}
	slug := base
	for i := 2; ; i++ {
		var n int64
		s.db.Model(&Circuito{}).Where("slug = ?", slug).Count(&n)
		if n == 0 {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	c := &Circuito{OrganizadorID: organizadorID, Nome: d.Nome, Slug: slug, Descricao: strings.TrimSpace(d.Descricao),
		TemporadaInicio: d.TemporadaInicio, TemporadaFim: d.TemporadaFim, CriadoEm: time.Now()}
	return c, s.db.Create(c).Error
}

// PedidoCircuito é um evento pedindo (ou já aceito) para entrar no circuito.
type PedidoCircuito struct {
	ID           int64
	EventoID     int64
	EventoTitulo string
	EventoSlug   string
	InicioEm     *time.Time
	Status       string
}

type MeuCircuito struct {
	Circuito
	Pedidos []PedidoCircuito
}

func (s *CircuitoService) pedidosDe(circuitoID int64) ([]PedidoCircuito, error) {
	var ps []PedidoCircuito
	err := s.db.Table("circuito_eventos ce").
		Select("ce.id, ce.evento_id, e.titulo AS evento_titulo, e.slug AS evento_slug, e.inicio_em, ce.status").
		Joins("JOIN eventos e ON e.id = ce.evento_id").
		Where("ce.circuito_id = ? AND ce.status <> 'recusado'", circuitoID).
		Order("e.inicio_em").Scan(&ps).Error
	return ps, err
}

func (s *CircuitoService) Meus(organizadorID int64) ([]MeuCircuito, error) {
	var cs []Circuito
	if err := s.db.Where("organizador_id = ?", organizadorID).Order("temporada_inicio DESC").Find(&cs).Error; err != nil {
		return nil, err
	}
	out := make([]MeuCircuito, 0, len(cs))
	for _, c := range cs {
		ps, err := s.pedidosDe(c.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, MeuCircuito{Circuito: c, Pedidos: ps})
	}
	return out, nil
}

// Abertos: circuitos com a temporada ainda valendo (para escolher onde entrar).
func (s *CircuitoService) Abertos() ([]Circuito, error) {
	var cs []Circuito
	err := s.db.Where("temporada_fim >= CURRENT_DATE").Order("nome").Limit(200).Find(&cs).Error
	return cs, err
}

func dentroDaTemporada(c *Circuito, inicio *time.Time, e *domain.Evento) bool {
	if inicio == nil {
		return false
	}
	dia := inicio.In(FusoDoEvento(e)).Format("2006-01-02")
	return dia >= c.TemporadaInicio.Format("2006-01-02") && dia <= c.TemporadaFim.Format("2006-01-02")
}

// PedirEntrada coloca o evento no circuito (aceito na hora se o circuito é
// do mesmo organizador).
func (s *CircuitoService) PedirEntrada(organizadorID, eventoID, circuitoID int64) (*CircuitoEvento, error) {
	e, err := s.doOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	var c Circuito
	if err := s.db.First(&c, circuitoID).Error; err != nil {
		return nil, ErrCircuitoNaoEncontrado
	}
	if !dentroDaTemporada(&c, e.InicioEm, e) {
		return nil, ErrForaDaTemporada
	}
	ce := &CircuitoEvento{CircuitoID: c.ID, EventoID: e.ID, Status: "pendente", CriadoEm: time.Now()}
	if c.OrganizadorID == organizadorID {
		ce.Status, ce.DecididoEm = "aceito", &ce.CriadoEm
	}
	if err := s.db.Create(ce).Error; err != nil {
		if strings.Contains(err.Error(), "circuito_eventos_evento_un") {
			return nil, ErrEventoJaEmCircuito
		}
		return nil, err
	}
	return ce, nil
}

// SairDoCircuito tira o evento (os pontos dele deixam de contar).
func (s *CircuitoService) SairDoCircuito(organizadorID, eventoID int64) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return ErrEventoNaoPertenceAoOrganizador
	}
	return s.db.Where("evento_id = ? AND status <> 'recusado'", eventoID).Delete(&CircuitoEvento{}).Error
}

// Decidir: o dono do circuito aceita ou recusa um evento.
func (s *CircuitoService) Decidir(organizadorID, pedidoID int64, aceitar bool) error {
	var ce CircuitoEvento
	if err := s.db.First(&ce, pedidoID).Error; err != nil {
		return ErrPedidoNaoEncontrado8
	}
	var c Circuito
	if err := s.db.First(&c, ce.CircuitoID).Error; err != nil || c.OrganizadorID != organizadorID {
		return ErrPedidoNaoEncontrado8
	}
	status := "recusado"
	if aceitar {
		status = "aceito"
	}
	agora := time.Now()
	return s.db.Model(&ce).Updates(map[string]any{"status": status, "decidido_em": agora}).Error
}

// CircuitoDoEvento: em que circuito o evento está (aceito ou pendente).
type CircuitoDoEvento struct {
	Circuito
	Status string
}

func (s *CircuitoService) DoEvento(eventoID int64) (*CircuitoDoEvento, error) {
	var ce CircuitoEvento
	if err := s.db.Where("evento_id = ? AND status <> 'recusado'", eventoID).First(&ce).Error; err != nil {
		return nil, err
	}
	var c Circuito
	if err := s.db.First(&c, ce.CircuitoID).Error; err != nil {
		return nil, err
	}
	return &CircuitoDoEvento{Circuito: c, Status: ce.Status}, nil
}

// ---------- ranking ----------

type LinhaCircuito struct {
	Posicao    int
	UsuarioID  int64
	Nome       string
	Menor      bool
	Pontos     int
	Vitorias   int
	Segundos   int
	Eventos    int
	PerfilSlug string
}

type EtapaCircuito struct {
	Titulo   string
	Slug     string
	InicioEm *time.Time
	Liberado bool
}

type CircuitoPublico struct {
	Circuito Circuito
	Etapas   []EtapaCircuito
	Ranking  map[string][]LinhaCircuito
}

// Publico monta a página do circuito: etapas e ranking por categoria.
func (s *CircuitoService) Publico(slug string) (*CircuitoPublico, error) {
	var c Circuito
	if err := s.db.Where("slug = ?", slug).First(&c).Error; err != nil {
		return nil, ErrCircuitoNaoEncontrado
	}
	var etapas []struct {
		ID                  int64
		Titulo              string
		Slug                string
		InicioEm            *time.Time
		Timezone            string
		ResultadoLiberadoEm *time.Time
	}
	if err := s.db.Table("eventos e").
		Select("e.id, e.titulo, e.slug, e.inicio_em, e.timezone, e.resultado_liberado_em").
		Joins("JOIN circuito_eventos ce ON ce.evento_id = e.id AND ce.status = 'aceito'").
		Where("ce.circuito_id = ? AND e.status IN ?", c.ID, []domain.StatusEvento{domain.StatusEventoPublicado, domain.StatusEventoEncerrado}).
		Order("e.inicio_em").Scan(&etapas).Error; err != nil {
		return nil, err
	}
	out := &CircuitoPublico{Circuito: c, Ranking: map[string][]LinhaCircuito{}}
	var contam []int64
	for _, e := range etapas {
		out.Etapas = append(out.Etapas, EtapaCircuito{Titulo: e.Titulo, Slug: e.Slug, InicioEm: e.InicioEm, Liberado: e.ResultadoLiberadoEm != nil})
		ev := &domain.Evento{Timezone: e.Timezone}
		if e.ResultadoLiberadoEm != nil && dentroDaTemporada(&c, e.InicioEm, ev) {
			contam = append(contam, e.ID)
		}
	}
	if len(contam) == 0 {
		return out, nil
	}
	s.garantirGravados(contam)
	var rs []struct {
		ResultadoEvento
		InicioEm *time.Time
	}
	if err := s.db.Table("resultados_evento r").Select("r.*, e.inicio_em").
		Joins("JOIN eventos e ON e.id = r.evento_id").
		Where("r.evento_id IN ?", contam).Order("e.inicio_em, r.id").Scan(&rs).Error; err != nil {
		return nil, err
	}
	linhas := make([]ResultadoEvento, len(rs))
	for i := range rs {
		linhas[i] = rs[i].ResultadoEvento
	}
	out.Ranking = SomarCircuito(linhas)
	// Link do perfil: só adulto com perfil visível.
	var ids []int64
	for _, ls := range out.Ranking {
		for _, l := range ls {
			if !l.Menor {
				ids = append(ids, l.UsuarioID)
			}
		}
	}
	if s.perfis != nil {
		slugs := s.perfis.SlugsVisiveis(ids)
		for tipo, ls := range out.Ranking {
			for i := range ls {
				if !ls[i].Menor {
					ls[i].PerfilSlug = slugs[ls[i].UsuarioID]
				}
			}
			out.Ranking[tipo] = ls
		}
	}
	return out, nil
}

// SomarCircuito soma os pontos por pessoa e categoria. As linhas vêm em
// ordem cronológica: o nome exibido é o da etapa mais recente. Testada.
func SomarCircuito(linhas []ResultadoEvento) map[string][]LinhaCircuito {
	type chave struct {
		tipo    string
		usuario int64
	}
	acc := map[chave]*LinhaCircuito{}
	for _, r := range linhas {
		k := chave{r.Tipo, r.UsuarioID}
		l, ok := acc[k]
		if !ok {
			l = &LinhaCircuito{UsuarioID: r.UsuarioID}
			acc[k] = l
		}
		l.Nome, l.Menor = r.NomePublico, r.Menor
		l.Eventos++
		l.Pontos += Pontos(r.Posicao)
		if r.Posicao == 1 {
			l.Vitorias++
		}
		if r.Posicao == 2 {
			l.Segundos++
		}
	}
	out := map[string][]LinhaCircuito{}
	for k, l := range acc {
		if l.Pontos > 0 {
			out[k.tipo] = append(out[k.tipo], *l)
		}
	}
	for tipo, ls := range out {
		sort.SliceStable(ls, func(i, j int) bool {
			a, b := ls[i], ls[j]
			if a.Pontos != b.Pontos {
				return a.Pontos > b.Pontos
			}
			if a.Vitorias != b.Vitorias {
				return a.Vitorias > b.Vitorias
			}
			if a.Segundos != b.Segundos {
				return a.Segundos > b.Segundos
			}
			return a.Nome < b.Nome
		})
		for i := range ls {
			if i > 0 && ls[i].Pontos == ls[i-1].Pontos && ls[i].Vitorias == ls[i-1].Vitorias && ls[i].Segundos == ls[i-1].Segundos {
				ls[i].Posicao = ls[i-1].Posicao
			} else {
				ls[i].Posicao = i + 1
			}
		}
		out[tipo] = ls
	}
	return out
}

// ---------- seletivas ----------

type SeletivaInfo struct {
	ID       int64
	Vagas    int
	Status   string
	EventoID int64 // o "outro" evento (a final, vista da seletiva; a seletiva, vista da final)
	Titulo   string
	Slug     string
	InicioEm *time.Time
}

// CriarSeletiva: o dono da final marca um evento (pelo endereço) como seletiva.
func (s *CircuitoService) CriarSeletiva(organizadorID, finalID int64, seletivaSlug string, vagas int) (*Seletiva, error) {
	final, err := s.doOrganizador(organizadorID, finalID)
	if err != nil {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	if vagas < 1 || vagas > 10 {
		return nil, fmt.Errorf("%w: de 1 a 10 vagas por categoria", ErrSeletivaInvalida)
	}
	sel, err := s.eventos.BuscarPorSlug(strings.TrimSpace(seletivaSlug))
	if err != nil || sel.ID == final.ID {
		return nil, fmt.Errorf("%w: evento não encontrado", ErrSeletivaInvalida)
	}
	if sel.InicioEm == nil || final.InicioEm == nil || !sel.InicioEm.Before(*final.InicioEm) {
		return nil, fmt.Errorf("%w: a seletiva precisa acontecer antes da final", ErrSeletivaInvalida)
	}
	if sel.ModoParticipantes == domain.ModoParticipantesNenhum {
		return nil, fmt.Errorf("%w: a seletiva precisa ter concurso", ErrSeletivaInvalida)
	}
	st := &Seletiva{FinalEventoID: final.ID, SeletivaEventoID: sel.ID, Vagas: vagas, Status: "pendente", CriadoEm: time.Now()}
	if sel.OrganizadorID == organizadorID {
		st.Status = "aceita"
	}
	if err := s.db.Create(st).Error; err != nil {
		if strings.Contains(err.Error(), "seletivas_final_evento_id_seletiva_evento_id_key") {
			return nil, fmt.Errorf("%w: este evento já é seletiva desta final", ErrSeletivaInvalida)
		}
		return nil, err
	}
	if st.Status == "aceita" {
		s.Classificar(sel.ID)
	}
	return st, nil
}

// RemoverSeletiva: o dono da final desfaz (quem já se classificou continua).
func (s *CircuitoService) RemoverSeletiva(organizadorID, finalID, seletivaID int64) error {
	if _, err := s.doOrganizador(organizadorID, finalID); err != nil {
		return ErrEventoNaoPertenceAoOrganizador
	}
	res := s.db.Where("id = ? AND final_evento_id = ?", seletivaID, finalID).Delete(&Seletiva{})
	if res.RowsAffected == 0 {
		return ErrPedidoNaoEncontrado8
	}
	return res.Error
}

// DecidirSeletiva: o dono do evento seletiva aceita ou recusa.
func (s *CircuitoService) DecidirSeletiva(organizadorID, eventoID, seletivaID int64, aceitar bool) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return ErrEventoNaoPertenceAoOrganizador
	}
	status := "recusada"
	if aceitar {
		status = "aceita"
	}
	res := s.db.Model(&Seletiva{}).Where("id = ? AND seletiva_evento_id = ? AND status = 'pendente'", seletivaID, eventoID).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPedidoNaoEncontrado8
	}
	if aceitar {
		s.Classificar(eventoID)
	}
	return nil
}

func (s *CircuitoService) listarSeletivas(campo, outro string, eventoID int64) ([]SeletivaInfo, error) {
	var out []SeletivaInfo
	err := s.db.Table("seletivas s").
		Select("s.id, s.vagas, s.status, e.id AS evento_id, e.titulo, e.slug, e.inicio_em").
		Joins("JOIN eventos e ON e.id = s."+outro).
		Where("s."+campo+" = ? AND s.status <> 'recusada'", eventoID).
		Order("e.inicio_em").Scan(&out).Error
	return out, err
}

// DaFinal: seletivas que levam a esta final.
func (s *CircuitoService) DaFinal(eventoID int64) ([]SeletivaInfo, error) {
	return s.listarSeletivas("final_evento_id", "seletiva_evento_id", eventoID)
}

// ComoSeletiva: finais para as quais este evento classifica.
func (s *CircuitoService) ComoSeletiva(eventoID int64) ([]SeletivaInfo, error) {
	return s.listarSeletivas("seletiva_evento_id", "final_evento_id", eventoID)
}

// Classificar leva os N primeiros de cada categoria da seletiva para as
// finais (seletiva aceita, resultado liberado, final ainda não começou).
// Idempotente: quem já tem ficha na final não ganha outra.
func (s *CircuitoService) Classificar(seletivaEventoID int64) int {
	sel, err := s.eventos.BuscarPorID(seletivaEventoID)
	if err != nil || sel.ResultadoLiberadoEm == nil {
		return 0
	}
	var sels []Seletiva
	if err := s.db.Where("seletiva_evento_id = ? AND status = 'aceita'", seletivaEventoID).Find(&sels).Error; err != nil || len(sels) == 0 {
		return 0
	}
	s.garantirGravados([]int64{seletivaEventoID})
	total := 0
	agora := time.Now()
	for _, st := range sels {
		final, err := s.eventos.BuscarPorID(st.FinalEventoID)
		if err != nil || final.Status != domain.StatusEventoPublicado || (final.InicioEm != nil && !agora.Before(*final.InicioEm)) {
			continue
		}
		var rs []ResultadoEvento
		if err := s.db.Where("evento_id = ? AND posicao <= ?", seletivaEventoID, st.Vagas).Order("tipo, posicao").Find(&rs).Error; err != nil {
			continue
		}
		for _, r := range rs {
			id, nova, err := s.fichaNaFinal(final.ID, seletivaEventoID, r.FichaID)
			if err != nil {
				slog.Error("classificar na final", "final", final.ID, "ficha", r.FichaID, "erro", err)
				continue
			}
			if nova {
				if _, err := s.fichas.AprovarPeloSistema(id); err != nil {
					slog.Error("aprovar classificado", "ficha", id, "erro", err)
					continue
				}
				total++
			}
		}
	}
	return total
}

// fichaNaFinal cria a ficha do classificado na final copiando a da
// seletiva. Se a pessoa já tinha ficha pendente/lista de espera, aproveita
// essa; se já está aprovada, paga, rejeitada ou desistiu, não mexe.
func (s *CircuitoService) fichaNaFinal(finalID, seletivaID, fichaOrigemID int64) (int64, bool, error) {
	var origem domain.FichaParticipacao
	if err := s.db.First(&origem, fichaOrigemID).Error; err != nil {
		return 0, false, err
	}
	var id int64
	var nova bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var atual domain.FichaParticipacao
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("evento_id = ? AND usuario_id = ? AND papel = ?", finalID, origem.UsuarioID, domain.PapelParticipante).First(&atual).Error
		if err == nil {
			if atual.Status != domain.StatusFichaPendente && atual.Status != domain.StatusFichaListaEspera {
				return nil
			}
			id, nova = atual.ID, true
			return tx.Model(&atual).Update("classificada_por_evento_id", seletivaID).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		f := origem
		f.ID = 0
		f.EventoID = finalID
		f.Origem = domain.OrigemInscricao
		f.Status = domain.StatusFichaPendente
		f.MotivoRejeicao = ""
		f.OrdemApresentacao = nil
		f.PrazoPagamentoEm = nil
		f.ClassificadaPorEventoID = &seletivaID
		f.CriadoEm = time.Now()
		if err := tx.Create(&f).Error; err != nil {
			return err
		}
		id, nova = f.ID, true
		return nil
	})
	return id, nova, err
}
