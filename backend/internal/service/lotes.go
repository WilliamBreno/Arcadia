package service

import (
	"sort"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// loteDisponivel: ativo, dentro da janela de vendas (virada por data) e
// com estoque (virada por quantidade).
func loteDisponivel(t *domain.TipoIngresso, vendidos int64, agora time.Time) bool {
	if !t.Ativo {
		return false
	}
	if t.VendasInicio != nil && agora.Before(*t.VendasInicio) {
		return false
	}
	if t.VendasFim != nil && agora.After(*t.VendasFim) {
		return false
	}
	return vendidos < int64(t.Quantidade)
}

// LoteAtual devolve o id do lote vendável de um grupo (o primeiro
// disponível por ordem, id) ou 0 se nenhum estiver.
func LoteAtual(grupo []domain.TipoIngresso, vendidos map[int64]int64, agora time.Time) int64 {
	ordenados := append([]domain.TipoIngresso(nil), grupo...)
	sort.SliceStable(ordenados, func(i, j int) bool {
		if ordenados[i].Ordem != ordenados[j].Ordem {
			return ordenados[i].Ordem < ordenados[j].Ordem
		}
		return ordenados[i].ID < ordenados[j].ID
	})
	for i := range ordenados {
		if loteDisponivel(&ordenados[i], vendidos[ordenados[i].ID], agora) {
			return ordenados[i].ID
		}
	}
	return 0
}

// FiltrarLotes mantém os tipos sem lote_grupo e, para cada grupo, só o lote
// atual (a "virada automática" da seção 2.4 do checklist).
func FiltrarLotes(tipos []domain.TipoIngresso, vendidos map[int64]int64, agora time.Time) []domain.TipoIngresso {
	grupos := map[string][]domain.TipoIngresso{}
	for _, t := range tipos {
		if t.LoteGrupo != "" {
			grupos[t.LoteGrupo] = append(grupos[t.LoteGrupo], t)
		}
	}
	atual := map[string]int64{}
	for g, lista := range grupos {
		atual[g] = LoteAtual(lista, vendidos, agora)
	}

	var resultado []domain.TipoIngresso
	for _, t := range tipos {
		if t.LoteGrupo == "" || atual[t.LoteGrupo] == t.ID {
			resultado = append(resultado, t)
		}
	}
	return resultado
}

// calcularDesconto aplica o cupom ao preço do ingresso; nunca passa do preço.
func calcularDesconto(tipo domain.TipoCupom, valor, precoCentavos int64) int64 {
	var desconto int64
	if tipo == domain.CupomPercentual {
		desconto = precoCentavos * valor / 100
	} else {
		desconto = valor
	}
	if desconto > precoCentavos {
		desconto = precoCentavos
	}
	return desconto
}
