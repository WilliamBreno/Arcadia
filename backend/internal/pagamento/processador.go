// Package pagamento define o contrato de um processador de pagamento
// (item 5.2). Os serviços (checkout, cancelamento) falam só com esta
// interface; o Mercado Pago é a primeira implementação. Para somar outro
// processador (Pagar.me, Efí...), basta implementar Processador e
// registrá-lo — as regras de dinheiro não mudam.
package pagamento

import "time"

// Status canônico de um pagamento aprovado. É o mesmo texto que o
// Mercado Pago usa e que já está gravado em pagamentos.status.
const StatusAprovado = "approved"

// Métodos de pagamento que o comprador escolhe no checkout.
const (
	MetodoPix    = "pix"
	MetodoCartao = "cartao"
)

// Item é uma linha da cobrança (um ingresso).
type Item struct {
	Titulo        string
	ValorCentavos int64
}

// NovaCobranca é o pedido de checkout enviado ao processador.
type NovaCobranca struct {
	Referencia     string // id do pedido (volta no webhook)
	Itens          []Item
	Metodo         string    // MetodoPix ou MetodoCartao
	ExpiraEm       time.Time // fim da reserva: o Pix não pode ser pago depois disso
	URLSucesso     string
	URLFalha       string
	URLPendente    string
	URLNotificacao string
}

// Cobranca é o checkout criado: para onde mandar o comprador.
type Cobranca struct {
	ID  string
	URL string
}

// Pagamento é o pagamento consultado no processador, já normalizado.
type Pagamento struct {
	ID                      string
	Referencia              string
	Status                  string // StatusAprovado quando aprovado
	Metodo                  string // MetodoPix ou MetodoCartao
	ValorCentavos           int64
	TaxaProcessadorCentavos int64
	Bruto                   any // resposta original, guardada para auditoria
}

// Reembolso é o estorno confirmado pelo processador.
type Reembolso struct {
	ID string
}

// Processador é o contrato que todo processador de pagamento cumpre.
type Processador interface {
	Nome() string
	Habilitado() bool
	CriarCobranca(NovaCobranca) (*Cobranca, error)
	// BuscarPagamento SEMPRE é chamado com o id recebido no webhook — nunca
	// confiar em valor/status do corpo da notificação (seção 7.3).
	BuscarPagamento(id string) (*Pagamento, error)
	// Reembolsar estorna valorCentavos (parcial) de um pagamento.
	Reembolsar(pagamentoID string, valorCentavos int64) (*Reembolso, error)
}
