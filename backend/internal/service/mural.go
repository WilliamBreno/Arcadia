package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrMuralAcesso = errors.New("você não tem permissão para esta ação no mural")
var ErrMuralDados = errors.New("dados do mural inválidos")
var ErrMuralNaoEncontrado = errors.New("publicação ou evento não encontrado")
var ErrMuralLimite = errors.New("limite de dez publicações por dia neste evento atingido")

type MuralService struct {
	db     *gorm.DB
	claude *anthropic.Client
}

func NovoMuralService(db *gorm.DB, chave string, opcoes ...option.RequestOption) *MuralService {
	s := &MuralService{db: db}
	if strings.TrimSpace(chave) != "" {
		opcoes = append([]option.RequestOption{option.WithAPIKey(chave), option.WithRequestTimeout(25 * time.Second), option.WithMaxRetries(0)}, opcoes...)
		c := anthropic.NewClient(opcoes...)
		s.claude = &c
	}
	return s
}

type MuralPost struct {
	ID        int64           `json:"id"`
	UsuarioID int64           `json:"-"`
	Autor     string          `json:"autor"`
	Texto     string          `json:"texto"`
	Status    string          `json:"status"`
	Destaque  bool            `json:"destaque"`
	Fotos     []int64         `json:"fotos" gorm:"-"`
	Curtidas  int64           `json:"curtidas"`
	CriadoEm  time.Time       `json:"criado_em"`
	IAStatus  string          `json:"ia_status,omitempty"`
	IAAlerta  bool            `json:"ia_alerta,omitempty"`
	IAMotivos json.RawMessage `json:"ia_motivos,omitempty" gorm:"column:ia_motivos"`
	Motivo    string          `json:"motivo,omitempty"`
	Denuncias int64           `json:"denuncias,omitempty"`
}
type MuralDenuncia struct {
	ID       int64     `json:"id"`
	Motivo   string    `json:"motivo"`
	CriadoEm time.Time `json:"criado_em"`
}
type MuralEu struct {
	PodePostar   bool        `json:"pode_postar"`
	Motivo       string      `json:"motivo"`
	Moderador    bool        `json:"moderador"`
	EntradaLivre bool        `json:"entrada_livre"`
	Meus         []MuralPost `json:"meus"`
	Curtidas     []int64     `json:"curtidas"`
}
type MuralPagina struct {
	Titulo    string      `json:"titulo"`
	Slug      string      `json:"slug"`
	Posts     []MuralPost `json:"posts"`
	Destaques []MuralPost `json:"destaques"`
}

func muralEvento(db *gorm.DB, slug string) (*domain.Evento, error) {
	var e domain.Evento
	err := db.Where("slug=? AND status IN ? AND visibilidade IN ?", slug, []string{"publicado", "encerrado"}, []string{"publico", "nao_listado"}).First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMuralNaoEncontrado
	}
	return &e, err
}
func muralModerador(db *gorm.DB, usuarioID, eventoID int64) (bool, error) {
	var n int64
	err := db.Raw(`SELECT count(*) FROM eventos e JOIN organizadores o ON o.id=e.organizador_id WHERE e.id=? AND (o.usuario_id=? OR EXISTS(SELECT 1 FROM organizador_membros m WHERE m.organizador_id=o.id AND m.usuario_id=? AND m.papel='gestor'))`, eventoID, usuarioID, usuarioID).Scan(&n).Error
	return n > 0, err
}
func janelaMural(e *domain.Evento, agora time.Time) (bool, string) {
	if e.Status != domain.StatusEventoPublicado && e.Status != domain.StatusEventoEncerrado {
		return false, "o mural abre quando o evento é publicado e começa"
	}
	if e.InicioEm == nil || agora.Before(*e.InicioEm) {
		return false, "novas publicações abrem quando o evento começa"
	}
	fim := e.InicioEm
	if e.FimEm != nil {
		fim = e.FimEm
	}
	if agora.After(fim.AddDate(0, 0, 7)) {
		return false, "o prazo para novas publicações terminou (sete dias após o término)"
	}
	return true, ""
}
func nascimentoMural(db *gorm.DB, usuarioID int64) (*time.Time, error) {
	var r struct{ Data *time.Time }
	err := db.Raw(`SELECT MAX(d) AS data FROM (SELECT data_nascimento::date AS d FROM perfis_talento WHERE usuario_id=? UNION ALL SELECT data_nascimento::date FROM fichas_participacao WHERE usuario_id=? UNION ALL SELECT data_nascimento::date FROM grupo_integrantes WHERE usuario_id=?) x`, usuarioID, usuarioID, usuarioID).Scan(&r).Error
	return r.Data, err
}
func muralParticipa(db *gorm.DB, usuarioID int64, e *domain.Evento) (bool, error) {
	var n int64
	err := db.Raw(`SELECT (SELECT count(*) FROM papeis_evento WHERE evento_id=? AND usuario_id=? AND status='confirmado')+
 (SELECT count(*) FROM organizadores o WHERE o.id=? AND (o.usuario_id=? OR EXISTS(SELECT 1 FROM organizador_membros m WHERE m.organizador_id=o.id AND m.usuario_id=?)))+
 (SELECT count(*) FROM coproducoes c JOIN organizadores o ON o.id=c.organizador_id WHERE c.evento_id=? AND o.usuario_id=? AND c.status='ativa')+
 (SELECT count(*) FROM expositores WHERE evento_id=? AND usuario_id=? AND status='aprovado')+
 (SELECT count(*) FROM itens_pedido i JOIN pedidos p ON p.id=i.pedido_id WHERE p.evento_id=? AND p.usuario_id=? AND i.status='utilizado' AND NOT i.cortesia AND NOT i.adicional AND i.ficha_id IS NULL AND i.expositor_id IS NULL)+
 (SELECT count(*) FROM grupo_integrantes gi WHERE gi.usuario_id=? AND gi.status='ativo' AND (EXISTS(SELECT 1 FROM atracoes a WHERE a.grupo_id=gi.grupo_id AND a.evento_id=? AND a.status='confirmado') OR EXISTS(SELECT 1 FROM fichas_participacao f WHERE f.grupo_id=gi.grupo_id AND f.evento_id=? AND f.status='aprovado')))`, e.ID, usuarioID, e.OrganizadorID, usuarioID, usuarioID, e.ID, usuarioID, e.ID, usuarioID, e.ID, usuarioID, usuarioID, e.ID, e.ID).Scan(&n).Error
	return n > 0, err
}
func muralLivre(db *gorm.DB, id int64) (bool, error) {
	var n int64
	err := db.Table("tipos_ingresso").Where("evento_id=? AND NOT inscricao_talento AND NOT espaco_expositor AND NOT adicional", id).Count(&n).Error
	return n == 0, err
}
func muralPodePostar(db *gorm.DB, usuarioID int64, e *domain.Evento, confirmouLivre bool) (bool, string, error) {
	if ok, motivo := janelaMural(e, time.Now()); !ok {
		return false, motivo, nil
	}
	var u domain.Usuario
	if err := db.First(&u, usuarioID).Error; err != nil {
		return false, "", err
	}
	if u.EmailVerificadoEm == nil {
		return false, "verifique seu e-mail antes de publicar", nil
	}
	nasc, err := nascimentoMural(db, usuarioID)
	if err != nil {
		return false, "", err
	}
	if nasc != nil && Idade(*nasc, time.Now()) < 18 {
		if Idade(*nasc, time.Now()) < 12 {
			return false, "publicações no mural são para pessoas com 12 anos ou mais", nil
		}
		var p PerfilTalento
		if err = db.Where("usuario_id=?", usuarioID).First(&p).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", err
		}
		if p.AutorizadoEm == nil {
			return false, "para publicar sendo menor de idade, complete o perfil de talento e obtenha a autorização do responsável", nil
		}
	}
	participa, err := muralParticipa(db, usuarioID, e)
	if err != nil {
		return false, "", err
	}
	if participa {
		return true, "", nil
	}
	livre, err := muralLivre(db, e.ID)
	if err != nil {
		return false, "", err
	}
	if livre && confirmouLivre {
		return true, "", nil
	}
	if livre {
		return false, "confirme que está participando deste evento de entrada livre", nil
	}
	return false, "o mural aceita publicações da equipe e participantes confirmados; para a plateia, é necessário ter feito check-in", nil
}
func preencherMural(db *gorm.DB, posts []MuralPost) error {
	if len(posts) == 0 {
		return nil
	}
	ids, usuarios := []int64{}, []int64{}
	for _, p := range posts {
		ids = append(ids, p.ID)
		usuarios = append(usuarios, p.UsuarioID)
	}
	var fotos []struct{ ID, PostID int64 }
	if err := db.Table("mural_fotos").Select("id,post_id").Where("post_id IN ?", ids).Order("id").Scan(&fotos).Error; err != nil {
		return err
	}
	var nomes []struct {
		ID         int64
		Nome       string
		Nascimento *time.Time
	}
	if err := db.Raw(`SELECT u.id,u.nome,(SELECT MAX(d) FROM (SELECT data_nascimento::date AS d FROM perfis_talento WHERE usuario_id=u.id UNION ALL SELECT data_nascimento::date FROM fichas_participacao WHERE usuario_id=u.id UNION ALL SELECT data_nascimento::date FROM grupo_integrantes WHERE usuario_id=u.id) n) AS nascimento FROM usuarios u WHERE u.id IN ?`, usuarios).Scan(&nomes).Error; err != nil {
		return err
	}
	for i := range posts {
		p := &posts[i]
		p.Fotos = []int64{}
		p.Autor = "Participante do evento"
		for _, f := range fotos {
			if f.PostID == p.ID {
				p.Fotos = append(p.Fotos, f.ID)
			}
		}
		for _, n := range nomes {
			if n.ID == p.UsuarioID && n.Nascimento != nil && Idade(*n.Nascimento, time.Now()) >= 18 {
				p.Autor = NomeAvaliador(n.Nome)
			}
		}
	}
	return nil
}

const muralNascimentoPublicoSQL = `(SELECT MAX(d) FROM (SELECT data_nascimento::date AS d FROM perfis_talento WHERE usuario_id=p.usuario_id UNION ALL SELECT data_nascimento::date FROM fichas_participacao WHERE usuario_id=p.usuario_id UNION ALL SELECT data_nascimento::date FROM grupo_integrantes WHERE usuario_id=p.usuario_id) n)`

// Uma revogação de autorização de menor também retira seus posts/fotos do público.
const muralConsentimentoPublicoSQL = `(` + muralNascimentoPublicoSQL + ` IS NULL OR ` + muralNascimentoPublicoSQL + ` <= CURRENT_DATE-interval '18 years' OR (` + muralNascimentoPublicoSQL + ` <= CURRENT_DATE-interval '12 years' AND EXISTS(SELECT 1 FROM perfis_talento t WHERE t.usuario_id=p.usuario_id AND t.autorizado_em IS NOT NULL AND (t.autoriza_foto OR NOT EXISTS(SELECT 1 FROM mural_fotos f WHERE f.post_id=p.id)))))`

func muralPosts(db *gorm.DB, eventoID, antes, usuarioID int64, status string, destaque, interno bool) ([]MuralPost, error) {
	posts := []MuralPost{}
	cols := "p.id,p.usuario_id,p.texto,p.status,p.destaque,p.criado_em,(SELECT count(*) FROM mural_curtidas c WHERE c.post_id=p.id) AS curtidas"
	if interno {
		cols += ",p.ia_status,p.ia_alerta,p.ia_motivos,p.motivo,(SELECT count(*) FROM mural_denuncias d WHERE d.post_id=p.id AND d.resolvido_em IS NULL) AS denuncias"
	}
	q := db.Table("mural_posts p").Select(cols).Where("p.evento_id=?", eventoID)
	if !interno {
		q = q.Where(muralConsentimentoPublicoSQL)
	}
	if antes > 0 {
		q = q.Where("p.id<?", antes)
	}
	if usuarioID > 0 {
		q = q.Where("p.usuario_id=?", usuarioID)
	}
	if status != "" {
		q = q.Where("p.status=?", status)
	}
	if destaque {
		q = q.Where("p.destaque=true AND p.status='aprovado'")
	}
	limite := 20
	if usuarioID > 0 {
		limite = 100
	}
	if err := q.Order("p.id DESC").Limit(limite).Scan(&posts).Error; err != nil {
		return nil, err
	}
	return posts, preencherMural(db, posts)
}
func (s *MuralService) Publico(slug string, antes int64) (*MuralPagina, error) {
	e, err := muralEvento(s.db, slug)
	if err != nil {
		return nil, err
	}
	posts, err := muralPosts(s.db, e.ID, antes, 0, "aprovado", false, false)
	if err != nil {
		return nil, err
	}
	d, err := muralPosts(s.db, e.ID, 0, 0, "aprovado", true, false)
	return &MuralPagina{e.Titulo, e.Slug, posts, d}, err
}
func (s *MuralService) Eu(usuarioID int64, slug string) (*MuralEu, error) {
	e, err := muralEvento(s.db, slug)
	if err != nil {
		return nil, err
	}
	r := &MuralEu{Curtidas: []int64{}}
	if r.PodePostar, r.Motivo, err = muralPodePostar(s.db, usuarioID, e, false); err != nil {
		return nil, err
	}
	if r.Moderador, err = muralModerador(s.db, usuarioID, e.ID); err != nil {
		return nil, err
	}
	if r.EntradaLivre, err = muralLivre(s.db, e.ID); err != nil {
		return nil, err
	}
	if r.Meus, err = muralPosts(s.db, e.ID, 0, usuarioID, "", false, true); err != nil {
		return nil, err
	}
	err = s.db.Table("mural_curtidas c").Joins("JOIN mural_posts p ON p.id=c.post_id").Where("p.evento_id=? AND c.usuario_id=? AND p.status='aprovado'", e.ID, usuarioID).Pluck("c.post_id", &r.Curtidas).Error
	return r, err
}
func (s *MuralService) Criar(ctx context.Context, usuarioID int64, slug, texto, chave string, fotos [][]byte, consentiu, confirmouLivre bool) (int64, error) {
	texto = strings.TrimSpace(texto)
	if !consentiu || utf8.RuneCountInString(texto) > 2000 || (texto == "" && len(fotos) == 0) || len(fotos) > 5 || len(chave) < 8 || len(chave) > 80 {
		return 0, ErrMuralDados
	}
	var id int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		e, err := muralEvento(tx, slug)
		if err != nil {
			return err
		}
		if err = tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).First(e, e.ID).Error; err != nil {
			return err
		}
		if err = tx.Table("mural_posts").Where("evento_id=? AND usuario_id=? AND chave=?", e.ID, usuarioID, chave).Pluck("id", &id).Error; err != nil {
			return err
		}
		if id > 0 {
			return nil
		}
		pode, motivo, err := muralPodePostar(tx, usuarioID, e, confirmouLivre)
		if err != nil {
			return err
		}
		if !pode {
			return fmt.Errorf("%w: %s", ErrMuralAcesso, motivo)
		}
		nasc, err := nascimentoMural(tx, usuarioID)
		if err != nil {
			return err
		}
		if len(fotos) > 0 && nasc != nil && Idade(*nasc, time.Now()) < 18 {
			var p PerfilTalento
			if err = tx.Where("usuario_id=?", usuarioID).First(&p).Error; err != nil {
				return err
			}
			if !p.AutorizaFoto {
				return fmt.Errorf("%w: o responsável precisa autorizar fotos no perfil de talento", ErrMuralAcesso)
			}
		}
		var n int64
		if err = tx.Table("mural_posts").Where("evento_id=? AND usuario_id=? AND criado_em>?", e.ID, usuarioID, time.Now().Add(-24*time.Hour)).Count(&n).Error; err != nil {
			return err
		}
		if n >= 10 {
			return ErrMuralLimite
		}
		if err = tx.Raw(`INSERT INTO mural_posts(evento_id,usuario_id,chave,texto) VALUES (?,?,?,?) RETURNING id`, e.ID, usuarioID, chave, texto).Scan(&id).Error; err != nil {
			return err
		}
		for _, foto := range fotos {
			if err = tx.Exec("INSERT INTO mural_fotos(post_id,jpeg) VALUES (?,?)", id, foto).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	s.Analisar(ctx, id)
	return id, nil
}
