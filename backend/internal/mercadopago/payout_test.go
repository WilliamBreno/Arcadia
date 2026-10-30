package mercadopago

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func chavePEM(t *testing.T) (string, ed25519.PublicKey) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), pub
}

func TestLerChaveEAssinar(t *testing.T) {
	p, pub := chavePEM(t)
	// Variável de ambiente com "\n" literal.
	k, err := LerChavePrivadaEd25519(strings.ReplaceAll(p, "\n", `\n`))
	if err != nil {
		t.Fatal(err)
	}
	corpo := []byte(`{"a":1}`)
	sig, _ := base64.StdEncoding.DecodeString(AssinarCorpo(k, corpo))
	if !ed25519.Verify(pub, corpo, sig) {
		t.Fatal("assinatura não confere")
	}
	if _, err := LerChavePrivadaEd25519("lixo"); err == nil {
		t.Error("aceitou PEM inválido")
	}
}

func TestEnviarPayoutAssinaECentavos(t *testing.T) {
	p, pub := chavePEM(t)
	k, _ := LerChavePrivadaEd25519(p)
	var cabecalhos http.Header
	var corpo []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cabecalhos, corpo = r.Header.Clone(), must(io.ReadAll(r.Body))
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id": 987, "status": "pending", "external_reference": "EVVE-REP-1-1"}`))
	}))
	defer srv.Close()
	c := NovoCliente("APP_USR-x").ComBaseURL(srv.URL)
	r, err := c.EnviarPayout(NovoPayoutPix("EVVE-REP-1-1", "Repasse evento 1", "CPF", "52998224725", 123456), "EVVE-REP-1-1", k)
	if err != nil || r.ID.String() != "987" {
		t.Fatalf("resposta: %v %+v", err, r)
	}
	sig, _ := base64.StdEncoding.DecodeString(cabecalhos.Get("X-signature"))
	if !ed25519.Verify(pub, corpo, sig) || cabecalhos.Get("X-enforce-signature") != "true" || cabecalhos.Get("X-Idempotency-Key") != "EVVE-REP-1-1" {
		t.Errorf("cabeçalhos: %v", cabecalhos)
	}
	var enviado PayoutRequest
	_ = json.Unmarshal(corpo, &enviado)
	if enviado.Transacoes[0].Valor.Valor != 1234.56 || enviado.Transacoes[0].Pix.Tipo != "CPF" {
		t.Errorf("corpo: %s", corpo)
	}
}

func TestEnviarPayoutRecusado(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid pix key"}`))
	}))
	defer srv.Close()
	_, err := NovoCliente("TEST-x").ComBaseURL(srv.URL).EnviarPayout(NovoPayoutPix("R", "d", "EMAIL", "a@b.c", 100), "R", nil)
	if err == nil || !strings.Contains(err.Error(), "recusado") {
		t.Fatalf("esperava recusa definitiva: %v", err)
	}
}

func must(b []byte, _ error) []byte { return b }
