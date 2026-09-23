package domain

import "time"

type TipoLancamento string

const (
	LancamentoVendaPreco              TipoLancamento = "venda_preco"
	LancamentoTaxaPlataforma          TipoLancamento = "taxa_plataforma"
	LancamentoGarantia                TipoLancamento = "garantia"
	LancamentoTaxaProcessador         TipoLancamento = "taxa_processador"
	LancamentoReembolsoPreco          TipoLancamento = "reembolso_preco"
	LancamentoReembolsoTaxa           TipoLancamento = "reembolso_taxa"
	LancamentoReembolsoGarantia       TipoLancamento = "reembolso_garantia"
	LancamentoCustoProcessadorPerdido TipoLancamento = "custo_processador_perdido"
	LancamentoRepasse                 TipoLancamento = "repasse"
)

// Lancamento é o ledger financeiro (seção 7.6/7.7): toda entrada e saída
// de dinheiro vira uma linha aqui, nunca é só implícita em um status.
type Lancamento struct {
	ID            int64 `gorm:"primaryKey"`
	Tipo          TipoLancamento
	ValorCentavos int64
	Sinal         string // "+" ou "-"
	EventoID      *int64
	OrganizadorID *int64
	PedidoID      *int64
	ItemID        *int64
	CriadoEm      time.Time
}

func (Lancamento) TableName() string {
	return "lancamentos"
}
