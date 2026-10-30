package service

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Avaliação do evento pelo público (item 6.5). Só quem comprou e fez
// check-in avalia; a média aparece a partir de 3 avaliações.

var ErrNaoPodeAvaliar = errors.New("não é possível avaliar este evento")

const (
	MinimoParaMedia    = 3
	DiasParaAvaliar    = 30
	MaxComentarioChars = 500
)

type AvaliacaoPublico struct {
	ID           int64 `gorm:"primaryKey"`
	EventoID     int64
	UsuarioID    int64
	Nota         int
	Comentario   string
	Ocultada     bool
	CriadoEm     time.Time
	AtualizadoEm time.Time
}

func (AvaliacaoPublico) TableName() string { return "avaliacoes_publico" }

// fimParaAvaliar: até 30 dias depois do fim (sem fim: depois do início).
func fimParaAvaliar(e *domain.Evento) time.Time {
	ref := *e.InicioEm
	if e.FimEm != nil {
		ref = *e.FimEm
	}
	return ref.AddDate(0, 0, DiasParaAvaliar)
}

// janelaAvaliacao: aberta do início do evento até 30 dias depois do fim.
// Devolve o motivo quando fechada. Função pura, testada.
func janelaAvaliacao(e *domain.Evento, agora time.Time) (bool, string) {
	if e.InicioEm == nil || agora.Before(*e.InicioEm) {
		return false, "a avaliação abre quando o evento começa"
	}
	if agora.After(fimParaAvaliar(e)) {
		return false, "o prazo para avaliar este evento terminou"
	}
	return true, ""
}

// NomeAvaliador: "Maria S." — nunca o nome completo em público. Testado.
func NomeAvaliador(nome string) string {
	p := strings.Fields(nome)
	switch len(p) {
	case 0:
		return "Participante"
	case 1:
		return p[0]
	}
	inicial, _ := utf8.DecodeRuneInString(p[len(p)-1])
	return p[0] + " " + strings.ToUpper(string(inicial)) + "."
}

type AvaliacaoPublicoService struct {
	db          *gorm.DB
	mail        *mail.Cliente
	frontendURL string
	posEvento   *PosEventoService // item 9.7
}

// DefinirPosEvento liga o e-mail pós-evento completo (item 9.7).
func (s *AvaliacaoPublicoService) DefinirPosEvento(p *PosEventoService) { s.posEvento = p }

func NovoAvaliacaoPublicoService(db *gorm.DB, m *mail.Cliente, frontendURL string) *AvaliacaoPublicoService {
	return &AvaliacaoPublicoService{db: db, mail: m, frontendURL: strings.TrimRight(frontendURL, "/")}
}

// fezCheckin: a pessoa tem, numa compra dela, ingresso usado neste evento
// (cortesia não conta: fica na conta do organizador).
func (s *AvaliacaoPublicoService) fezCheckin(eventoID, usuarioID int64) (bool, error) {
	var n int64
	err := s.db.Table("itens_pedido").
		Joins("JOIN pedidos ON pedidos.id = itens_pedido.pedido_id").
		Where("pedidos.evento_id = ? AND pedidos.usuario_id = ? AND itens_pedido.status = ? AND NOT itens_pedido.cortesia",
			eventoID, usuarioID, domain.StatusItemUtilizado).
		Count(&n).Error
	return n > 0, err
}

// PodeAvaliar diz se a pessoa pode avaliar agora e, se não, por quê.
func (s *AvaliacaoPublicoService) PodeAvaliar(e *domain.Evento, usuarioID int64, agora time.Time) (bool, string, error) {
	if ok, motivo := janelaAvaliacao(e, agora); !ok {
		return false, motivo, nil
	}
	ok, err := s.fezCheckin(e.ID, usuarioID)
	if err != nil || !ok {
		return false, "só quem foi ao evento (fez check-in) pode avaliar", err
	}
	return true, "", nil
}

func (s *AvaliacaoPublicoService) Minha(eventoID, usuarioID int64) (*AvaliacaoPublico, error) {
	var a AvaliacaoPublico
	err := s.db.Where("evento_id = ? AND usuario_id = ?", eventoID, usuarioID).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

// Salvar cria ou edita a avaliação da pessoa (uma por evento).
func (s *AvaliacaoPublicoService) Salvar(e *domain.Evento, usuarioID int64, nota int, comentario string) (*AvaliacaoPublico, error) {
	ok, motivo, err := s.PodeAvaliar(e, usuarioID, time.Now())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNaoPodeAvaliar, motivo)
	}
	if nota < 1 || nota > 5 {
		return nil, fmt.Errorf("%w: a nota vai de 1 a 5", ErrNaoPodeAvaliar)
	}
	comentario = strings.TrimSpace(comentario)
	if utf8.RuneCountInString(comentario) > MaxComentarioChars {
		return nil, fmt.Errorf("%w: o comentário tem no máximo %d caracteres", ErrNaoPodeAvaliar, MaxComentarioChars)
	}
	if err := s.db.Exec(`INSERT INTO avaliacoes_publico (evento_id, usuario_id, nota, comentario) VALUES (?, ?, ?, ?)
		ON CONFLICT (evento_id, usuario_id) DO UPDATE SET nota = EXCLUDED.nota, comentario = EXCLUDED.comentario, atualizado_em = now()`,
		e.ID, usuarioID, nota, comentario).Error; err != nil {
		return nil, err
	}
	return s.Minha(e.ID, usuarioID)
}

// Resumo: média (só com 3 ou mais) e total, sem as ocultadas.
type ResumoAvaliacoes struct {
	Media        float64 `json:"media"`
	Total        int64   `json:"total"`
	MostrarMedia bool    `json:"mostrar_media"`
}

func resumir(total int64, soma int64) ResumoAvaliacoes {
	r := ResumoAvaliacoes{Total: total, MostrarMedia: total >= MinimoParaMedia}
	if total > 0 && r.MostrarMedia {
		r.Media = float64((soma*20/total+1)/2) / 10 // uma casa, arredondada
	}
	return r
}

func (s *AvaliacaoPublicoService) ResumoDoEvento(eventoID int64) (ResumoAvaliacoes, error) {
	var l struct{ Total, Soma int64 }
	err := s.db.Model(&AvaliacaoPublico{}).Select("COUNT(*) as total, COALESCE(SUM(nota),0) as soma").
		Where("evento_id = ? AND NOT ocultada", eventoID).Scan(&l).Error
	return resumir(l.Total, l.Soma), err
}

func (s *AvaliacaoPublicoService) ResumoDoOrganizador(organizadorID int64) (ResumoAvaliacoes, error) {
	var l struct{ Total, Soma int64 }
	err := s.db.Table("avaliacoes_publico").Select("COUNT(*) as total, COALESCE(SUM(avaliacoes_publico.nota),0) as soma").
		Joins("JOIN eventos ON eventos.id = avaliacoes_publico.evento_id").
		Where("eventos.organizador_id = ? AND NOT avaliacoes_publico.ocultada", organizadorID).Scan(&l).Error
	return resumir(l.Total, l.Soma), err
}

// AvaliacaoListada: o que aparece em público (nome curto, nunca e-mail).
type AvaliacaoListada struct {
	ID         int64     `json:"id"`
	Nome       string    `json:"nome"`
	Nota       int       `json:"nota"`
	Comentario string    `json:"comentario"`
	CriadoEm   time.Time `json:"criado_em"`
}

func (s *AvaliacaoPublicoService) ListarDoEvento(eventoID int64, limite int) ([]AvaliacaoListada, error) {
	var linhas []struct {
		ID         int64
		Nome       string
		Nota       int
		Comentario string
		CriadoEm   time.Time
	}
	err := s.db.Table("avaliacoes_publico").
		Select("avaliacoes_publico.id, usuarios.nome, avaliacoes_publico.nota, avaliacoes_publico.comentario, avaliacoes_publico.criado_em").
		Joins("JOIN usuarios ON usuarios.id = avaliacoes_publico.usuario_id").
		Where("avaliacoes_publico.evento_id = ? AND NOT avaliacoes_publico.ocultada", eventoID).
		Order("avaliacoes_publico.comentario = '', avaliacoes_publico.criado_em DESC"). // com comentário primeiro
		Limit(limite).Scan(&linhas).Error
	out := make([]AvaliacaoListada, 0, len(linhas))
	for _, l := range linhas {
		out = append(out, AvaliacaoListada{ID: l.ID, Nome: NomeAvaliador(l.Nome), Nota: l.Nota, Comentario: l.Comentario, CriadoEm: l.CriadoEm})
	}
	return out, err
}

// --- Admin ---

type AvaliacaoAdmin struct {
	ID           int64     `json:"id"`
	EventoTitulo string    `json:"evento_titulo"`
	EventoSlug   string    `json:"evento_slug"`
	UsuarioEmail string    `json:"usuario_email"`
	Nota         int       `json:"nota"`
	Comentario   string    `json:"comentario"`
	Ocultada     bool      `json:"ocultada"`
	CriadoEm     time.Time `json:"criado_em"`
}

func (s *AvaliacaoPublicoService) ListarAdmin(limite int) ([]AvaliacaoAdmin, error) {
	var out []AvaliacaoAdmin
	err := s.db.Table("avaliacoes_publico").
		Select("avaliacoes_publico.id, eventos.titulo as evento_titulo, eventos.slug as evento_slug, usuarios.email as usuario_email, avaliacoes_publico.nota, avaliacoes_publico.comentario, avaliacoes_publico.ocultada, avaliacoes_publico.criado_em").
		Joins("JOIN eventos ON eventos.id = avaliacoes_publico.evento_id").
		Joins("JOIN usuarios ON usuarios.id = avaliacoes_publico.usuario_id").
		Order("avaliacoes_publico.criado_em DESC").Limit(limite).Scan(&out).Error
	return out, err
}

func (s *AvaliacaoPublicoService) DefinirOcultada(id int64, ocultada bool) (bool, error) {
	res := s.db.Model(&AvaliacaoPublico{}).Where("id = ?", id).Update("ocultada", ocultada)
	return res.RowsAffected > 0, res.Error
}

// --- Convite "Como foi?" depois do evento ---

// deveConvidar: entre 3 e 48 horas depois do fim (sem fim: do início + 3 h).
// Testado.
func deveConvidar(e *domain.Evento, agora time.Time) bool {
	if e.InicioEm == nil {
		return false
	}
	fim := e.InicioEm.Add(3 * time.Hour)
	if e.FimEm != nil {
		fim = *e.FimEm
	}
	desde := agora.Sub(fim)
	return desde >= 3*time.Hour && desde <= 48*time.Hour
}

// ConvidarParaAvaliar manda o e-mail "Como foi?" a quem fez check-in (uma
// vez por pessoa e evento). Roda junto do job de lembretes.
func (s *AvaliacaoPublicoService) ConvidarParaAvaliar(agora time.Time) (int, error) {
	// Item 9.7: o e-mail pós-evento passou a levar também a pesquisa e os
	// certificados (um e-mail só por pessoa).
	if s.posEvento != nil {
		return s.posEvento.Processar(agora)
	}
	var eventos []domain.Evento
	if err := s.db.Where("status = ? AND inicio_em BETWEEN ? AND ?", domain.StatusEventoPublicado, agora.AddDate(0, 0, -40), agora).
		Find(&eventos).Error; err != nil {
		return 0, err
	}
	enviados := 0
	for i := range eventos {
		e := &eventos[i]
		if !deveConvidar(e, agora) {
			continue
		}
		var pessoas []struct{ Nome, Email string }
		if err := s.db.Table("itens_pedido").
			Select("DISTINCT usuarios.nome, usuarios.email").
			Joins("JOIN pedidos ON pedidos.id = itens_pedido.pedido_id").
			Joins("JOIN usuarios ON usuarios.id = pedidos.usuario_id").
			Where("pedidos.evento_id = ? AND itens_pedido.status = ? AND NOT itens_pedido.cortesia", e.ID, domain.StatusItemUtilizado).
			Scan(&pessoas).Error; err != nil {
			return enviados, err
		}
		for _, p := range pessoas {
			res := s.db.Exec("INSERT INTO lembretes_enviados (evento_id, email, tipo) VALUES (?, ?, 'pos') ON CONFLICT DO NOTHING", e.ID, strings.ToLower(p.Email))
			if res.Error != nil {
				return enviados, res.Error
			}
			if res.RowsAffected == 0 {
				continue
			}
			link := s.frontendURL + "/e/" + e.Slug + "#avaliar"
			corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111"><p>Olá, %s! Como foi o <strong>%s</strong>?</p><p>Sua avaliação ajuda outras pessoas a escolher e o organizador a melhorar. Leva menos de um minuto.</p><p><a href="%s" style="background:#111;color:#fff;padding:10px 16px;border-radius:8px;text-decoration:none">Avaliar o evento</a></p><p style="color:#777;font-size:12px">Você pode avaliar até %s.</p></div>`,
				html.EscapeString(primeiroNome(p.Nome)), html.EscapeString(e.Titulo), link, fimParaAvaliar(e).In(FusoDoEvento(e)).Format("02/01/2006"))
			if err := s.mail.Enviar(p.Email, "Como foi o "+e.Titulo+"?", corpo); err == nil {
				enviados++
			}
		}
	}
	return enviados, nil
}
