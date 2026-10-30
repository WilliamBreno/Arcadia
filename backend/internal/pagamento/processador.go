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
	// Item 7.6: máximo de parcelas oferecido no site do Mercado Pago (plano B).
	MaxParcelas int
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

// NovoPagamento é o pagamento criado direto pela API, sem sair da página
// do Evve (item 7.1). Pix: só e-mail e validade. Cartão: o token vem do
// formulário seguro do processador (o número do cartão nunca passa pelo
// Evve); o documento do titular vai direto ao processador e não é guardado.
type NovoPagamento struct {
	Referencia      string
	Descricao       string
	ValorCentavos   int64
	Metodo          string
	EmailPagador    string
	ExpiraEm        time.Time // Pix: validade do QR
	CartaoToken     string
	BandeiraID      string // ex.: "visa", "master"
	EmissorID       string
	DocumentoTipo   string // "CPF"
	DocumentoNumero string
	// Item 7.6: parcelas no cartão (juros do comprador, cobrados pelo MP).
	Parcelas          int
	ChaveIdempotencia string
	URLNotificacao    string
}

// Status de um pagamento direto.
const (
	StatusPendente  = "pending"
	StatusRecusado  = "rejected"
	StatusEmAnalise = "in_process"
	// DetalheDesafio3DS: o banco pediu a confirmação do titular (3DS).
	DetalheDesafio3DS = "pending_challenge"
)

// PagamentoCriado é a resposta do processador ao criar o pagamento.
type PagamentoCriado struct {
	ID            string
	Status        string
	StatusDetalhe string
	PixCopiaCola  string
	PixQRBase64   string
	PixExpiraEm   time.Time
	// Desafio 3DS: o frontend abre a tela do banco com estes dados.
	DesafioURL  string
	DesafioCreq string
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
	// CriarPagamento cobra direto na API, com o comprador na página do Evve.
	CriarPagamento(NovoPagamento) (*PagamentoCriado, error)
	// CancelarPagamento invalida um Pix ainda não pago.
	CancelarPagamento(id string) error
	// Devolver estorna o pagamento inteiro uma única vez (a chave impede
	// estorno em dobro quando o pedido é repetido).
	Devolver(pagamentoID, chave string) (*Reembolso, error)
}
