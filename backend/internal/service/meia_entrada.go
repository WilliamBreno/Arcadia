package service

import (
	"errors"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// CotaMeiaPercentual é o teto legal da Lei 12.933/2013 (seção 7.9).
const CotaMeiaPercentual = 40

var ErrCotaMeiaExcedida = errors.New("meia-entrada não pode passar de 40% do total de ingressos do evento (Lei 12.933/2013)")

// validarCotaMeia soma só tipos ativos: a quantidade de meias tem de ser no
// máximo 40% da quantidade total (meias + inteiras). Idosos não têm cota,
// mas o modelo trata só "meia" como tipo marcado (ver seção 14).
func validarCotaMeia(tipos []domain.TipoIngresso) error {
	var total, meia int64
	for _, t := range tipos {
		if !t.Ativo {
			continue
		}
		total += int64(t.Quantidade)
		if t.MeiaEntrada {
			meia += int64(t.Quantidade)
		}
	}
	if meia*100 > total*CotaMeiaPercentual {
		return ErrCotaMeiaExcedida
	}
	return nil
}

// checarCotaMeia valida a lista do evento com o candidato aplicado (novo ou
// alterado) antes de gravar.
func (s *TipoIngressoService) checarCotaMeia(eventoID int64, candidato *domain.TipoIngresso) error {
	tipos, err := s.tipos.ListarPorEvento(eventoID)
	if err != nil {
		return err
	}
	substituido := false
	for i := range tipos {
		if tipos[i].ID == candidato.ID && candidato.ID != 0 {
			tipos[i] = *candidato
			substituido = true
		}
	}
	if !substituido {
		tipos = append(tipos, *candidato)
	}
	return validarCotaMeia(tipos)
}
