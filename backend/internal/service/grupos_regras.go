package service

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Item 11.1: regras do perfil de grupo e de artista solo (sem banco, testadas).

var (
	ErrArtistaInvalido      = errors.New("dados do grupo inválidos")
	ErrArtistaNaoEncontrado = errors.New("grupo não encontrado")
	ErrArtistaSemPermissao  = errors.New("só o líder pode fazer isso")
	ErrLimiteArtistas       = errors.New("cada conta lidera até 5 grupos ou perfis solo")
	ErrConviteArtista       = errors.New("convite inválido ou já usado")
	ErrConviteOutroEmail    = errors.New("este convite foi enviado para outro e-mail; entre com a conta desse e-mail")
	ErrIntegranteInvalido   = errors.New("dados do integrante inválidos")
	ErrIntegranteDuplicado  = errors.New("esta pessoa já está no grupo ou foi convidada")
	ErrLimiteIntegrantes    = errors.New("o grupo chegou ao limite de 60 integrantes")
	ErrMidiaInvalida        = errors.New("mídia inválida")
	ErrLimiteMidias         = errors.New("o portfólio chegou ao limite de 40 fotos e vídeos")
	ErrLiderNaoSai          = errors.New("o líder não pode sair do grupo; passe a liderança antes")
	ErrSoloUmIntegrante     = errors.New("perfil solo tem só você como integrante")
)

const (
	MaxGruposPorLider    = 5
	MaxIntegrantesGrupo  = 60
	MaxMidiasGrupo       = 40
	TamanhoCodigoInsta   = 8
	alfabetoCodigoInsta  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	prefixoCodigoInsta   = "EVVE-"
	maxBioGrupo          = 1500
	maxNomeGrupo         = 80
	maxLegendaMidia      = 140
	minAnoExperiencia    = 1950
	tiposGrupoPermitidos = "danca cosplay canto teatro outro"
)

var TiposGrupo = map[string]string{
	"danca":   "Dança",
	"cosplay": "Cosplay",
	"canto":   "Canto e música",
	"teatro":  "Teatro e atuação",
	"outro":   "Outro",
}

// DadosGrupo é o que o líder envia.
type DadosGrupo struct {
	Nome      string `json:"nome"`
	Tipo      string `json:"tipo"`
	Solo      bool   `json:"solo"`
	Bio       string `json:"bio"`
	Cidade    string `json:"cidade"`
	UF        string `json:"uf"`
	Instagram string `json:"instagram"`
	TikTok    string `json:"tiktok"`
	YouTube   string `json:"youtube"`
	LogoURL   string `json:"logo_url"`
	CapaURL   string `json:"capa_url"`
	Publico   *bool  `json:"publico"`
}

func urlImagemValida(u string) bool {
	return u == "" || strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// NormalizarGrupo valida e limpa. Testada.
func NormalizarGrupo(d DadosGrupo) (DadosGrupo, error) {
	d.Nome = limpar(d.Nome)
	d.Tipo = strings.TrimSpace(d.Tipo)
	d.Bio = strings.TrimSpace(d.Bio)
	d.Cidade = limpar(d.Cidade)
	d.UF = strings.ToUpper(strings.TrimSpace(d.UF))
	switch {
	case utf8.RuneCountInString(d.Nome) < 2 || utf8.RuneCountInString(d.Nome) > maxNomeGrupo:
		return d, fmt.Errorf("%w: o nome tem de 2 a 80 caracteres", ErrArtistaInvalido)
	case TiposGrupo[d.Tipo] == "":
		return d, fmt.Errorf("%w: escolha o tipo", ErrArtistaInvalido)
	case utf8.RuneCountInString(d.Bio) > maxBioGrupo:
		return d, fmt.Errorf("%w: a apresentação tem no máximo 1.500 caracteres", ErrArtistaInvalido)
	case d.UF != "" && !reUF.MatchString(d.UF):
		return d, fmt.Errorf("%w: UF com 2 letras", ErrArtistaInvalido)
	case utf8.RuneCountInString(d.Cidade) > 80:
		return d, fmt.Errorf("%w: cidade muito longa", ErrArtistaInvalido)
	case !urlImagemValida(d.LogoURL) || !urlImagemValida(d.CapaURL):
		return d, fmt.Errorf("%w: imagem inválida", ErrArtistaInvalido)
	}
	for _, h := range []*string{&d.Instagram, &d.TikTok, &d.YouTube} {
		*h = limparHandle(*h)
		if *h != "" && !reHandle.MatchString(*h) {
			return d, fmt.Errorf("%w: use só o @ das redes (letras, números, ponto e _)", ErrArtistaInvalido)
		}
	}
	return d, nil
}

// DadosIntegrante: o que o líder informa ao convidar.
type DadosIntegrante struct {
	Email         string `json:"email"`
	NomeArtistico string `json:"nome_artistico"`
	Funcao        string `json:"funcao"`
	DesdeAno      *int   `json:"desde_ano"`
}

// NormalizarIntegrante valida o convite. Testada.
func NormalizarIntegrante(d DadosIntegrante, agora time.Time) (DadosIntegrante, error) {
	d.Email = NormalizarEmail(d.Email)
	d.NomeArtistico = limpar(d.NomeArtistico)
	d.Funcao = limpar(d.Funcao)
	switch {
	case !strings.Contains(d.Email, "@") || !strings.Contains(d.Email[strings.Index(d.Email, "@"):], ".") || len(d.Email) > 255:
		return d, fmt.Errorf("%w: e-mail inválido", ErrIntegranteInvalido)
	case utf8.RuneCountInString(d.NomeArtistico) < 2 || utf8.RuneCountInString(d.NomeArtistico) > 60:
		return d, fmt.Errorf("%w: o nome tem de 2 a 60 caracteres", ErrIntegranteInvalido)
	case utf8.RuneCountInString(d.Funcao) > 60:
		return d, fmt.Errorf("%w: a função tem no máximo 60 caracteres", ErrIntegranteInvalido)
	case d.DesdeAno != nil && (*d.DesdeAno < minAnoExperiencia || *d.DesdeAno > agora.Year()):
		return d, fmt.Errorf("%w: ano de início inválido", ErrIntegranteInvalido)
	}
	return d, nil
}

// AnosDeExperiencia conta os anos desde o ano de início (mínimo 1). Testada.
func AnosDeExperiencia(desde *int, agora time.Time) int {
	if desde == nil {
		return 0
	}
	return max(agora.Year()-*desde, 1)
}

// SituacaoIntegrante diz o que aparece do integrante no perfil público,
// pelas regras do item 8.1. Testada.
type SituacaoIntegrante struct {
	Visivel         bool
	Anonimo         bool // menor de 12: aparece só como "Integrante"
	MostraFoto      bool
	MostraInstagram bool
	Menor           bool
	Pendente        string // por que ainda não aparece (para o líder e para a pessoa)
}

func CalcularSituacaoIntegrante(status string, nascimento *time.Time, autorizado bool, autorizaFoto bool, temFoto bool, agora time.Time) SituacaoIntegrante {
	s := SituacaoIntegrante{}
	switch {
	case status == "convidado":
		s.Pendente = "Aguardando a pessoa aceitar o convite."
		return s
	case status != "ativo":
		return s
	case nascimento == nil:
		s.Pendente = "Falta a data de nascimento."
		return s
	}
	idade := Idade(*nascimento, agora)
	s.Menor = idade < IdadeAdulto
	switch {
	case idade < IdadeMinimaPerfil:
		s.Visivel, s.Anonimo = true, true
	case s.Menor && !autorizado:
		s.Pendente = "Aguardando a autorização do responsável pelo e-mail."
	case s.Menor:
		s.Visivel = true
		s.MostraFoto = temFoto && autorizaFoto
	default:
		s.Visivel, s.MostraFoto, s.MostraInstagram = true, temFoto, true
	}
	return s
}

var (
	reYouTubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
)

// VideoNormalizado é um vídeo por link aceito no portfólio.
type VideoNormalizado struct {
	URL        string // link canônico para abrir
	Embed      string // endereço para incorporar (só YouTube)
	Plataforma string
}

// NormalizarVideo aceita links do YouTube, Instagram e TikTok. Testada.
func NormalizarVideo(bruto string) (VideoNormalizado, error) {
	u, err := url.Parse(strings.TrimSpace(bruto))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return VideoNormalizado{}, fmt.Errorf("%w: cole o link do vídeo", ErrMidiaInvalida)
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	host = strings.TrimPrefix(host, "m.")
	caminho := strings.Trim(u.Path, "/")
	switch host {
	case "youtube.com", "youtu.be", "music.youtube.com":
		id := ""
		switch {
		case host == "youtu.be":
			id = strings.Split(caminho, "/")[0]
		case caminho == "watch":
			id = u.Query().Get("v")
		case strings.HasPrefix(caminho, "shorts/"), strings.HasPrefix(caminho, "embed/"), strings.HasPrefix(caminho, "live/"):
			id = strings.Split(caminho, "/")[1]
		}
		if !reYouTubeID.MatchString(id) {
			return VideoNormalizado{}, fmt.Errorf("%w: link do YouTube sem o vídeo", ErrMidiaInvalida)
		}
		return VideoNormalizado{URL: "https://www.youtube.com/watch?v=" + id, Embed: "https://www.youtube-nocookie.com/embed/" + id, Plataforma: "youtube"}, nil
	case "instagram.com":
		partes := strings.Split(caminho, "/")
		if len(partes) >= 2 && (partes[0] == "reel" || partes[0] == "p" || partes[0] == "tv") && partes[1] != "" {
			return VideoNormalizado{URL: "https://www.instagram.com/" + partes[0] + "/" + partes[1] + "/", Plataforma: "instagram"}, nil
		}
	case "tiktok.com", "vm.tiktok.com":
		if caminho != "" {
			return VideoNormalizado{URL: "https://" + host + "/" + caminho, Plataforma: "tiktok"}, nil
		}
	}
	return VideoNormalizado{}, fmt.Errorf("%w: use um link do YouTube, Instagram ou TikTok", ErrMidiaInvalida)
}

// EmbedDoVideo devolve o endereço de incorporação de um link já salvo.
func EmbedDoVideo(link string) (embed, plataforma string) {
	v, err := NormalizarVideo(link)
	if err != nil {
		return "", ""
	}
	return v.Embed, v.Plataforma
}

// CodigoInstagram gera o código que o líder põe na bio.
func CodigoInstagram(aleatorio []byte) string {
	var b strings.Builder
	b.WriteString(prefixoCodigoInsta)
	for i := 0; i < TamanhoCodigoInsta && i < len(aleatorio); i++ {
		b.WriteByte(alfabetoCodigoInsta[int(aleatorio[i])%len(alfabetoCodigoInsta)])
	}
	return b.String()
}
