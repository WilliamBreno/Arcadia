package mercadopago

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
)

func TestRequisicaoPagamentoPix(t *testing.T) {
	exp := time.Date(2026, 10, 29, 20, 31, 0, 0, time.FixedZone("BRT", -3*3600))
	r := requisicaoPagamento(pagamento.NovoPagamento{
		Referencia: "42", ValorCentavos: 2599, Metodo: pagamento.MetodoPix, EmailPagador: "a@b.c", ExpiraEm: exp,
		CartaoToken: "nao-deve-ir", DocumentoNumero: "123",
	})
	if r.PaymentMethodID != "pix" || r.TransactionAmount != 25.99 || r.ExternalReference != "42" {
		t.Fatalf("pix montado errado: %+v", r)
	}
	if r.DateOfExpiration != "2026-10-29T20:31:00.000-03:00" {
		t.Errorf("validade: %q", r.DateOfExpiration)
	}
	if r.Token != "" || r.Payer.Identification != nil || r.ThreeDSecureMode != "" || r.Installments != 0 {
		t.Errorf("Pix não leva dados de cartão: %+v", r)
	}
}

func TestRequisicaoPagamentoCartao(t *testing.T) {
	r := requisicaoPagamento(pagamento.NovoPagamento{
		Referencia: "7", ValorCentavos: 10000, Metodo: pagamento.MetodoCartao, EmailPagador: "a@b.c",
		CartaoToken: "tok", BandeiraID: "visa", EmissorID: "25", DocumentoNumero: "52998224725",
	})
	if r.PaymentMethodID != "visa" || r.Token != "tok" || r.IssuerID != "25" {
		t.Fatalf("cartão montado errado: %+v", r)
	}
	if r.Installments != 1 {
		t.Errorf("sem parcelas informadas: 1x, veio %d", r.Installments)
	}
	if r6 := requisicaoPagamento(pagamento.NovoPagamento{Metodo: pagamento.MetodoCartao, BandeiraID: "visa", CartaoToken: "t", Parcelas: 6}); r6.Installments != 6 {
		t.Errorf("6x pedido, veio %d", r6.Installments)
	}
	if r.ThreeDSecureMode != "optional" {
		t.Errorf("3DS = %q, esperado optional", r.ThreeDSecureMode)
	}
	if r.Payer.Identification == nil || r.Payer.Identification.Type != "CPF" || r.Payer.Identification.Number != "52998224725" {
		t.Errorf("documento: %+v", r.Payer.Identification)
	}
	if r.DateOfExpiration != "" {
		t.Error("cartão não tem validade")
	}
}

func TestCriarPagamentoMandaIdempotenciaELeQR(t *testing.T) {
	var chave string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chave = r.Header.Get("X-Idempotency-Key")
		if r.Method != http.MethodPost || r.URL.Path != "/v1/payments" {
			t.Errorf("chamada inesperada %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"id":99,"status":"pending","status_detail":"pending_waiting_transfer","date_of_expiration":"2026-10-29T20:31:00.000-03:00",
			"point_of_interaction":{"transaction_data":{"qr_code":"000201PIX","qr_code_base64":"aGk="}}}`))
	}))
	defer srv.Close()
	p := NovoProcessador(NovoCliente("TEST-x").ComBaseURL(srv.URL))
	c, err := p.CriarPagamento(pagamento.NovoPagamento{Metodo: pagamento.MetodoPix, ValorCentavos: 100, ChaveIdempotencia: "pix-1"})
	if err != nil {
		t.Fatal(err)
	}
	if chave != "pix-1" {
		t.Errorf("X-Idempotency-Key = %q", chave)
	}
	if c.ID != "99" || c.PixCopiaCola != "000201PIX" || c.PixExpiraEm.IsZero() {
		t.Errorf("resposta: %+v", c)
	}
}

func TestDesafio3DSEDevolucao(t *testing.T) {
	var corpoRefund, chaveRefund string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/refunds") {
			chaveRefund = r.Header.Get("X-Idempotency-Key")
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			bs, _ := json.Marshal(b)
			corpoRefund = string(bs)
			w.Write([]byte(`{"id":5,"status":"approved"}`))
			return
		}
		w.Write([]byte(`{"id":8,"status":"pending","status_detail":"pending_challenge","three_ds_info":{"external_resource_url":"https://banco/3ds","creq":"abc"}}`))
	}))
	defer srv.Close()
	p := NovoProcessador(NovoCliente("TEST-x").ComBaseURL(srv.URL))
	c, err := p.CriarPagamento(pagamento.NovoPagamento{Metodo: pagamento.MetodoCartao, ValorCentavos: 100, BandeiraID: "master", CartaoToken: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if c.StatusDetalhe != pagamento.DetalheDesafio3DS || c.DesafioURL != "https://banco/3ds" || c.DesafioCreq != "abc" {
		t.Errorf("desafio: %+v", c)
	}
	r, err := p.Devolver("8", "devolucao-8")
	if err != nil || r.ID != "5" {
		t.Fatalf("devolver: %v %+v", err, r)
	}
	if chaveRefund != "devolucao-8" || corpoRefund != "{}" {
		t.Errorf("estorno total sem valor e com chave: corpo=%s chave=%s", corpoRefund, chaveRefund)
	}
}
