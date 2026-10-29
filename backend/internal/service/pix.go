package service

import (
	"errors"
	"regexp"
	"strings"
)

// Chave Pix do organizador (é para onde vai o repasse). Antes o tipo era
// texto livre e a chave não era conferida; um erro de digitação mandaria o
// dinheiro para o lugar errado ou faria o Pix falhar.

var ErrChavePixInvalida = errors.New("chave Pix inválida")

// Tipos de chave aceitos pelo Banco Central.
const (
	PixCPF       = "cpf"
	PixCNPJ      = "cnpj"
	PixEmail     = "email"
	PixTelefone  = "telefone"
	PixAleatoria = "aleatoria"
)

var (
	reEmailPix     = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reAleatoriaPix = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// NormalizarChavePix confere a chave conforme o tipo e devolve o formato
// guardado (CPF/CNPJ só dígitos, telefone +55..., e-mail e aleatória em
// minúsculas). Tipo e chave vazios juntos = "sem Pix ainda" (permitido).
func NormalizarChavePix(tipo, chave string) (string, string, error) {
	tipo = strings.ToLower(strings.TrimSpace(tipo))
	chave = strings.TrimSpace(chave)
	if tipo == "" && chave == "" {
		return "", "", nil
	}
	if tipo == "" || chave == "" {
		return "", "", errors.Join(ErrChavePixInvalida, errors.New("informe o tipo e a chave Pix"))
	}
	switch tipo {
	case PixCPF:
		d := somenteDigitos(chave)
		if !cpfValido(d) {
			return "", "", errors.Join(ErrChavePixInvalida, errors.New("CPF inválido (confira os números)"))
		}
		return tipo, d, nil
	case PixCNPJ:
		d := somenteDigitos(chave)
		if !cnpjValido(d) {
			return "", "", errors.Join(ErrChavePixInvalida, errors.New("CNPJ inválido (confira os números)"))
		}
		return tipo, d, nil
	case PixEmail:
		e := strings.ToLower(chave)
		if len(e) > 77 || !reEmailPix.MatchString(e) {
			return "", "", errors.Join(ErrChavePixInvalida, errors.New("e-mail inválido"))
		}
		return tipo, e, nil
	case PixTelefone:
		d := somenteDigitos(chave)
		if strings.HasPrefix(d, "55") && (len(d) == 12 || len(d) == 13) {
			d = d[2:]
		}
		// DDD (2) + número (8 fixo ou 9 celular).
		if len(d) != 10 && len(d) != 11 || d[0] == '0' {
			return "", "", errors.Join(ErrChavePixInvalida, errors.New("telefone inválido: use DDD + número, ex.: (82) 99999-9999"))
		}
		return tipo, "+55" + d, nil
	case PixAleatoria:
		a := strings.ToLower(chave)
		if !reAleatoriaPix.MatchString(a) {
			return "", "", errors.Join(ErrChavePixInvalida, errors.New("chave aleatória inválida: copie exatamente como aparece no app do seu banco"))
		}
		return tipo, a, nil
	}
	return "", "", errors.Join(ErrChavePixInvalida, errors.New("tipo de chave deve ser CPF, CNPJ, e-mail, telefone ou aleatória"))
}

func todosIguais(d string) bool {
	return strings.Count(d, d[:1]) == len(d)
}

func cpfValido(d string) bool {
	if len(d) != 11 || todosIguais(d) {
		return false
	}
	for _, n := range []int{9, 10} {
		soma := 0
		for i := 0; i < n; i++ {
			soma += int(d[i]-'0') * (n + 1 - i)
		}
		dv := soma * 10 % 11
		if dv == 10 {
			dv = 0
		}
		if dv != int(d[n]-'0') {
			return false
		}
	}
	return true
}

func cnpjValido(d string) bool {
	if len(d) != 14 || todosIguais(d) {
		return false
	}
	pesos := [][]int{{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}, {6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}}
	for k, p := range pesos {
		soma := 0
		for i, w := range p {
			soma += int(d[i]-'0') * w
		}
		dv := soma % 11
		if dv < 2 {
			dv = 0
		} else {
			dv = 11 - dv
		}
		if dv != int(d[12+k]-'0') {
			return false
		}
	}
	return true
}
