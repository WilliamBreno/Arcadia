package mail

import (
	"testing"
	"time"
)

func TestIntervaloNovaTentativa(t *testing.T) {
	casos := map[int]time.Duration{
		0: time.Minute, 1: time.Minute, 2: 2 * time.Minute, 3: 4 * time.Minute,
		7: 64 * time.Minute, 8: 2 * time.Hour, 50: 2 * time.Hour,
	}
	for n, esperado := range casos {
		if got := intervaloNovaTentativa(n); got != esperado {
			t.Errorf("intervaloNovaTentativa(%d) = %v, esperado %v", n, got, esperado)
		}
	}
}
