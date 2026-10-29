// Package mercadopago fala com a API REST do Mercado Pago diretamente
// (sem o SDK oficial) — só o que a plataforma precisa: criar uma preference
// (Checkout Pro), criar um pagamento direto (Pix e cartão na página, item
// 7.1), consultar, cancelar e estornar.
// Os serviços usam o adaptador Processador (contrato pagamento.Processador). Mantém a
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
	baseURL     string // trocado só em teste
}

func NovoCliente(accessToken string) *Cliente {
	return &Cliente{accessToken: accessToken, httpClient: &http.Client{}, baseURL: baseURL}
}

// ComBaseURL aponta o cliente para outra API (servidor falso em testes
// locais; a config de produção recusa essa opção).
func (c *Cliente) ComBaseURL(url string) *Cliente {
	if url != "" {
		c.baseURL = url
	}
	return c
}

func (c *Cliente) Habilitado() bool {
	return c.accessToken != ""
}

func (c *Cliente) requisitar(metodo, caminho string, corpo any, resposta any) error {
	return c.requisitarCom(metodo, caminho, corpo, resposta, "")
}

// requisitarCom manda X-Idempotency-Key quando há chave: o MP devolve a
// mesma resposta se a chamada for repetida, sem cobrar ou estornar de novo.
func (c *Cliente) requisitarCom(metodo, caminho string, corpo any, resposta any, chave string) error {
	var leitor io.Reader
	if corpo != nil {
		bruto, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(bruto)
	}

	req, err := http.NewRequest(metodo, c.baseURL+caminho, leitor)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	if chave != "" {
		req.Header.Set("X-Idempotency-Key", chave)
	}

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

type tipoExcluido struct {
	ID string `json:"id"`
}

type metodosPagamento struct {
	ExcludedPaymentTypes []tipoExcluido `json:"excluded_payment_types,omitempty"`
	Installments         int            `json:"installments,omitempty"` // máximo de parcelas
}

type criarPreferenceRequest struct {
	Items             []ItemPreference  `json:"items"`
	ExternalReference string            `json:"external_reference"`
	BackURLs          backURLs          `json:"back_urls"`
	AutoReturn        string            `json:"auto_return,omitempty"`
	NotificationURL   string            `json:"notification_url"`
	PaymentMethods    *metodosPagamento `json:"payment_methods,omitempty"`
	Expires           bool              `json:"expires,omitempty"`
	ExpirationDateTo  string            `json:"expiration_date_to,omitempty"`
	DateOfExpiration  string            `json:"date_of_expiration,omitempty"` // validade do Pix
}

type Preference struct {
	ID               string `json:"id"`
	InitPoint        string `json:"init_point"`
	SandboxInitPoint string `json:"sandbox_init_point"`
}

// criarPreference cria uma sessão de checkout (Checkout Pro). Não move
// dinheiro — só gera o link para onde o comprador é redirecionado.
//
// auto_return fica de fora de propósito: o MP só aceita esse campo
// quando back_urls.success é uma URL pública (rejeita localhost), o que
// quebraria em dev. Sem ele, o comprador só precisa clicar em "voltar ao
// site" após pagar em vez de ser redirecionado automaticamente — o
// webhook (que de fato confirma o pagamento) não depende disso.
func (c *Cliente) criarPreference(req criarPreferenceRequest) (*Preference, error) {
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
	DateOfExpiration  string      `json:"date_of_expiration"`
	// Pix: o QR e o "copia e cola".
	PointOfInteraction struct {
		TransactionData struct {
			QRCode       string `json:"qr_code"`
			QRCodeBase64 string `json:"qr_code_base64"`
		} `json:"transaction_data"`
	} `json:"point_of_interaction"`
	// Cartão com desafio 3DS (status_detail = pending_challenge).
	ThreeDSInfo struct {
		ExternalResourceURL string `json:"external_resource_url"`
		Creq                string `json:"creq"`
	} `json:"three_ds_info"`
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

type identificacao struct {
	Type   string `json:"type"`
	Number string `json:"number"`
}

type pagador struct {
	Email          string         `json:"email"`
	Identification *identificacao `json:"identification,omitempty"`
}

// CriarPagamentoRequest é o corpo de POST /v1/payments (Checkout
// Transparente): Pix com QR na tela ou cartão tokenizado pelo formulário
// do MP, com 3DS no modo "optional" (o banco decide quando desafiar).
type CriarPagamentoRequest struct {
	TransactionAmount   float64 `json:"transaction_amount"`
	Description         string  `json:"description"`
	PaymentMethodID     string  `json:"payment_method_id"`
	Token               string  `json:"token,omitempty"`
	IssuerID            string  `json:"issuer_id,omitempty"`
	Installments        int     `json:"installments,omitempty"`
	Payer               pagador `json:"payer"`
	ExternalReference   string  `json:"external_reference"`
	NotificationURL     string  `json:"notification_url,omitempty"`
	DateOfExpiration    string  `json:"date_of_expiration,omitempty"`
	ThreeDSecureMode    string  `json:"three_d_secure_mode,omitempty"`
	StatementDescriptor string  `json:"statement_descriptor,omitempty"`
}

func (c *Cliente) CriarPagamento(req CriarPagamentoRequest, chave string) (*Pagamento, error) {
	var resp Pagamento
	if err := c.requisitarCom(http.MethodPost, "/v1/payments", req, &resp, chave); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelarPagamento cancela um pagamento pendente (Pix não pago).
func (c *Cliente) CancelarPagamento(id string) error {
	return c.requisitar(http.MethodPut, "/v1/payments/"+id, map[string]string{"status": "cancelled"}, nil)
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
	return c.SolicitarReembolsoCom(paymentID, valorCentavos, "")
}

// SolicitarReembolsoCom é o estorno com chave de idempotência.
func (c *Cliente) SolicitarReembolsoCom(paymentID string, valorCentavos *int64, chave string) (*Reembolso, error) {
	req := reembolsoRequest{}
	if valorCentavos != nil {
		valor := float64(*valorCentavos) / 100
		req.Amount = &valor
	}

	var resp Reembolso
	if err := c.requisitarCom(http.MethodPost, "/v1/payments/"+paymentID+"/refunds", req, &resp, chave); err != nil {
		return nil, err
	}
	return &resp, nil
}
