package service

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrOrganizadorJaExiste  = errors.New("usuário já tem perfil de organizador")
	ErrOrganizadorNaoExiste = errors.New("perfil de organizador não encontrado")
	ErrDocumentoInvalido    = errors.New("documento inválido para o tipo de pessoa informado")
)

type OrganizadorService struct {
	organizadores *repository.OrganizadorRepository
}

func NovoOrganizadorService(organizadores *repository.OrganizadorRepository) *OrganizadorService {
	return &OrganizadorService{organizadores: organizadores}
}

func (s *OrganizadorService) CriarPerfil(usuarioID int64, nomePublico string, tipoPessoa domain.TipoPessoa, documento string) (*domain.Organizador, error) {
	if _, err := s.organizadores.BuscarPorUsuarioID(usuarioID); err == nil {
		return nil, ErrOrganizadorJaExiste
	}

	if !documentoValido(tipoPessoa, documento) {
		return nil, ErrDocumentoInvalido
	}

	slug, err := s.gerarSlugUnico(nomePublico)
	if err != nil {
		return nil, err
	}

	organizador := &domain.Organizador{
		UsuarioID:   usuarioID,
		NomePublico: nomePublico,
		Slug:        slug,
		TipoPessoa:  tipoPessoa,
		Documento:   somenteDigitos(documento),
		Status:      domain.StatusOrganizadorPendente,
		CriadoEm:    time.Now(),
	}
	if err := s.organizadores.Criar(organizador); err != nil {
		return nil, err
	}
	return organizador, nil
}

type AtualizarPerfilDados struct {
	NomePublico  string
	Descricao    string
	LogoURL      string
	ChavePix     string
	TipoChavePix string
	Instagram    string
	Site         string
}

func (s *OrganizadorService) AtualizarPerfil(usuarioID int64, dados AtualizarPerfilDados) (*domain.Organizador, error) {
	organizador, err := s.organizadores.BuscarPorUsuarioID(usuarioID)
	if err != nil {
		return nil, ErrOrganizadorNaoExiste
	}

	organizador.NomePublico = dados.NomePublico
	organizador.Descricao = dados.Descricao
	organizador.LogoURL = dados.LogoURL
	organizador.ChavePix = dados.ChavePix
	organizador.TipoChavePix = dados.TipoChavePix
	organizador.Instagram = dados.Instagram
	organizador.Site = dados.Site

	if err := s.organizadores.Salvar(organizador); err != nil {
		return nil, err
	}
	return organizador, nil
}

func (s *OrganizadorService) gerarSlugUnico(nome string) (string, error) {
	base := slugify(nome)
	slug := base
	for sufixo := 2; ; sufixo++ {
		existe, err := s.organizadores.SlugExiste(slug)
		if err != nil {
			return "", err
		}
		if !existe {
			return slug, nil
		}
		slug = base + "-" + strconv.Itoa(sufixo)
	}
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var semAcento strings.Builder
	for _, r := range s {
		switch {
		case strings.ContainsRune("áàâãä", r):
			semAcento.WriteRune('a')
		case strings.ContainsRune("éèêë", r):
			semAcento.WriteRune('e')
		case strings.ContainsRune("íìîï", r):
			semAcento.WriteRune('i')
		case strings.ContainsRune("óòôõö", r):
			semAcento.WriteRune('o')
		case strings.ContainsRune("úùûü", r):
			semAcento.WriteRune('u')
		case r == 'ç':
			semAcento.WriteRune('c')
		default:
			semAcento.WriteRune(r)
		}
	}

	naoAlfanumerico := regexp.MustCompile(`[^a-z0-9]+`)
	slug := naoAlfanumerico.ReplaceAllString(semAcento.String(), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "organizador"
	}
	return slug
}

func somenteDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// documentoValido faz uma checagem simples de tamanho (11 dígitos para
// CPF, 14 para CNPJ) — sem validação de dígito verificador. Suficiente
// para o MVP; se precisar de validação forte, revisar aqui.
func documentoValido(tipo domain.TipoPessoa, documento string) bool {
	digitos := somenteDigitos(documento)
	switch tipo {
	case domain.TipoPessoaFisica:
		return len(digitos) == 11
	case domain.TipoPessoaJuridica:
		return len(digitos) == 14
	default:
		return false
	}
}
