package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// QR rotativo (anti-print, item da Fase 4): o QR do ingresso passa a carregar
// um token que muda a cada JanelaQRSegundos. Formato: "R<janela>.<hmac>", com
// janela = unix/30. O servidor aceita a janela atual e as vizinhas (±1) para
// tolerar diferença de relógio; screenshot/impressão deixa de valer em ~1 min.
const JanelaQRSegundos = 30

type statusQR int

const (
	qrValido statusQR = iota
	qrExpirado
	qrInvalido
)

func janelaQR(t time.Time) int64 { return t.Unix() / JanelaQRSegundos }

func tokenRotativo(segredo, codigo string, janela int64) string {
	h := hmac.New(sha256.New, []byte(segredo))
	h.Write([]byte("rot:" + codigo + ":" + strconv.FormatInt(janela, 10)))
	return "R" + strconv.FormatInt(janela, 10) + "." + hex.EncodeToString(h.Sum(nil))
}

func ehTokenRotativo(token string) bool { return strings.HasPrefix(token, "R") }

func validarTokenRotativo(segredo, codigo, token string, agora time.Time) statusQR {
	partes := strings.SplitN(strings.TrimPrefix(token, "R"), ".", 2)
	if len(partes) != 2 {
		return qrInvalido
	}
	janela, err := strconv.ParseInt(partes[0], 10, 64)
	if err != nil {
		return qrInvalido
	}
	esperado := tokenRotativo(segredo, codigo, janela)
	if subtle.ConstantTimeCompare([]byte(esperado), []byte(token)) != 1 {
		return qrInvalido
	}
	if d := janelaQR(agora) - janela; d > 1 || d < -1 {
		return qrExpirado
	}
	return qrValido
}

// PayloadQR é o texto que vai dentro do QR do ingresso.
func PayloadQR(segredo, codigo, qrTokenEstatico string, rotativo bool, agora time.Time) string {
	if rotativo {
		return codigo + ":" + tokenRotativo(segredo, codigo, janelaQR(agora))
	}
	return codigo + ":" + qrTokenEstatico
}
