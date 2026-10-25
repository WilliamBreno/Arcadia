package mercadopago

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
)

func TestTiposExcluidos(t *testing.T) {
	pix := tiposExcluidos(pagamento.MetodoPix)
	for _, tipo := range []string{"ticket", "atm", "credit_card", "debit_card", "prepaid_card"} {
		if !slices.Contains(pix, tipo) {
			t.Errorf("Pix deveria excluir %s: %v", tipo, pix)
		}
	}
	if slices.Contains(pix, "bank_transfer") {
		t.Errorf("Pix não pode excluir bank_transfer (é o próprio Pix): %v", pix)
	}

	cartao := tiposExcluidos(pagamento.MetodoCartao)
	for _, tipo := range []string{"ticket", "atm", "bank_transfer"} {
		if !slices.Contains(cartao, tipo) {
			t.Errorf("cartão deveria excluir %s: %v", tipo, cartao)
		}
	}
	if slices.Contains(cartao, "credit_card") {
		t.Errorf("cartão não pode excluir credit_card: %v", cartao)
	}

	// Chamadas seguidas não podem compartilhar o slice base.
	_ = tiposExcluidos(pagamento.MetodoPix)
	if len(tiposSempreExcluidos) != 2 {
		t.Errorf("tiposSempreExcluidos foi alterado: %v", tiposSempreExcluidos)
	}
}

func TestCriarCobrancaPixEnviaFiltroEValidade(t *testing.T) {
	var corpo map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkout/preferences" || r.Header.Get("Authorization") != "Bearer TEST-x" {
			t.Errorf("requisição inesperada: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		_, _ = w.Write([]byte(`{"id":"pref-1","init_point":"https://mp/checkout/pref-1"}`))
	}))
	defer srv.Close()

	c := NovoCliente("TEST-x")
	c.baseURL = srv.URL
	expira := time.Date(2026, 10, 25, 12, 15, 0, 0, time.FixedZone("BRT", -3*3600))
	cob, err := NovoProcessador(c).CriarCobranca(pagamento.NovaCobranca{
		Referencia: "42", Metodo: pagamento.MetodoPix, ExpiraEm: expira,
		Itens: []pagamento.Item{{Titulo: "Inteira", ValorCentavos: 3099}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cob.ID != "pref-1" || cob.URL != "https://mp/checkout/pref-1" {
		t.Errorf("cobrança inesperada: %+v", cob)
	}
	if corpo["external_reference"] != "42" {
		t.Errorf("external_reference = %v", corpo["external_reference"])
	}
	if corpo["date_of_expiration"] != "2026-10-25T12:15:00.000-03:00" || corpo["expires"] != true {
		t.Errorf("validade não enviada: %v / %v", corpo["date_of_expiration"], corpo["expires"])
	}
	item := corpo["items"].([]any)[0].(map[string]any)
	if item["unit_price"] != 30.99 {
		t.Errorf("unit_price = %v", item["unit_price"])
	}
	pm := corpo["payment_methods"].(map[string]any)
	if pm["installments"] != float64(1) {
		t.Errorf("installments = %v", pm["installments"])
	}
	var ids []string
	for _, e := range pm["excluded_payment_types"].([]any) {
		ids = append(ids, e.(map[string]any)["id"].(string))
	}
	if !slices.Contains(ids, "credit_card") || !slices.Contains(ids, "ticket") || slices.Contains(ids, "bank_transfer") {
		t.Errorf("excluded_payment_types do Pix = %v", ids)
	}
}
