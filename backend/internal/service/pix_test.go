package service

import (
	"errors"
	"testing"
)

func TestNormalizarChavePix(t *testing.T) {
	casos := []struct {
		tipo, chave     string
		tipoOK, chaveOK string
		valida          bool
	}{
		{"", "", "", "", true}, // sem Pix ainda
		{"cpf", "529.982.247-25", "cpf", "52998224725", true},
		{"CPF", "52998224725", "cpf", "52998224725", true},
		{"cpf", "529.982.247-26", "", "", false}, // dígito errado
		{"cpf", "111.111.111-11", "", "", false},
		{"cnpj", "11.222.333/0001-81", "cnpj", "11222333000181", true},
		{"cnpj", "11.222.333/0001-82", "", "", false},
		{"email", " Org@Evve.App ", "email", "org@evve.app", true},
		{"email", "org@evve", "", "", false},
		{"telefone", "(82) 99999-1234", "telefone", "+5582999991234", true},
		{"telefone", "+55 82 3333-1234", "telefone", "+558233331234", true},
		{"telefone", "99999-1234", "", "", false}, // sem DDD
		{"aleatoria", "123E4567-E89B-12D3-A456-426614174000", "aleatoria", "123e4567-e89b-12d3-a456-426614174000", true},
		{"aleatoria", "123e4567", "", "", false},
		{"pix", "qualquer", "", "", false},
		{"cpf", "", "", "", false},
		{"", "52998224725", "", "", false},
	}
	for _, c := range casos {
		tipo, chave, err := NormalizarChavePix(c.tipo, c.chave)
		if c.valida != (err == nil) {
			t.Errorf("NormalizarChavePix(%q, %q): erro = %v", c.tipo, c.chave, err)
			continue
		}
		if err != nil && !errors.Is(err, ErrChavePixInvalida) {
			t.Errorf("erro deveria ser ErrChavePixInvalida: %v", err)
		}
		if err == nil && (tipo != c.tipoOK || chave != c.chaveOK) {
			t.Errorf("NormalizarChavePix(%q, %q) = (%q, %q), esperado (%q, %q)", c.tipo, c.chave, tipo, chave, c.tipoOK, c.chaveOK)
		}
	}
}
