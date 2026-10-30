package service

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNormalizarPergunta(t *testing.T) {
	p, err := NormalizarPergunta(Pergunta{Texto: "  Tamanho   da camiseta ", Tipo: PerguntaOpcao, Opcoes: []string{" P", "M", "m", "", "G"}, Finalidade: "Para separar o kit"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Texto != "Tamanho da camiseta" || strings.Join(p.Opcoes, ",") != "P,M,G" || p.TipoIngressoIDs == nil {
		t.Errorf("normalização: %+v", p)
	}
	ruins := []Pergunta{
		{Texto: "Oi", Tipo: PerguntaTexto, Finalidade: "Para saber"},
		{Texto: "Como soube?", Tipo: PerguntaTexto, Finalidade: ""},
		{Texto: "Como soube?", Tipo: "data", Finalidade: "Para divulgar"},
		{Texto: "Tamanho", Tipo: PerguntaOpcao, Opcoes: []string{"P"}, Finalidade: "Para o kit"},
	}
	for _, r := range ruins {
		if _, err := NormalizarPergunta(r); !errors.Is(err, ErrPerguntaInvalida) {
			t.Errorf("deveria recusar %+v", r)
		}
	}
}

func TestValidarRespostas(t *testing.T) {
	ps := []Pergunta{
		{ID: 1, Texto: "Camiseta", Tipo: PerguntaOpcao, Opcoes: []string{"P", "M"}, Obrigatoria: true},
		{ID: 2, Texto: "Como soube", Tipo: PerguntaTexto},
		{ID: 3, Texto: "Só VIP", Tipo: PerguntaTexto, Obrigatoria: true, TipoIngressoIDs: []int64{99}},
	}
	got, err := ValidarRespostas(ps, 10, map[int64]string{1: "M", 2: "  Instagram  ", 7: "ignorada"})
	if err != nil || got[1] != "M" || got[2] != "Instagram" || len(got) != 2 {
		t.Errorf("válidas: %v %v", got, err)
	}
	if _, err := ValidarRespostas(ps, 10, map[int64]string{2: "x"}); !errors.Is(err, ErrRespostaInvalida) {
		t.Error("obrigatória em branco deve falhar")
	}
	if _, err := ValidarRespostas(ps, 10, map[int64]string{1: "GG"}); !errors.Is(err, ErrRespostaInvalida) {
		t.Error("opção fora da lista deve falhar")
	}
	if _, err := ValidarRespostas(ps, 99, map[int64]string{1: "P"}); !errors.Is(err, ErrRespostaInvalida) {
		t.Error("pergunta do tipo 99 é obrigatória para ele")
	}
	if _, err := ValidarRespostas(ps, 10, map[int64]string{1: "P", 2: strings.Repeat("a", 301)}); !errors.Is(err, ErrRespostaInvalida) {
		t.Error("resposta longa demais deve falhar")
	}
}

func TestRespostasCSVProtegeFormula(t *testing.T) {
	var b bytes.Buffer
	ps := []Pergunta{{ID: 1, Texto: "Como soube"}}
	err := EscreverRespostasCSV(&b, ps, []LinhaRespostas{{Codigo: "ABC", Tipo: "Pista", Titular: "Ana", Email: "a@b.c", Status: "pago", Respostas: map[int64]string{1: "=HYPERLINK(\"x\")"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "'=HYPERLINK") || !strings.HasPrefix(b.String(), "codigo;ingresso;titular;email;situacao;Como soube") {
		t.Errorf("csv: %q", b.String())
	}
}
