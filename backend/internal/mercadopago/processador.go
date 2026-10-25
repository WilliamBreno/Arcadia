package mercadopago

import (
	"fmt"
	"strconv"

	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
)

// Processador adapta o Cliente do Mercado Pago ao contrato
// pagamento.Processador (item 5.2).
type Processador struct {
	cliente *Cliente
}

func NovoProcessador(c *Cliente) *Processador { return &Processador{cliente: c} }

var _ pagamento.Processador = (*Processador)(nil)

func (p *Processador) Nome() string     { return "mercadopago" }
func (p *Processador) Habilitado() bool { return p.cliente.Habilitado() }

// Tipos de pagamento do Checkout Pro. Boleto ("ticket") e lotérica ("atm")
// ficam sempre fora: compensam em dias e a reserva dura minutos — o
// comprador pagaria por um ingresso já devolvido ao estoque.
var (
	tiposSempreExcluidos = []string{"ticket", "atm"}
	tiposForaDoPix       = []string{"credit_card", "debit_card", "prepaid_card"}
	tiposForaDoCartao    = []string{"bank_transfer"} // o Pix é "bank_transfer" no MP
)

// tiposExcluidos devolve o que esconder no checkout do MP para o método
// escolhido pelo comprador. Função pura, testada.
func tiposExcluidos(metodo string) []string {
	excluidos := append([]string{}, tiposSempreExcluidos...)
	switch metodo {
	case pagamento.MetodoPix:
		excluidos = append(excluidos, tiposForaDoPix...)
	case pagamento.MetodoCartao:
		excluidos = append(excluidos, tiposForaDoCartao...)
	}
	return excluidos
}

func (p *Processador) CriarCobranca(n pagamento.NovaCobranca) (*pagamento.Cobranca, error) {
	itens := make([]ItemPreference, 0, len(n.Itens))
	for _, it := range n.Itens {
		itens = append(itens, ItemPreference{
			Title: it.Titulo, Quantity: 1, UnitPrice: float64(it.ValorCentavos) / 100, CurrencyID: "BRL",
		})
	}
	excluidos := make([]tipoExcluido, 0)
	for _, t := range tiposExcluidos(n.Metodo) {
		excluidos = append(excluidos, tipoExcluido{ID: t})
	}
	req := criarPreferenceRequest{
		Items:             itens,
		ExternalReference: n.Referencia,
		BackURLs:          backURLs{Success: n.URLSucesso, Failure: n.URLFalha, Pending: n.URLPendente},
		NotificationURL:   n.URLNotificacao,
		PaymentMethods:    &metodosPagamento{ExcludedPaymentTypes: excluidos, Installments: 1},
	}
	if !n.ExpiraEm.IsZero() {
		// Validade do Pix (e da própria preference) = fim da reserva.
		exp := n.ExpiraEm.Format("2006-01-02T15:04:05.000-07:00")
		req.Expires = true
		req.ExpirationDateTo = exp
		req.DateOfExpiration = exp
	}
	pref, err := p.cliente.criarPreference(req)
	if err != nil {
		return nil, err
	}
	url := pref.InitPoint
	if url == "" {
		url = pref.SandboxInitPoint
	}
	return &pagamento.Cobranca{ID: pref.ID, URL: url}, nil
}

func (p *Processador) BuscarPagamento(id string) (*pagamento.Pagamento, error) {
	mp, err := p.cliente.BuscarPagamento(id)
	if err != nil {
		return nil, err
	}
	metodo := pagamento.MetodoCartao
	if mp.PaymentTypeID == "bank_transfer" || mp.PaymentMethodID == "pix" {
		metodo = pagamento.MetodoPix
	}
	return &pagamento.Pagamento{
		ID:                      strconv.FormatInt(mp.ID, 10),
		Referencia:              mp.ExternalReference,
		Status:                  mp.Status,
		Metodo:                  metodo,
		ValorCentavos:           int64(mp.TransactionAmount*100 + 0.5),
		TaxaProcessadorCentavos: mp.TaxaProcessadorCentavos(),
		Bruto:                   mp,
	}, nil
}

func (p *Processador) Reembolsar(pagamentoID string, valorCentavos int64) (*pagamento.Reembolso, error) {
	r, err := p.cliente.SolicitarReembolso(pagamentoID, &valorCentavos)
	if err != nil {
		return nil, err
	}
	return &pagamento.Reembolso{ID: fmt.Sprintf("%d", r.ID)}, nil
}
