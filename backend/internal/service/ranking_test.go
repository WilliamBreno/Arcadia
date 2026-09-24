package service

import (
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func criterios() []domain.CriterioAvaliacao {
	return []domain.CriterioAvaliacao{
		{ID: 1, Nome: "Fidelidade", Peso: 3, NotaMin: 0, NotaMax: 10, Passo: 1, Ordem: 1},
		{ID: 2, Nome: "Performance", Peso: 1, NotaMin: 0, NotaMax: 10, Passo: 1, Ordem: 2},
	}
}

func aval(ficha, jurado, criterio int64, nota float64) domain.Avaliacao {
	return domain.Avaliacao{FichaID: ficha, JuradoUsuarioID: jurado, CriterioID: criterio, Nota: nota, Finalizada: true}
}

func TestCalcularRankingPesosEPosicoes(t *testing.T) {
	fichas := []EntradaRanking{{FichaID: 10, Nome: "A", TipoApresentacao: "cosplay"}, {FichaID: 11, Nome: "B", TipoApresentacao: "cosplay"}}
	avs := []domain.Avaliacao{
		aval(10, 1, 1, 10), aval(10, 1, 2, 0), // (10*3+0*1)/4 = 7.5
		aval(11, 1, 1, 6), aval(11, 1, 2, 10), // (6*3+10*1)/4 = 7.0
	}
	r := CalcularRanking(criterios(), fichas, avs)["cosplay"]
	if len(r) != 2 || r[0].FichaID != 10 || r[0].NotaFinal != 7.5 || r[1].NotaFinal != 7 {
		t.Fatalf("ranking inesperado: %+v", r)
	}
	if r[0].Posicao != 1 || r[1].Posicao != 2 {
		t.Errorf("posições: %+v", r)
	}
}

func TestCalcularRankingDesempatePeloCriterioDeMaiorPeso(t *testing.T) {
	// ambos 6.0 final: A = (10*3+... ) montado para empatar; desempata por Fidelidade (peso 3)
	fichas := []EntradaRanking{{FichaID: 10, TipoApresentacao: "cosplay"}, {FichaID: 11, TipoApresentacao: "cosplay"}}
	avs := []domain.Avaliacao{
		aval(10, 1, 1, 8), aval(10, 1, 2, 0), // 6.0
		aval(11, 1, 1, 4), aval(11, 1, 2, 8), // (12+8)/4 = 5.0 -> ajusta abaixo
	}
	avs[2] = aval(11, 1, 1, 6)
	avs[3] = aval(11, 1, 2, 6) // (18+6)/4 = 6.0, fidelidade 6 < 8
	r := CalcularRanking(criterios(), fichas, avs)["cosplay"]
	if r[0].NotaFinal != 6 || r[1].NotaFinal != 6 {
		t.Fatalf("esperava empate em 6.0: %+v", r)
	}
	if r[0].FichaID != 10 || r[0].Posicao != 1 || r[1].Posicao != 2 {
		t.Errorf("desempate por fidelidade deveria colocar 10 em 1º: %+v", r)
	}
}

func TestCalcularRankingEmpateRealDividePosicao(t *testing.T) {
	fichas := []EntradaRanking{{FichaID: 10, TipoApresentacao: "danca"}, {FichaID: 11, TipoApresentacao: "danca"}}
	avs := []domain.Avaliacao{aval(10, 1, 1, 5), aval(10, 1, 2, 5), aval(11, 1, 1, 5), aval(11, 1, 2, 5)}
	r := CalcularRanking(criterios(), fichas, avs)["danca"]
	if r[0].Posicao != 1 || r[1].Posicao != 1 {
		t.Errorf("empate total deve dividir a posição: %+v", r)
	}
}

func TestCalcularRankingIgnoraNaoFinalizadasEEscalasDiferentes(t *testing.T) {
	cs := []domain.CriterioAvaliacao{{ID: 1, Peso: 1, NotaMin: 0, NotaMax: 5, Passo: 1}}
	fichas := []EntradaRanking{{FichaID: 10, TipoApresentacao: "canto"}, {FichaID: 11, TipoApresentacao: "canto"}}
	rascunho := domain.Avaliacao{FichaID: 11, JuradoUsuarioID: 1, CriterioID: 1, Nota: 5, Finalizada: false}
	r := CalcularRanking(cs, fichas, []domain.Avaliacao{aval(10, 1, 1, 5), rascunho})["canto"]
	if len(r) != 1 || r[0].NotaFinal != 10 {
		t.Errorf("nota 5/5 normaliza para 10 e rascunho não conta: %+v", r)
	}
}

func TestNotaValida(t *testing.T) {
	c := &domain.CriterioAvaliacao{NotaMin: 0, NotaMax: 10, Passo: 0.5}
	for nota, ok := range map[float64]bool{0: true, 7.5: true, 10: true, 7.3: false, -1: false, 10.5: false} {
		if notaValida(nota, c) != ok {
			t.Errorf("notaValida(%v) deveria ser %v", nota, ok)
		}
	}
}

func TestNomePublicoProtegeSobrenomeEMenores(t *testing.T) {
	casos := []struct {
		l    LinhaRanking
		want string
	}{
		{LinhaRanking{Nome: "Carlos Silva Souza", NomeArtistico: "Cosplay Man"}, "Cosplay Man"},
		{LinhaRanking{Nome: "Carlos Silva Souza"}, "Carlos S."},
		{LinhaRanking{Nome: "Maria Menor Teste", Menor: true}, "Maria"},
		{LinhaRanking{Nome: "Bruno"}, "Bruno"},
	}
	for _, c := range casos {
		if got := NomePublico(c.l); got != c.want {
			t.Errorf("NomePublico(%+v) = %q, want %q", c.l, got, c.want)
		}
	}
}
