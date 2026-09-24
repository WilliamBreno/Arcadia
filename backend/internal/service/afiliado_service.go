package service

import (
	"crypto/rand"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var ErrAfiliadoNaoEncontrado = errors.New("afiliado não encontrado")

type AfiliadoService struct {
	eventos   *EventoService
	afiliados *repository.AfiliadoRepository
}

func NovoAfiliadoService(e *EventoService, a *repository.AfiliadoRepository) *AfiliadoService {
	return &AfiliadoService{eventos: e, afiliados: a}
}

func gerarCodigoAfiliado() (string, error) {
	const alfabeto = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = alfabeto[int(b)%len(alfabeto)]
	}
	return string(buf), nil
}

func (s *AfiliadoService) Listar(organizadorID, eventoID int64) ([]repository.AfiliadoComStats, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.afiliados.ListarComStats(eventoID)
}

func (s *AfiliadoService) Criar(organizadorID, eventoID int64, nome string) (*domain.Afiliado, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return nil, errors.New("informe o nome do divulgador")
	}
	codigo, err := gerarCodigoAfiliado()
	if err != nil {
		return nil, err
	}
	a := &domain.Afiliado{EventoID: eventoID, Nome: nome, Codigo: codigo, Ativo: true, CriadoEm: time.Now()}
	return a, s.afiliados.Criar(a)
}

func (s *AfiliadoService) Desativar(organizadorID, eventoID, afiliadoID int64) error {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	a, err := s.afiliados.BuscarPorID(afiliadoID)
	if err != nil || a.EventoID != eventoID {
		return ErrAfiliadoNaoEncontrado
	}
	a.Ativo = false
	return s.afiliados.Salvar(a)
}

var padraoCorTema = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// corTemaOuPadrao valida o formato #RRGGBB (o valor vira CSS no frontend,
// então nunca aceitar texto livre); inválido é ignorado e mantém o atual.
func corTemaOuPadrao(v *string, atual string) string {
	if v == nil {
		return atual
	}
	if *v == "" || padraoCorTema.MatchString(*v) {
		return strings.ToLower(*v)
	}
	return atual
}
