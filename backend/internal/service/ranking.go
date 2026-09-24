package service

import (
	"math"
	"sort"
	"strings"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// EntradaRanking é uma ficha com o que o ranking precisa saber dela.
type EntradaRanking struct {
	FichaID          int64
	Nome             string
	NomeArtistico    string
	TipoApresentacao string
	Menor            bool
}

type LinhaRanking struct {
	Posicao       int
	FichaID       int64
	Nome          string
	NomeArtistico string
	Menor         bool
	Tipo          string
	NotaFinal     float64 // 0 a 10
	Jurados       int
	PorCriterio   map[int64]float64 // média (0-10) entre jurados, por critério
}

// normalizar leva a nota para a escala 0-10, para critérios com escalas
// diferentes poderem ser ponderados juntos.
func normalizar(nota float64, c *domain.CriterioAvaliacao) float64 {
	return (nota - c.NotaMin) / (c.NotaMax - c.NotaMin) * 10
}

func criterioAplica(c *domain.CriterioAvaliacao, tipo string) bool {
	return c.TipoApresentacao == nil || string(*c.TipoApresentacao) == tipo
}

// arredondar2 evita empates falsos/ruído de ponto flutuante nas comparações.
func arredondar2(v float64) float64 {
	return math.Round(v*100) / 100
}

// CalcularRanking (seção 3.3): só entram avaliações FINALIZADAS.
//   - nota do jurado numa ficha = média ponderada (por peso) dos critérios
//     aplicáveis, cada nota normalizada para 0-10;
//   - nota final = média das notas dos jurados que finalizaram;
//   - ranking separado por tipo de apresentação (cosplay não compete com dança);
//   - desempate: maior média no critério de maior peso, depois no seguinte
//     (peso desc, ordem, id); se ainda empatar, dividem a mesma posição.
//
// Fichas sem nenhuma avaliação finalizada ficam de fora.
func CalcularRanking(criterios []domain.CriterioAvaliacao, fichas []EntradaRanking, avaliacoes []domain.Avaliacao) map[string][]LinhaRanking {
	crit := map[int64]*domain.CriterioAvaliacao{}
	for i := range criterios {
		crit[criterios[i].ID] = &criterios[i]
	}

	// ficha -> jurado -> critério -> nota normalizada
	notas := map[int64]map[int64]map[int64]float64{}
	for _, a := range avaliacoes {
		c, ok := crit[a.CriterioID]
		if !ok || !a.Finalizada {
			continue
		}
		if notas[a.FichaID] == nil {
			notas[a.FichaID] = map[int64]map[int64]float64{}
		}
		if notas[a.FichaID][a.JuradoUsuarioID] == nil {
			notas[a.FichaID][a.JuradoUsuarioID] = map[int64]float64{}
		}
		notas[a.FichaID][a.JuradoUsuarioID][a.CriterioID] = normalizar(a.Nota, c)
	}

	linhasPorTipo := map[string][]LinhaRanking{}
	for _, f := range fichas {
		porJurado := notas[f.FichaID]
		if len(porJurado) == 0 {
			continue
		}

		var somaJurados float64
		somaCriterio := map[int64]float64{}
		contCriterio := map[int64]int{}
		for _, porCriterio := range porJurado {
			var soma, pesos float64
			for cid, n := range porCriterio {
				c := crit[cid]
				if !criterioAplica(c, f.TipoApresentacao) {
					continue
				}
				soma += n * c.Peso
				pesos += c.Peso
				somaCriterio[cid] += n
				contCriterio[cid]++
			}
			if pesos > 0 {
				somaJurados += soma / pesos
			}
		}

		medias := map[int64]float64{}
		for cid, s := range somaCriterio {
			medias[cid] = arredondar2(s / float64(contCriterio[cid]))
		}
		linhasPorTipo[f.TipoApresentacao] = append(linhasPorTipo[f.TipoApresentacao], LinhaRanking{
			FichaID: f.FichaID, Nome: f.Nome, NomeArtistico: f.NomeArtistico, Menor: f.Menor, Tipo: f.TipoApresentacao,
			NotaFinal: arredondar2(somaJurados / float64(len(porJurado))), Jurados: len(porJurado), PorCriterio: medias,
		})
	}

	ordemDesempate := append([]domain.CriterioAvaliacao(nil), criterios...)
	sort.SliceStable(ordemDesempate, func(i, j int) bool {
		if ordemDesempate[i].Peso != ordemDesempate[j].Peso {
			return ordemDesempate[i].Peso > ordemDesempate[j].Peso
		}
		if ordemDesempate[i].Ordem != ordemDesempate[j].Ordem {
			return ordemDesempate[i].Ordem < ordemDesempate[j].Ordem
		}
		return ordemDesempate[i].ID < ordemDesempate[j].ID
	})

	compara := func(a, b LinhaRanking) int { // >0 se a vence b
		if a.NotaFinal != b.NotaFinal {
			if a.NotaFinal > b.NotaFinal {
				return 1
			}
			return -1
		}
		for _, c := range ordemDesempate {
			if !criterioAplica(&c, a.Tipo) {
				continue
			}
			va, vb := a.PorCriterio[c.ID], b.PorCriterio[c.ID]
			if va != vb {
				if va > vb {
					return 1
				}
				return -1
			}
		}
		return 0
	}

	for tipo, linhas := range linhasPorTipo {
		sort.SliceStable(linhas, func(i, j int) bool {
			if c := compara(linhas[i], linhas[j]); c != 0 {
				return c > 0
			}
			return linhas[i].FichaID < linhas[j].FichaID
		})
		for i := range linhas {
			if i > 0 && compara(linhas[i-1], linhas[i]) == 0 {
				linhas[i].Posicao = linhas[i-1].Posicao
			} else {
				linhas[i].Posicao = i + 1
			}
		}
		linhasPorTipo[tipo] = linhas
	}
	return linhasPorTipo
}

// notaValida confere a nota contra a escala do critério (limites e passo).
func notaValida(nota float64, c *domain.CriterioAvaliacao) bool {
	if nota < c.NotaMin-1e-9 || nota > c.NotaMax+1e-9 {
		return false
	}
	passos := (nota - c.NotaMin) / c.Passo
	return math.Abs(passos-math.Round(passos)) < 1e-6
}

// NomePublico é o nome exibido no resultado PÚBLICO (dado pessoal): nome
// artístico se houver; senão primeiro nome + inicial do sobrenome; menor de
// idade nunca aparece com sobrenome. O organizador vê o nome completo.
func NomePublico(l LinhaRanking) string {
	if l.NomeArtistico != "" {
		return l.NomeArtistico
	}
	partes := strings.Fields(l.Nome)
	if len(partes) == 0 {
		return "Participante"
	}
	if l.Menor || len(partes) == 1 {
		return partes[0]
	}
	return partes[0] + " " + string([]rune(partes[len(partes)-1])[0]) + "."
}
