package service

import (
	"strings"
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestMotivoParaAprovar(t *testing.T) {
	ok := func() *domain.Organizador {
		return &domain.Organizador{Status: domain.StatusOrganizadorAtivo, Documento: "52998224725", TipoChavePix: PixCPF, ChavePix: "52998224725"}
	}
	casos := []struct {
		nome          string
		valor, limite int64
		org           func() *domain.Organizador
		ref, con      int64
		contem        string
	}{
		{"tudo certo", 100000, 2000000, ok, 0, 0, ""},
		{"no limite exato", 2000000, 2000000, ok, 0, 0, ""},
		{"acima do limite", 2000001, 2000000, ok, 0, 0, "limite"},
		{"sem valor", 0, 2000000, ok, 0, 0, "não há valor"},
		{"reembolso falho", 100, 2000000, ok, 1, 0, "reembolso"},
		{"contestação", 100, 2000000, ok, 0, 2, "contestação"},
		{"suspenso", 100, 2000000, func() *domain.Organizador { o := ok(); o.Status = domain.StatusOrganizadorSuspenso; return o }, 0, 0, "ativo"},
		{"chave de outro CPF", 100, 2000000, func() *domain.Organizador { o := ok(); o.ChavePix = "11144477735"; return o }, 0, 0, "próprio"},
		{"chave e-mail", 100, 2000000, func() *domain.Organizador { o := ok(); o.TipoChavePix, o.ChavePix = PixEmail, "a@b.com"; return o }, 0, 0, "próprio"},
		{"CNPJ igual", 100, 2000000, func() *domain.Organizador {
			o := ok()
			o.Documento, o.TipoChavePix, o.ChavePix = "11222333000181", PixCNPJ, "11222333000181"
			return o
		}, 0, 0, ""},
		{"sem chave", 100, 2000000, func() *domain.Organizador { o := ok(); o.ChavePix = ""; return o }, 0, 0, "sem chave"},
	}
	for _, c := range casos {
		m := MotivoParaAprovar(c.valor, c.limite, c.org(), c.ref, c.con)
		if (c.contem == "") != (m == "") || (c.contem != "" && !strings.Contains(m, c.contem)) {
			t.Errorf("%s: motivo %q, esperado conter %q", c.nome, m, c.contem)
		}
	}
}

func TestTipoPixMP(t *testing.T) {
	for in, out := range map[string]string{PixCPF: "CPF", PixCNPJ: "CNPJ", PixEmail: "EMAIL", PixTelefone: "PHONE", PixAleatoria: ""} {
		if tipoPixMP(in) != out {
			t.Errorf("%s -> %q", in, tipoPixMP(in))
		}
	}
}
