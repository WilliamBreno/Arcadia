package service

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/go-pdf/fpdf"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrGestaoArtista = errors.New("convite ou dados de gestão inválidos; confira o e-mail, o prazo e as permissões")
var ErrAgendaArtista = errors.New("informe o tipo, título e período válido da agenda (até 366 dias)")

type GrupoGestor struct {
	GrupoID   int64
	UsuarioID int64
}

func (GrupoGestor) TableName() string { return "grupo_gestores" }

type GrupoGestaoConvite struct {
	ID             int64     `json:"id"`
	GrupoID        int64     `json:"grupo_id"`
	Email          string    `json:"email"`
	Papel          string    `json:"papel"`
	AntigoCogestor bool      `json:"antigo_cogestor"`
	CriadoPor      int64     `json:"-"`
	TokenHash      string    `json:"-"`
	ExpiraEm       time.Time `json:"expira_em"`
	CriadoEm       time.Time `json:"-"`
}

func (GrupoGestaoConvite) TableName() string { return "grupo_gestao_convites" }

type GestorVisao struct {
	UsuarioID int64  `json:"usuario_id"`
	Email     string `json:"email"`
	Lider     bool   `json:"lider"`
}
type GestaoArtista struct {
	Lider    bool                 `json:"lider"`
	Gestores []GestorVisao        `json:"gestores"`
	Convites []GrupoGestaoConvite `json:"convites"`
}

func (s *GruposService) doGestor(usuarioID, grupoID int64) (*Grupo, error) {
	var g Grupo
	if err := s.db.First(&g, grupoID).Error; err != nil {
		return nil, ErrArtistaNaoEncontrado
	}
	if g.LiderUsuarioID == usuarioID {
		return &g, nil
	}
	var n int64
	if err := s.db.Model(&GrupoGestor{}).Where("grupo_id = ? AND usuario_id = ?", grupoID, usuarioID).Count(&n).Error; err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrArtistaSemPermissao
	}
	return &g, nil
}

func (s *GruposService) Gestao(usuarioID, grupoID int64) (*GestaoArtista, error) {
	g, err := s.doGestor(usuarioID, grupoID)
	if err != nil {
		return nil, err
	}
	v := &GestaoArtista{Lider: g.LiderUsuarioID == usuarioID, Gestores: []GestorVisao{}, Convites: []GrupoGestaoConvite{}}
	if err := s.db.Raw(`SELECT u.id AS usuario_id, u.email, (u.id = ?) AS lider FROM usuarios u
 WHERE u.id = ? OR EXISTS (SELECT 1 FROM grupo_gestores gg WHERE gg.grupo_id = ? AND gg.usuario_id = u.id)
 ORDER BY lider DESC, u.id`, g.LiderUsuarioID, g.LiderUsuarioID, grupoID).Scan(&v.Gestores).Error; err != nil {
		return nil, err
	}
	if v.Lider {
		err = s.db.Where("grupo_id = ? AND expira_em > ?", grupoID, time.Now()).Order("id").Find(&v.Convites).Error
	}
	return v, err
}

func (s *GruposService) ConvidarGestao(usuarioID, grupoID int64, email, papel string, antigoCogestor bool) error {
	email = NormalizarEmail(email)
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 255 || (papel != "lider" && papel != "cogestor") {
		return ErrGestaoArtista
	}
	token, hash := novoTokenGrupo()
	var nome string
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var g Grupo
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, grupoID).Error; err != nil {
			return ErrArtistaNaoEncontrado
		}
		if g.LiderUsuarioID != usuarioID {
			return ErrArtistaSemPermissao
		}
		var lider domain.Usuario
		if err := tx.First(&lider, usuarioID).Error; err != nil {
			return err
		}
		if NormalizarEmail(lider.Email) == email {
			return ErrGestaoArtista
		}
		// O perfil solo representa a própria pessoa; transferi-lo trocaria a identidade e as autorizações de imagem.
		if g.Solo && papel == "lider" {
			return ErrSoloUmIntegrante
		}
		nome = g.Nome
		q := tx.Where("grupo_id = ? AND papel = ?", grupoID, papel)
		if papel == "cogestor" {
			q = q.Where("email = ?", email)
		}
		if err := q.Delete(&GrupoGestaoConvite{}).Error; err != nil {
			return err
		}
		return tx.Create(&GrupoGestaoConvite{GrupoID: grupoID, Email: email, Papel: papel, AntigoCogestor: antigoCogestor, CriadoPor: usuarioID, TokenHash: hash, ExpiraEm: time.Now().Add(7 * 24 * time.Hour)}).Error
	})
	if err != nil {
		return err
	}
	if s.mail != nil {
		link := s.frontendURL + "/artista-gestao/" + token
		corpo := s.envelope(fmt.Sprintf("<p>Você recebeu um convite para ser %s de <strong>%s</strong>.</p><p>Entre com a conta Evve deste e-mail e sua própria senha. O convite vale por 7 dias.</p>%s", html.EscapeString(papel), html.EscapeString(nome), s.botao(link, "Responder ao convite")))
		return s.mail.Enviar(email, "Convite de gestão: "+nome, corpo)
	}
	return nil
}

type ConviteGestaoVisao struct {
	GrupoNome      string    `json:"grupo_nome"`
	Papel          string    `json:"papel"`
	AntigoCogestor bool      `json:"antigo_cogestor"`
	ExpiraEm       time.Time `json:"expira_em"`
}

func (s *GruposService) VerGestaoConvite(usuarioID int64, token string) (*ConviteGestaoVisao, error) {
	var c GrupoGestaoConvite
	if len(token) < 20 || s.db.Where("token_hash = ? AND expira_em > ?", hashToken(token), time.Now()).First(&c).Error != nil {
		return nil, ErrGestaoArtista
	}
	var u domain.Usuario
	if err := s.db.First(&u, usuarioID).Error; err != nil {
		return nil, err
	}
	if NormalizarEmail(u.Email) != c.Email {
		return nil, ErrConviteOutroEmail
	}
	var g Grupo
	if err := s.db.First(&g, c.GrupoID).Error; err != nil {
		return nil, err
	}
	if c.CriadoPor != g.LiderUsuarioID {
		return nil, ErrGestaoArtista
	}
	return &ConviteGestaoVisao{g.Nome, c.Papel, c.AntigoCogestor, c.ExpiraEm}, nil
}

func (s *GruposService) DecidirGestaoConvite(usuarioID int64, token string, aceitar bool) (int64, error) {
	var id int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var c GrupoGestaoConvite
		if len(token) < 20 || tx.Where("token_hash = ?", hashToken(token)).First(&c).Error != nil {
			return ErrGestaoArtista
		}
		var g Grupo
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, c.GrupoID).Error; err != nil {
			return err
		}
		// Releitura após a trava também protege contra revogação e dois aceites simultâneos.
		if tx.Where("id = ? AND expira_em > ?", c.ID, time.Now()).First(&c).Error != nil || c.CriadoPor != g.LiderUsuarioID {
			return ErrGestaoArtista
		}
		var u domain.Usuario
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, usuarioID).Error; err != nil {
			return err
		}
		if NormalizarEmail(u.Email) != c.Email {
			return ErrConviteOutroEmail
		}
		id = g.ID
		if aceitar {
			if c.Papel == "lider" {
				if g.Solo {
					return ErrSoloUmIntegrante
				}
				var n int64
				if err := tx.Model(&Grupo{}).Where("lider_usuario_id = ?", usuarioID).Count(&n).Error; err != nil {
					return err
				}
				if n >= MaxGruposPorLider {
					return ErrLimiteArtistas
				}
				if c.AntigoCogestor {
					if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&GrupoGestor{g.ID, g.LiderUsuarioID}).Error; err != nil {
						return err
					}
				} else if err := tx.Where("grupo_id = ? AND usuario_id = ?", g.ID, g.LiderUsuarioID).Delete(&GrupoGestor{}).Error; err != nil {
					return err
				}
				if err := tx.Where("grupo_id = ? AND usuario_id = ?", g.ID, usuarioID).Delete(&GrupoGestor{}).Error; err != nil {
					return err
				}
				if err := tx.Model(&Grupo{}).Where("id = ?", g.ID).Updates(map[string]any{"lider_usuario_id": usuarioID, "atualizado_em": time.Now()}).Error; err != nil {
					return err
				}
				return tx.Where("grupo_id = ?", g.ID).Delete(&GrupoGestaoConvite{}).Error
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&GrupoGestor{g.ID, usuarioID}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&c).Error
	})
	return id, err
}

func (s *GruposService) RemoverGestao(usuarioID, grupoID, alvoID int64, convite bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var g Grupo
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, grupoID).Error; err != nil {
			return ErrArtistaNaoEncontrado
		}
		if usuarioID != g.LiderUsuarioID {
			return ErrArtistaSemPermissao
		}
		if convite {
			return tx.Where("grupo_id = ? AND id = ?", grupoID, alvoID).Delete(&GrupoGestaoConvite{}).Error
		}
		if alvoID == g.LiderUsuarioID {
			return ErrArtistaSemPermissao
		}
		return tx.Where("grupo_id = ? AND usuario_id = ?", grupoID, alvoID).Delete(&GrupoGestor{}).Error
	})
}

type AgendaArtista struct {
	ID      int64  `json:"id"`
	GrupoID int64  `json:"-"`
	Tipo    string `json:"tipo"`
	Titulo  string `json:"titulo"`
	Inicio  string `json:"inicio"`
	Fim     string `json:"fim"`
	Publico bool   `json:"publico"`
}

func (AgendaArtista) TableName() string { return "grupo_agenda" }

func NormalizarAgenda(d AgendaArtista) (AgendaArtista, error) {
	d.Titulo = strings.TrimSpace(d.Titulo)
	inicio, e1 := time.Parse("2006-01-02", d.Inicio)
	fim, e2 := time.Parse("2006-01-02", d.Fim)
	if e1 != nil || e2 != nil || inicio.Year() < 1 || fim.Before(inicio) || fim.Sub(inicio) > 366*24*time.Hour || utf8.RuneCountInString(d.Titulo) > 120 ||
		(d.Tipo != "bloqueio" && d.Tipo != "apresentacao") || (d.Tipo == "apresentacao" && d.Titulo == "") {
		return d, ErrAgendaArtista
	}
	return d, nil
}

// DATE é formatado no SQL para não depender da serialização do driver.
func (s *GruposService) agenda(grupoID int64, publica bool) ([]AgendaArtista, error) {
	out := []AgendaArtista{}
	err := s.db.Raw(`SELECT id, grupo_id, tipo, titulo, to_char(inicio,'YYYY-MM-DD') AS inicio,
 to_char(fim,'YYYY-MM-DD') AS fim, publico FROM grupo_agenda WHERE grupo_id = ? AND (NOT ? OR publico) ORDER BY inicio, id`, grupoID, publica).Scan(&out).Error
	if publica {
		for i := range out {
			if out[i].Tipo == "bloqueio" {
				out[i].Titulo = "Indisponível"
			}
		}
	}
	return out, err
}
func (s *GruposService) Agenda(usuarioID, grupoID int64) ([]AgendaArtista, error) {
	if _, err := s.doGestor(usuarioID, grupoID); err != nil {
		return nil, err
	}
	return s.agenda(grupoID, false)
}
func (s *GruposService) SalvarAgenda(usuarioID, grupoID, id int64, d AgendaArtista) error {
	if _, err := s.doGestor(usuarioID, grupoID); err != nil {
		return err
	}
	d, err := NormalizarAgenda(d)
	if err != nil {
		return err
	}
	if id == 0 {
		d.ID, d.GrupoID = 0, grupoID
		return s.db.Create(&d).Error
	}
	r := s.db.Model(&AgendaArtista{}).Where("id = ? AND grupo_id = ?", id, grupoID).Updates(map[string]any{"tipo": d.Tipo, "titulo": d.Titulo, "inicio": d.Inicio, "fim": d.Fim, "publico": d.Publico})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return ErrArtistaNaoEncontrado
	}
	return nil
}
func (s *GruposService) RemoverAgenda(usuarioID, grupoID, id int64) error {
	if _, err := s.doGestor(usuarioID, grupoID); err != nil {
		return err
	}
	return s.db.Where("id = ? AND grupo_id = ?", id, grupoID).Delete(&AgendaArtista{}).Error
}

// Usa exclusivamente a mesma projeção do perfil público, inclusive as regras de menores.
func (s *GruposService) MidiaKit(usuarioID, grupoID int64) ([]byte, error) {
	g, err := s.doGestor(usuarioID, grupoID)
	if err != nil {
		return nil, err
	}
	p, err := s.Publico(g.Slug)
	if err != nil {
		return nil, err
	}
	return gerarMidiaKit(p, s.frontendURL, s.nome, s.imagemMidiaKit)
}
func gerarMidiaKit(p *GrupoPublico, frontendURL, plataforma string, carregarImagem func(string) []byte) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Mídia kit — "+p.Nome, true)
	pdf.SetAuthor(plataforma, true)
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AddPage()
	tr := pdf.UnicodeTranslatorFromDescriptor("cp1252")
	texto := func(t string) { pdf.MultiCell(174, 6, tr(t), "", "L", false); pdf.Ln(2) }
	quantidadeFotos := 0
	foto := func(url string) {
		if url == "" || carregarImagem == nil || quantidadeFotos >= 8 {
			return
		}
		quantidadeFotos++
		b := carregarImagem(url)
		if len(b) == 0 {
			return
		}
		nome := fmt.Sprintf("foto-%d", quantidadeFotos)
		info := pdf.RegisterImageOptionsReader(nome, fpdf.ImageOptions{ImageType: "JPG"}, bytes.NewReader(b))
		if info == nil {
			return
		}
		w, h := 90.0, 90.0*info.Height()/info.Width()
		if h > 65 {
			w, h = w*65/h, 65
		}
		if pdf.GetY()+h > 275 {
			pdf.AddPage()
		}
		pdf.ImageOptions(nome, 18, pdf.GetY(), w, h, false, fpdf.ImageOptions{ImageType: "JPG"}, 0, "")
		pdf.SetY(pdf.GetY() + h + 4)
	}
	pdf.SetFont("Helvetica", "B", 24)
	texto(p.Nome)
	pdf.SetFont("Helvetica", "", 11)
	texto("Mídia kit • " + p.TipoNome)
	foto(p.LogoURL)
	foto(p.CapaURL)
	texto(p.Bio)
	texto(strings.Trim(p.Cidade+" / "+p.UF, " /"))
	if p.Instagram != "" {
		texto("Instagram: @" + p.Instagram)
	}
	if p.TikTok != "" {
		texto("TikTok: " + p.TikTok)
	}
	if p.YouTube != "" {
		texto("YouTube: " + p.YouTube)
	}
	texto("Perfil: " + frontendURL + "/artista/" + p.Slug)
	for _, i := range p.Integrantes {
		linha := i.NomeArtistico
		if i.Funcao != "" {
			linha += " • " + i.Funcao
		}
		if i.Anos > 0 {
			linha += fmt.Sprintf(" • %d anos de experiência", i.Anos)
		}
		texto(linha)
		foto(i.FotoURL)
		if i.FotoURL != "" {
			texto("Foto: " + i.FotoURL)
		}
	}
	if p.OutrosIntegrantes > 0 {
		texto(fmt.Sprintf("+ %d integrantes", p.OutrosIntegrantes))
	}
	for _, m := range p.Midias {
		if m.Tipo == "foto" {
			foto(m.URL)
		}
		texto(m.Legenda + "\n" + m.URL)
	}
	for _, r := range p.Resultados {
		linha := r.EventoTitulo + " • " + r.Tipo
		if r.Posicao > 0 {
			linha += fmt.Sprintf(" • posição %d", r.Posicao)
		}
		texto(linha)
	}
	var b bytes.Buffer
	if err := pdf.Output(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
