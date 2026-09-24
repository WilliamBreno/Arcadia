package service

import (
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestValidarCotaMeia(t *testing.T) {
	inteira := domain.TipoIngresso{Quantidade: 60, Ativo: true}
	meiaOK := domain.TipoIngresso{Quantidade: 40, Ativo: true, MeiaEntrada: true}
	meiaAcima := domain.TipoIngresso{Quantidade: 41, Ativo: true, MeiaEntrada: true}
	meiaInativa := domain.TipoIngresso{Quantidade: 500, Ativo: false, MeiaEntrada: true}

	if err := validarCotaMeia([]domain.TipoIngresso{inteira, meiaOK}); err != nil {
		t.Errorf("40 de 100 (exatamente 40%%) deve passar: %v", err)
	}
	if validarCotaMeia([]domain.TipoIngresso{inteira, meiaAcima}) == nil {
		t.Error("41 de 101 (>40%) deve falhar")
	}
	if err := validarCotaMeia([]domain.TipoIngresso{inteira, meiaInativa}); err != nil {
		t.Errorf("tipo inativo não conta: %v", err)
	}
	if err := validarCotaMeia(nil); err != nil {
		t.Errorf("sem tipos passa: %v", err)
	}
}
