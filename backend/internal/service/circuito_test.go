package service

import "testing"

func TestPontos(t *testing.T) {
	casos := map[int]int{0: 0, 1: 25, 2: 18, 3: 15, 10: 1, 11: 0}
	for pos, esperado := range casos {
		if got := Pontos(pos); got != esperado {
			t.Errorf("Pontos(%d) = %d, esperado %d", pos, got, esperado)
		}
	}
}

func TestSomarCircuito(t *testing.T) {
	r := func(tipo string, usuario int64, pos int, nome string, menor bool) ResultadoEvento {
		return ResultadoEvento{Tipo: tipo, UsuarioID: usuario, Posicao: pos, NomePublico: nome, Menor: menor}
	}
	linhas := []ResultadoEvento{
		// etapa 1
		r("cosplay", 1, 1, "Ana antiga", false), r("cosplay", 2, 2, "Bia", false), r("cosplay", 3, 3, "Caio", true), r("cosplay", 4, 11, "Davi", false),
		// etapa 2
		r("cosplay", 2, 1, "Bia", false), r("cosplay", 1, 2, "Ana", false), r("cosplay", 3, 4, "Caio", true),
		// outra categoria não mistura
		r("danca", 1, 1, "Ana", false),
	}
	got := SomarCircuito(linhas)
	cos := got["cosplay"]
	if len(cos) != 3 {
		t.Fatalf("esperado 3 pontuando (Davi fora do top 10), veio %d: %+v", len(cos), cos)
	}
	// Ana 25+18=43 (1 vitória), Bia 18+25=43 (1 vitória) -> empate total? Ana 1 segundo, Bia 1 segundo -> nome decide, mesma posição.
	if cos[0].Pontos != 43 || cos[1].Pontos != 43 || cos[0].Posicao != 1 || cos[1].Posicao != 1 {
		t.Errorf("empate em pontos, vitórias e 2º lugares divide a posição: %+v", cos[:2])
	}
	if cos[0].Nome != "Ana" {
		t.Errorf("nome exibido é o da etapa mais recente: %q", cos[0].Nome)
	}
	if cos[2].Nome != "Caio" || cos[2].Pontos != 27 || cos[2].Posicao != 3 || !cos[2].Menor || cos[2].Eventos != 2 {
		t.Errorf("Caio: %+v", cos[2])
	}
	if len(got["danca"]) != 1 || got["danca"][0].Pontos != 25 {
		t.Errorf("dança separada: %+v", got["danca"])
	}

	// Desempate por vitórias.
	d := SomarCircuito([]ResultadoEvento{
		r("canto", 1, 2, "A", false), r("canto", 1, 2, "A", false), r("canto", 1, 10, "A", false), // 18+18+1 = 37
		r("canto", 2, 1, "B", false), r("canto", 2, 6, "B", false), r("canto", 2, 7, "B", false), // 25+8+6 = 39
		r("canto", 3, 1, "C", false), r("canto", 3, 5, "C", false), r("canto", 3, 10, "C", false), r("canto", 3, 10, "C", false), // 25+10+1+1 = 37
	})["canto"]
	if d[0].Nome != "B" || d[1].Nome != "C" || d[2].Nome != "A" || d[2].Posicao != 3 {
		t.Errorf("C (1 vitória) à frente de A (0) com 37 pontos: %+v", d)
	}
}
