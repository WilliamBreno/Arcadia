// Package mercadopago fala com a API REST do Mercado Pago diretamente
// (sem o SDK oficial) — só as duas chamadas que a plataforma precisa:
// criar uma preference (Checkout Pro) e consultar um pagamento. Mantém a
// dependência mínima e o comportamento totalmente sob controle.
package mercadopago

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseURL = "https://api.mercadopago.com"

type Cliente struct {
	accessToken string
	httpClient  *http.Client
}

func NovoCliente(accessToken string) *Cliente {
	return &Cliente{accessToken: accessToken, httpClient: &http.Client{}}
}

func (c *Cliente) Habilitado() bool {
	return c.accessToken != ""
}

func (c *Cliente) requisitar(metodo, caminho string, corpo any, resposta any) error {
	var leitor io.Reader
	if corpo != nil {
		bruto, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(bruto)
	}

	req, err := http.NewRequest(metodo, baseURL+caminho, leitor)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	corpoResposta, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mercadopago: status %d: %s", resp.StatusCode, string(corpoResposta))
	}

	if resposta != nil {
		return json.Unmarshal(corpoResposta, resposta)
	}
	return nil
}

type ItemPreference struct {
	Title      string  `json:"title"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	CurrencyID string  `json:"currency_id"`
}

type backURLs struct {
	Success string `json:"success"`
	Failure string `json:"failure"`
	Pending string `json:"pending"`
}

type criarPreferenceRequest struct {
	Items             []ItemPreference `json:"items"`
	ExternalReference string           `json:"external_reference"`
	BackURLs          backURLs         `json:"back_urls"`
	AutoReturn        string           `json:"auto_return,omitempty"`
	NotificationURL   string           `json:"notification_url"`
}

type Preference struct {
	ID               string `json:"id"`
	InitPoint        string `json:"init_point"`
	SandboxInitPoint string `json:"sandbox_init_point"`
}

// CriarPreference cria uma sessão de checkout (Checkout Pro). Não move
// dinheiro — só gera o link para onde o comprador é redirecionado.
//
// auto_return fica de fora de propósito: o MP só aceita esse campo
// quando back_urls.success é uma URL pública (rejeita localhost), o que
// quebraria em dev. Sem ele, o comprador só precisa clicar em "voltar ao
// site" após pagar em vez de ser redirecionado automaticamente — o
// webhook (que de fato confirma o pagamento) não depende disso.
func (c *Cliente) CriarPreference(externalReference string, itens []ItemPreference, urlSucesso, urlFalha, urlPendente, notificationURL string) (*Preference, error) {
	req := criarPreferenceRequest{
		Items:             itens,
		ExternalReference: externalReference,
		BackURLs:          backURLs{Success: urlSucesso, Failure: urlFalha, Pending: urlPendente},
		NotificationURL:   notificationURL,
	}

	var resp Preference
	if err := c.requisitar(http.MethodPost, "/checkout/preferences", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type FeeDetail struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}

type Pagamento struct {
	ID                int64       `json:"id"`
	Status            string      `json:"status"`
	StatusDetail      string      `json:"status_detail"`
	TransactionAmount float64     `json:"transaction_amount"`
	PaymentMethodID   string      `json:"payment_method_id"`
	PaymentTypeID     string      `json:"payment_type_id"`
	ExternalReference string      `json:"external_reference"`
	FeeDetails        []FeeDetail `json:"fee_details"`
}

// TaxaProcessadorCentavos soma os fee_details retornados pelo MP
// (seção 7.7) e converte para centavos.
func (p *Pagamento) TaxaProcessadorCentavos() int64 {
	var total float64
	for _, f := range p.FeeDetails {
		total += f.Amount
	}
	return int64(total*100 + 0.5)
}

// BuscarPagamento SEMPRE deve ser chamado a partir do ID recebido no
// webhook — nunca confiar em valor/status do corpo da notificação em si
// (seção 7.3 do plano: é a defesa contra webhook falsificado).
func (c *Cliente) BuscarPagamento(id string) (*Pagamento, error) {
	var resp Pagamento
	if err := c.requisitar(http.MethodGet, "/v1/payments/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type reembolsoRequest struct {
	Amount *float64 `json:"amount,omitempty"`
}

type Reembolso struct {
	ID     int64   `json:"id"`
	Status string  `json:"status"`
	Amount float64 `json:"amount"`
}

// SolicitarReembolso estorna um pagamento — total se valorCentavos for
// nil, parcial caso contrário (um pagamento pode cobrir vários itens do
// pedido, seção 7.4). O MP garante que a soma dos parciais nunca excede
// o valor pago; não precisamos reimplementar essa checagem aqui.
func (c *Cliente) SolicitarReembolso(paymentID string, valorCentavos *int64) (*Reembolso, error) {
	req := reembolsoRequest{}
	if valorCentavos != nil {
		valor := float64(*valorCentavos) / 100
		req.Amount = &valor
	}

	var resp Reembolso
	if err := c.requisitar(http.MethodPost, "/v1/payments/"+paymentID+"/refunds", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
