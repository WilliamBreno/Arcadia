package service

import "testing"

func TestHashTokenQR(t *testing.T) {
	// SHA-256("abc") — o front usa crypto.subtle.digest com o mesmo texto
	esperado := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := HashTokenQR("abc"); got != esperado {
		t.Errorf("HashTokenQR(abc) = %s", got)
	}
	if HashTokenQR("a") == HashTokenQR("b") {
		t.Error("tokens diferentes devem ter hashes diferentes")
	}
}
