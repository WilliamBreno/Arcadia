package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestLoteAtualViradaPorQuantidadeEData(t *testing.T) {
	agora := time.Now()
	futuro := agora.Add(24 * time.Hour)
	lotes := []domain.TipoIngresso{
		{ID: 1, Ordem: 1, Quantidade: 10, Ativo: true, LoteGrupo: "inteira"},
		{ID: 2, Ordem: 2, Quantidade: 10, Ativo: true, LoteGrupo: "inteira"},
		{ID: 3, Ordem: 3, Quantidade: 10, Ativo: true, LoteGrupo: "inteira", VendasInicio: &futuro},
	}

	if got := LoteAtual(lotes, map[int64]int64{}, agora); got != 1 {
		t.Errorf("com estoque, lote 1; got %d", got)
	}
	if got := LoteAtual(lotes, map[int64]int64{1: 10}, agora); got != 2 {
		t.Errorf("lote 1 esgotado, vira para 2; got %d", got)
	}
	if got := LoteAtual(lotes, map[int64]int64{1: 10, 2: 10}, agora); got != 0 {
		t.Errorf("lote 3 ainda não abriu; got %d", got)
	}
	if got := LoteAtual(lotes, map[int64]int64{1: 10, 2: 10}, futuro.Add(time.Minute)); got != 3 {
		t.Errorf("depois da data, lote 3; got %d", got)
	}
}

func TestFiltrarLotesMantemTiposSemGrupo(t *testing.T) {
	tipos := []domain.TipoIngresso{
		{ID: 1, Ordem: 1, Quantidade: 5, Ativo: true, LoteGrupo: "a"},
		{ID: 2, Ordem: 2, Quantidade: 5, Ativo: true, LoteGrupo: "a"},
		{ID: 3, Quantidade: 5, Ativo: true},
	}
	got := FiltrarLotes(tipos, map[int64]int64{}, time.Now())
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 3 {
		t.Errorf("esperava [1 3], got %+v", got)
	}
}

func TestCalcularDesconto(t *testing.T) {
	if got := calcularDesconto(domain.CupomPercentual, 10, 3000); got != 300 {
		t.Errorf("10%% de 3000 = 300; got %d", got)
	}
	if got := calcularDesconto(domain.CupomValor, 500, 3000); got != 500 {
		t.Errorf("valor 500; got %d", got)
	}
	if got := calcularDesconto(domain.CupomValor, 5000, 3000); got != 3000 {
		t.Errorf("desconto nunca passa do preço; got %d", got)
	}
}
