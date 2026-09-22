package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GerarTokenOpaco cria um token aleatório de 32 bytes em base64url, usado
// para refresh tokens, verificação de e-mail e redefinição de senha.
// Segue a mesma convenção dos tokens de convite (seção 7.11 do plano):
// apenas o hash (sha256) é salvo no banco, nunca o token em texto puro.
func GerarTokenOpaco() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken calcula o hash de um token opaco para armazenamento/consulta.
func HashToken(token string) string {
	soma := sha256.Sum256([]byte(token))
	return hex.EncodeToString(soma[:])
}
