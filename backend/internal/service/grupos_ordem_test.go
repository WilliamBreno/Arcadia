package service

import "testing"

func TestValidarOrdem(t *testing.T) {
	for _, c := range []struct {
		nome        string
		ids, atuais []int64
		ok          bool
	}{
		{"troca", []int64{3, 1, 2}, []int64{1, 2, 3}, true},
		{"vazia", []int64{}, []int64{}, true},
		{"duplicado", []int64{1, 1, 3}, []int64{1, 2, 3}, false},
		{"outro perfil", []int64{1, 2, 9}, []int64{1, 2, 3}, false},
		{"faltando", []int64{1, 2}, []int64{1, 2, 3}, false},
		{"inserido depois", []int64{1, 2, 3}, []int64{1, 2, 3, 4}, false},
		{"removido depois", []int64{1, 2, 3}, []int64{1, 2}, false},
		{"zero", []int64{0}, []int64{1}, false},
	} {
		t.Run(c.nome, func(t *testing.T) {
			if validarOrdem(c.ids, c.atuais) != c.ok {
				t.Fatal("validação incorreta")
			}
		})
	}
}
