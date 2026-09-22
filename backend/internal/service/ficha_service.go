package service

import (
	"errors"
	"time"

	"gorm.io/datatypes"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrFichaNaoEncontrada  = errors.New("ficha não encontrada")
	ErrMenorSemAutorizacao = errors.New("menor de 18 anos precisa de nome, contato e autorização do responsável")
)

type FichaDados struct {
	Nome                      string
	NomeArtistico             string
	Instagram                 string
	DataNascimento            *time.Time
	FotoURL                   string
	Telefone                  string
	TipoApresentacao          *domain.TipoApresentacao
	Dados                     datatypes.JSON
	ResponsavelNome           string
	ResponsavelContato        string
	AutorizacaoResponsavelURL string
}

func aplicarDadosFicha(f *domain.FichaParticipacao, dados FichaDados) {
	f.Nome = dados.Nome
	f.NomeArtistico = dados.NomeArtistico
	f.Instagram = dados.Instagram
	f.DataNascimento = dados.DataNascimento
	f.FotoURL = dados.FotoURL
	f.Telefone = dados.Telefone
	f.TipoApresentacao = dados.TipoApresentacao
	if len(dados.Dados) == 0 {
		f.Dados = datatypes.JSON("{}")
	} else {
		f.Dados = dados.Dados
	}
	f.ResponsavelNome = dados.ResponsavelNome
	f.ResponsavelContato = dados.ResponsavelContato
	f.AutorizacaoResponsavelURL = dados.AutorizacaoResponsavelURL
}

// validarMenorDeIdade aplica a regra da seção 7.12: menor de 18 (por
// data_nascimento) precisa de responsável e autorização.
func validarMenorDeIdade(f *domain.FichaParticipacao) error {
	if !f.MenorDeIdade() {
		return nil
	}
	if f.ResponsavelNome == "" || f.ResponsavelContato == "" || f.AutorizacaoResponsavelURL == "" {
		return ErrMenorSemAutorizacao
	}
	return nil
}

type FichaService struct {
	fichas *repository.FichaParticipacaoRepository
}

func NovoFichaService(fichas *repository.FichaParticipacaoRepository) *FichaService {
	return &FichaService{fichas: fichas}
}

func (s *FichaService) ObterMinha(eventoID, usuarioID int64, papel domain.Papel) (*domain.FichaParticipacao, error) {
	ficha, err := s.fichas.Buscar(eventoID, usuarioID, papel)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}
	return ficha, nil
}

// AtualizarMinha só é permitida enquanto a ficha ainda pode ser editada
// (rascunho/pendente/aprovado) — depois de rejeitada/desistiu o
// organizador decide o próximo passo, não o próprio usuário editando
// livremente.
func (s *FichaService) AtualizarMinha(eventoID, usuarioID int64, papel domain.Papel, dados FichaDados) (*domain.FichaParticipacao, error) {
	ficha, err := s.fichas.Buscar(eventoID, usuarioID, papel)
	if err != nil {
		return nil, ErrFichaNaoEncontrada
	}

	aplicarDadosFicha(ficha, dados)
	if err := validarMenorDeIdade(ficha); err != nil {
		return nil, err
	}

	if err := s.fichas.Salvar(ficha); err != nil {
		return nil, err
	}
	return ficha, nil
}
