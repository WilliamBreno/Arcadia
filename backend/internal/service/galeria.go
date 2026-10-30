package service

// Item 8.7: galeria oficial do evento. Decisões do dono (30/10/2026):
//   - o organizador e fotógrafos convidados (papel "fotografo") sobem
//     fotos, até 500 por evento e 10 MB cada; cada foto leva o crédito;
//   - só a própria pessoa adulta se marca (precisa de perfil de talento
//     18+); a foto marcada aparece no perfil dela; menor nunca é marcado;
//   - qualquer pessoa pede "não quero aparecer": a foto some na hora e o
//     organizador revisa (manter ou remover de vez).
//
// A imagem é sempre decodificada e regravada em JPEG: isso tira os
// metadados (EXIF, inclusive GPS) e reduz fotos grandes para 2048 px.

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png" // fotos em PNG também
	"io"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/storage"
)

const (
	MaxFotosPorEvento   = 500
	MaxBytesFoto        = 10 << 20
	ladoMaximoFoto      = 2048
	maxPixelsDecodifica = 60_000_000
)

var (
	ErrFotoInvalida       = errors.New("arquivo de imagem inválido")
	ErrFotoGrande         = errors.New("foto maior que 10 MB")
	ErrLimiteFotos        = fmt.Errorf("o evento já tem %d fotos (o limite)", MaxFotosPorEvento)
	ErrSemAcessoGaleria   = errors.New("sem acesso à galeria deste evento")
	ErrFotoNaoEncontrada  = errors.New("foto não encontrada")
	ErrMarcacaoSoAdulto   = errors.New("para se marcar, crie seu perfil de talento (só para quem tem 18 anos ou mais)")
	ErrRemocaoNaoEncontra = errors.New("pedido não encontrado")
)

type FotoEvento struct {
	ID         int64 `gorm:"primaryKey"`
	EventoID   int64
	EnviadaPor int64
	URL        string `gorm:"column:url"`
	Oculta     bool
	CriadoEm   time.Time
}

func (FotoEvento) TableName() string { return "fotos_evento" }

type MarcacaoFoto struct {
	ID        int64 `gorm:"primaryKey"`
	FotoID    int64
	UsuarioID int64
	CriadoEm  time.Time
}

func (MarcacaoFoto) TableName() string { return "marcacoes_foto" }

type RemocaoFoto struct {
	ID        int64 `gorm:"primaryKey"`
	FotoID    int64
	UsuarioID *int64
	Motivo    string
	Status    string
	CriadoEm  time.Time
}

func (RemocaoFoto) TableName() string { return "remocoes_foto" }

type GaleriaService struct {
	db            *gorm.DB
	eventos       *repository.EventoRepository
	usuarios      *repository.UsuarioRepository
	papeis        *repository.PapelEventoRepository
	perfis        *PerfilTalentoService
	armazenamento storage.Armazenamento
}

func NovoGaleriaService(db *gorm.DB, e *repository.EventoRepository, u *repository.UsuarioRepository, p *repository.PapelEventoRepository,
	perfis *PerfilTalentoService, a storage.Armazenamento) *GaleriaService {
	return &GaleriaService{db: db, eventos: e, usuarios: u, papeis: p, perfis: perfis, armazenamento: a}
}

// podeEnviar: organizador do evento ou fotógrafo confirmado.
func (s *GaleriaService) podeEnviar(usuarioID int64, e *domain.Evento) bool {
	var n int64
	s.db.Table("organizadores").Where("id = ? AND usuario_id = ?", e.OrganizadorID, usuarioID).Count(&n)
	if n > 0 {
		return true
	}
	p, err := s.papeis.Buscar(e.ID, usuarioID, domain.PapelFotografo)
	return err == nil && p.Status == domain.StatusPapelConfirmado
}

func (s *GaleriaService) ehOrganizador(usuarioID int64, e *domain.Evento) bool {
	var n int64
	s.db.Table("organizadores").Where("id = ? AND usuario_id = ?", e.OrganizadorID, usuarioID).Count(&n)
	return n > 0
}

// PodeEnviar diz ao frontend se mostra o envio de fotos.
func (s *GaleriaService) PodeEnviar(usuarioID, eventoID int64) bool {
	e, err := s.eventos.BuscarPorID(eventoID)
	return err == nil && s.podeEnviar(usuarioID, e)
}

// ReduzirParaJPEG decodifica, reduz o maior lado para no máximo `lado` px
// (média de cada bloco) e regrava em JPEG, sem metadados. Testada.
func ReduzirParaJPEG(conteudo []byte, lado int) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(conteudo))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixelsDecodifica {
		return nil, ErrFotoInvalida
	}
	img, _, err := image.Decode(bytes.NewReader(conteudo))
	if err != nil {
		return nil, ErrFotoInvalida
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > lado || h > lado {
		nw, nh := lado, h*lado/w
		if h > w {
			nw, nh = w*lado/h, lado
		}
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
		img = reduzir(img, nw, nh)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// reduzir faz a média de cada bloco da imagem original (boa qualidade ao
// diminuir, sem dependência externa).
func reduzir(src image.Image, nw, nh int) *image.RGBA {
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		y0, y1 := y*h/nh, (y+1)*h/nh
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < nw; x++ {
			x0, x1 := x*w/nw, (x+1)*w/nw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, bl, a, n uint64
			for yy := y0; yy < y1; yy++ {
				i := yy*rgba.Stride + x0*4
				for xx := x0; xx < x1; xx++ {
					r += uint64(rgba.Pix[i])
					g += uint64(rgba.Pix[i+1])
					bl += uint64(rgba.Pix[i+2])
					a += uint64(rgba.Pix[i+3])
					n++
					i += 4
				}
			}
			j := y*out.Stride + x*4
			out.Pix[j], out.Pix[j+1], out.Pix[j+2], out.Pix[j+3] = uint8(r/n), uint8(g/n), uint8(bl/n), uint8(a/n)
		}
	}
	return out
}

// Enviar guarda uma foto na galeria do evento.
func (s *GaleriaService) Enviar(usuarioID, eventoID int64, arquivo io.Reader, tamanho int64) (*FotoEvento, error) {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || !s.podeEnviar(usuarioID, e) {
		return nil, ErrSemAcessoGaleria
	}
	if tamanho > MaxBytesFoto {
		return nil, ErrFotoGrande
	}
	var n int64
	s.db.Model(&FotoEvento{}).Where("evento_id = ?", eventoID).Count(&n)
	if n >= MaxFotosPorEvento {
		return nil, ErrLimiteFotos
	}
	conteudo, err := io.ReadAll(io.LimitReader(arquivo, MaxBytesFoto+1))
	if err != nil {
		return nil, ErrFotoInvalida
	}
	if len(conteudo) > MaxBytesFoto {
		return nil, ErrFotoGrande
	}
	jpg, err := ReduzirParaJPEG(conteudo, ladoMaximoFoto)
	if err != nil {
		return nil, err
	}
	nome, err := nomeArquivoAleatorio("galeria-", ".jpg")
	if err != nil {
		return nil, err
	}
	url, err := s.armazenamento.Salvar(nome, bytes.NewReader(jpg))
	if err != nil {
		return nil, err
	}
	f := &FotoEvento{EventoID: eventoID, EnviadaPor: usuarioID, URL: url, CriadoEm: time.Now()}
	return f, s.db.Create(f).Error
}

// Excluir: o organizador apaga qualquer foto; o fotógrafo, as dele.
func (s *GaleriaService) Excluir(usuarioID, eventoID, fotoID int64) error {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return ErrSemAcessoGaleria
	}
	var f FotoEvento
	if err := s.db.Where("id = ? AND evento_id = ?", fotoID, eventoID).First(&f).Error; err != nil {
		return ErrFotoNaoEncontrada
	}
	if !s.ehOrganizador(usuarioID, e) && !(f.EnviadaPor == usuarioID && s.podeEnviar(usuarioID, e)) {
		return ErrSemAcessoGaleria
	}
	return s.db.Delete(&f).Error
}

// FotoPublica é a foto na galeria (sem quem pediu remoção, sem e-mail).
type FotoPublica struct {
	ID       int64
	URL      string
	Credito  string
	Marcados []MarcadoFoto
}

type MarcadoFoto struct {
	Nome string
	Slug string
}

// Publica lista as fotos visíveis, com crédito e quem se marcou (só o nome
// artístico de perfis visíveis de adultos).
func (s *GaleriaService) Publica(eventoID int64) ([]FotoPublica, error) {
	var linhas []struct {
		ID      int64
		URL     string `gorm:"column:url"`
		Credito string
	}
	if err := s.db.Table("fotos_evento f").Select("f.id, f.url, u.nome AS credito").
		Joins("JOIN usuarios u ON u.id = f.enviada_por").
		Where("f.evento_id = ? AND NOT f.oculta", eventoID).Order("f.id DESC").Scan(&linhas).Error; err != nil {
		return nil, err
	}
	out := make([]FotoPublica, 0, len(linhas))
	ids := make([]int64, 0, len(linhas))
	for _, l := range linhas {
		out = append(out, FotoPublica{ID: l.ID, URL: l.URL, Credito: l.Credito, Marcados: []MarcadoFoto{}})
		ids = append(ids, l.ID)
	}
	if len(ids) == 0 {
		return out, nil
	}
	var ms []MarcacaoFoto
	s.db.Where("foto_id IN ?", ids).Find(&ms)
	usuarios := map[int64]bool{}
	for _, m := range ms {
		usuarios[m.UsuarioID] = true
	}
	perfis := s.perfisAdultosVisiveis(keys(usuarios))
	idx := map[int64]int{}
	for i, f := range out {
		idx[f.ID] = i
	}
	for _, m := range ms {
		if p, ok := perfis[m.UsuarioID]; ok {
			i := idx[m.FotoID]
			out[i].Marcados = append(out[i].Marcados, p)
		}
	}
	return out, nil
}

func keys(m map[int64]bool) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// perfisAdultosVisiveis: nome artístico e endereço de quem tem perfil
// público de adulto.
func (s *GaleriaService) perfisAdultosVisiveis(usuarioIDs []int64) map[int64]MarcadoFoto {
	out := map[int64]MarcadoFoto{}
	if len(usuarioIDs) == 0 || s.perfis == nil {
		return out
	}
	var ps []PerfilTalento
	s.db.Where("usuario_id IN ?", usuarioIDs).Find(&ps)
	agora := time.Now()
	for i := range ps {
		sit := Situacao(&ps[i], s.perfis.nascimentoEfetivo(&ps[i]), agora)
		if sit.Visivel && !sit.Menor {
			out[ps[i].UsuarioID] = MarcadoFoto{Nome: ps[i].NomeArtistico, Slug: ps[i].Slug}
		}
	}
	return out
}

// adulto: perfil de talento de quem tem 18 anos ou mais (pela data mais
// nova entre perfil e fichas, como no item 8.1).
func (s *GaleriaService) adulto(usuarioID int64) bool {
	if s.perfis == nil {
		return false
	}
	var p PerfilTalento
	if s.db.Where("usuario_id = ?", usuarioID).First(&p).Error != nil {
		return false
	}
	return Idade(s.perfis.nascimentoEfetivo(&p), time.Now()) >= IdadeAdulto
}

func (s *GaleriaService) fotoVisivel(fotoID int64) (*FotoEvento, error) {
	var f FotoEvento
	if err := s.db.Where("id = ? AND NOT oculta", fotoID).First(&f).Error; err != nil {
		return nil, ErrFotoNaoEncontrada
	}
	return &f, nil
}

// Marcar: a própria pessoa (adulta) diz que aparece na foto.
func (s *GaleriaService) Marcar(usuarioID, fotoID int64, marcar bool) error {
	if !marcar {
		return s.db.Where("foto_id = ? AND usuario_id = ?", fotoID, usuarioID).Delete(&MarcacaoFoto{}).Error
	}
	if _, err := s.fotoVisivel(fotoID); err != nil {
		return err
	}
	if !s.adulto(usuarioID) {
		return ErrMarcacaoSoAdulto
	}
	return s.db.Exec(`INSERT INTO marcacoes_foto (foto_id, usuario_id, criado_em) VALUES (?, ?, now()) ON CONFLICT DO NOTHING`, fotoID, usuarioID).Error
}

// MinhasMarcacoes: fotos do evento em que a pessoa se marcou.
func (s *GaleriaService) MinhasMarcacoes(usuarioID, eventoID int64) []int64 {
	var ids []int64
	s.db.Table("marcacoes_foto m").Joins("JOIN fotos_evento f ON f.id = m.foto_id").
		Where("m.usuario_id = ? AND f.evento_id = ?", usuarioID, eventoID).Pluck("m.foto_id", &ids)
	return ids
}

// PedirRemocao: some na hora; o organizador revisa.
func (s *GaleriaService) PedirRemocao(usuarioID *int64, fotoID int64, motivo string) error {
	motivo = strings.TrimSpace(motivo)
	if len([]rune(motivo)) > 500 {
		motivo = string([]rune(motivo)[:500])
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&FotoEvento{}).Where("id = ? AND NOT oculta", fotoID).Update("oculta", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrFotoNaoEncontrada
		}
		return tx.Create(&RemocaoFoto{FotoID: fotoID, UsuarioID: usuarioID, Motivo: motivo, Status: "pendente", CriadoEm: time.Now()}).Error
	})
}

// RemocaoPainel é o que o organizador revisa.
type RemocaoPainel struct {
	ID       int64
	FotoID   int64
	URL      string `gorm:"column:url"`
	Motivo   string
	CriadoEm time.Time
}

type PainelGaleria struct {
	Fotos    []FotoEvento
	Remocoes []RemocaoPainel
}

func (s *GaleriaService) Painel(usuarioID, eventoID int64) (*PainelGaleria, error) {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || !s.podeEnviar(usuarioID, e) {
		return nil, ErrSemAcessoGaleria
	}
	p := &PainelGaleria{}
	q := s.db.Where("evento_id = ?", eventoID)
	if !s.ehOrganizador(usuarioID, e) {
		q = q.Where("enviada_por = ?", usuarioID)
	}
	if err := q.Order("id DESC").Find(&p.Fotos).Error; err != nil {
		return nil, err
	}
	if s.ehOrganizador(usuarioID, e) {
		s.db.Table("remocoes_foto r").Select("r.id, r.foto_id, f.url, r.motivo, r.criado_em").
			Joins("JOIN fotos_evento f ON f.id = r.foto_id").
			Where("f.evento_id = ? AND r.status = 'pendente'", eventoID).Order("r.id").Scan(&p.Remocoes)
	}
	return p, nil
}

// DecidirRemocao: manter (a foto volta) ou remover (apaga a foto).
func (s *GaleriaService) DecidirRemocao(usuarioID, eventoID, remocaoID int64, manter bool) error {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || !s.ehOrganizador(usuarioID, e) {
		return ErrSemAcessoGaleria
	}
	var r RemocaoFoto
	if err := s.db.Table("remocoes_foto").Select("remocoes_foto.*").
		Joins("JOIN fotos_evento f ON f.id = remocoes_foto.foto_id").
		Where("remocoes_foto.id = ? AND f.evento_id = ? AND remocoes_foto.status = 'pendente'", remocaoID, eventoID).First(&r).Error; err != nil {
		return ErrRemocaoNaoEncontra
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if manter {
			if err := tx.Model(&RemocaoFoto{}).Where("foto_id = ? AND status = 'pendente'", r.FotoID).Update("status", "mantida").Error; err != nil {
				return err
			}
			return tx.Model(&FotoEvento{}).Where("id = ?", r.FotoID).Update("oculta", false).Error
		}
		if err := tx.Model(&RemocaoFoto{}).Where("foto_id = ? AND status = 'pendente'", r.FotoID).Update("status", "removida").Error; err != nil {
			return err
		}
		// A linha da remoção fica (registro); a foto e as marcações saem.
		if err := tx.Where("foto_id = ?", r.FotoID).Delete(&MarcacaoFoto{}).Error; err != nil {
			return err
		}
		return tx.Model(&FotoEvento{}).Where("id = ?", r.FotoID).Updates(map[string]any{"oculta": true, "url": ""}).Error
	})
}

// FotoDoPerfil: foto marcada que aparece no perfil do talento.
type FotoDoPerfil struct {
	URL          string `json:"url"`
	EventoTitulo string `json:"evento_titulo"`
	EventoSlug   string `json:"evento_slug"`
}

// FotosDoTalento: fotos visíveis em que o adulto se marcou.
func FotosDoTalento(db *gorm.DB, usuarioID int64) []FotoDoPerfil {
	var out []FotoDoPerfil
	db.Table("marcacoes_foto m").Select("f.url, e.titulo AS evento_titulo, e.slug AS evento_slug").
		Joins("JOIN fotos_evento f ON f.id = m.foto_id").Joins("JOIN eventos e ON e.id = f.evento_id").
		Where("m.usuario_id = ? AND NOT f.oculta AND f.url <> ''", usuarioID).Order("f.id DESC").Limit(60).Scan(&out)
	if out == nil {
		out = []FotoDoPerfil{}
	}
	return out
}

// ---------- fotógrafos ----------

type Fotografo struct {
	UsuarioID int64
	Nome      string
	Email     string
}

func (s *GaleriaService) orgDoEvento(organizadorID, eventoID int64) error {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != organizadorID {
		return ErrEventoNaoPertenceAoOrganizador
	}
	return nil
}

// AdicionarFotografo: pelo e-mail de quem já tem conta (como a equipe).
func (s *GaleriaService) AdicionarFotografo(organizadorID, eventoID int64, email string) (*Fotografo, error) {
	if err := s.orgDoEvento(organizadorID, eventoID); err != nil {
		return nil, err
	}
	u, err := s.usuarios.BuscarPorEmail(strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return nil, ErrUsuarioNaoEncontrado
	}
	err = s.db.Exec(`INSERT INTO papeis_evento (evento_id, usuario_id, papel, origem, status, criado_em)
		VALUES (?, ?, 'fotografo', 'manual', 'confirmado', now())
		ON CONFLICT (evento_id, usuario_id, papel) DO UPDATE SET status = 'confirmado'`, eventoID, u.ID).Error
	if err != nil {
		return nil, err
	}
	return &Fotografo{UsuarioID: u.ID, Nome: u.Nome, Email: u.Email}, nil
}

func (s *GaleriaService) Fotografos(organizadorID, eventoID int64) ([]Fotografo, error) {
	if err := s.orgDoEvento(organizadorID, eventoID); err != nil {
		return nil, err
	}
	var out []Fotografo
	err := s.db.Table("papeis_evento p").Select("u.id AS usuario_id, u.nome, u.email").
		Joins("JOIN usuarios u ON u.id = p.usuario_id").
		Where("p.evento_id = ? AND p.papel = 'fotografo' AND p.status = 'confirmado'", eventoID).Order("u.nome").Scan(&out).Error
	return out, err
}

func (s *GaleriaService) RemoverFotografo(organizadorID, eventoID, usuarioID int64) error {
	if err := s.orgDoEvento(organizadorID, eventoID); err != nil {
		return err
	}
	return s.db.Model(&domain.PapelEvento{}).Where("evento_id = ? AND usuario_id = ? AND papel = 'fotografo'", eventoID, usuarioID).
		Update("status", domain.StatusPapelRemovido).Error
}

func nomeArquivoAleatorio(prefixo, extensao string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefixo + hex.EncodeToString(buf) + extensao, nil
}
