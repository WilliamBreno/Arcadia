package mercadopago

// Item 9.1: envio de Pix para o organizador (Mercado Pago Payouts,
// POST /v1/payouts). Em produção cada envio é assinado com Ed25519 sobre o
// corpo bruto (cabeçalho X-signature, em base64); a chave pública vai para
// a equipe de integrações do MP e a privada fica só na API (variável de
// ambiente). Com token de teste, o MP pede X-test-token e não exige assinatura.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrPayoutRecusado: o MP respondeu com erro definitivo (4xx): nenhum
// dinheiro saiu e repetir igual não adianta.
var ErrPayoutRecusado = errors.New("envio recusado pelo Mercado Pago")

// LerChavePrivadaEd25519 aceita o PEM (PKCS#8) gerado por
// `openssl genpkey -algorithm ed25519`. "\n" escrito literalmente (comum
// em variável de ambiente) vira quebra de linha. Testada.
func LerChavePrivadaEd25519(pemTexto string) (ed25519.PrivateKey, error) {
	pemTexto = strings.ReplaceAll(strings.TrimSpace(pemTexto), `\n`, "\n")
	bloco, _ := pem.Decode([]byte(pemTexto))
	if bloco == nil {
		return nil, errors.New("chave privada do payout: PEM inválido")
	}
	k, err := x509.ParsePKCS8PrivateKey(bloco.Bytes)
	if err != nil {
		return nil, fmt.Errorf("chave privada do payout: %w", err)
	}
	ed, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("chave privada do payout não é Ed25519")
	}
	return ed, nil
}

// AssinarCorpo devolve a assinatura Ed25519 do corpo, em base64. Testada.
func AssinarCorpo(chave ed25519.PrivateKey, corpo []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(chave, corpo))
}

type PixDestino struct {
	Tipo  string `json:"type"` // CPF, CNPJ, EMAIL, PHONE
	Chave string `json:"chave"`
}

type ValorPayout struct {
	Moeda string  `json:"currency"`
	Valor float64 `json:"value"`
}

type TransacaoPayoutRequest struct {
	Descricao  string      `json:"description,omitempty"`
	Tipo       string      `json:"type"`
	Pix        PixDestino  `json:"pix"`
	Valor      ValorPayout `json:"amount"`
	Referencia string      `json:"external_reference,omitempty"`
}

type PayoutRequest struct {
	Referencia string                   `json:"external_reference"`
	Descricao  string                   `json:"description,omitempty"`
	Transacoes []TransacaoPayoutRequest `json:"transactions"`
}

// NovoPayoutPix monta o envio de um Pix. O valor vai em reais (o MP usa
// decimal); a conversão sai de centavos inteiros, sem arredondamento.
func NovoPayoutPix(referencia, descricao, tipoChave, chave string, centavos int64) PayoutRequest {
	return PayoutRequest{
		Referencia: referencia,
		Descricao:  descricao,
		Transacoes: []TransacaoPayoutRequest{{
			Descricao: descricao, Tipo: "pix", Pix: PixDestino{Tipo: tipoChave, Chave: chave},
			Valor: ValorPayout{Moeda: "BRL", Valor: float64(centavos) / 100}, Referencia: referencia,
		}},
	}
}

type Payout struct {
	ID         json.Number `json:"id"`
	Status     string      `json:"status"`
	Referencia string      `json:"external_reference"`
}

type TransacaoPayout struct {
	ID           json.Number `json:"id"`
	Status       string      `json:"status"` // pending, in_process, success, error, canceled
	StatusDetail string      `json:"status_detail"`
}

// EnviarPayout cria o lote. A chave de idempotência garante que repetir a
// mesma chamada (queda de rede no meio) não manda o Pix duas vezes.
// assinatura == nil: ambiente de teste.
func (c *Cliente) EnviarPayout(req PayoutRequest, chaveIdempotencia string, assinatura ed25519.PrivateKey) (*Payout, error) {
	corpo, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/payouts", bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Bearer "+c.accessToken)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Idempotency-Key", chaveIdempotencia)
	if assinatura != nil {
		r.Header.Set("X-enforce-signature", "true")
		r.Header.Set("X-signature", AssinarCorpo(assinatura, corpo))
	} else {
		r.Header.Set("X-test-token", "true")
		r.Header.Set("X-enforce-signature", "false")
	}
	resp, err := c.httpClient.Do(r)
	if err != nil {
		return nil, err // rede: pode ter chegado; repetir com a mesma chave
	}
	defer resp.Body.Close()
	bruto, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: status %d: %s", ErrPayoutRecusado, resp.StatusCode, string(bruto))
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mercadopago payout: status %d: %s", resp.StatusCode, string(bruto))
	}
	var p Payout
	if err := json.Unmarshal(bruto, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// TransacoesPayout consulta a situação de cada Pix do lote.
func (c *Cliente) TransacoesPayout(payoutID string) ([]TransacaoPayout, error) {
	var resp struct {
		Resultados []TransacaoPayout `json:"results"`
	}
	if err := c.requisitar(http.MethodGet, "/v1/payouts/"+payoutID+"/transactions?limit=100", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Resultados, nil
}

// ModoTeste: token de teste do MP (sem assinatura, com X-test-token).
func (c *Cliente) ModoTeste() bool { return strings.HasPrefix(c.accessToken, "TEST-") }
