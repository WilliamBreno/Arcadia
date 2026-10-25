package service

import (
	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/pagamento"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// ResumoMetodos é o quadro "Pix x cartão" do painel financeiro (item 5.2):
// mostra ao organizador quanto ele pagou de taxa em cada método e quanto
// economizou com o Pix. É só informativo — não entra em nenhum cálculo de
// repasse.
type ResumoMetodos struct {
	PixQuantidade       int64 `json:"pix_quantidade"`
	PixValorCentavos    int64 `json:"pix_valor_centavos"`
	PixTaxaCentavos     int64 `json:"pix_taxa_centavos"`
	CartaoQuantidade    int64 `json:"cartao_quantidade"`
	CartaoValorCentavos int64 `json:"cartao_valor_centavos"`
	CartaoTaxaCentavos  int64 `json:"cartao_taxa_centavos"`
	// TaxaCartaoBP é a taxa do cartão usada na estimativa: a média real do
	// evento quando houve venda no cartão, senão a referência da config.
	TaxaCartaoBP        int64 `json:"taxa_cartao_bp"`
	TaxaCartaoEstimada  bool  `json:"taxa_cartao_estimada"`
	EconomiaPixCentavos int64 `json:"economia_pix_centavos"`
}

// calcularEconomiaPix: quanto as vendas no Pix custariam no cartão, menos
// o que de fato custaram. Nunca negativo. Função pura, testada.
func calcularEconomiaPix(pixValor, pixTaxa, cartaoValor, cartaoTaxa, referenciaBP int64) (economia, taxaCartaoBP int64, estimada bool) {
	taxaCartaoBP, estimada = referenciaBP, true
	if cartaoValor > 0 {
		taxaCartaoBP, estimada = cartaoTaxa*10000/cartaoValor, false
	}
	economia = pixValor*taxaCartaoBP/10000 - pixTaxa
	if economia < 0 {
		economia = 0
	}
	return economia, taxaCartaoBP, estimada
}

// DefinirPagamentos liga o quadro Pix x cartão (item 5.2).
func (s *RepasseService) DefinirPagamentos(p *repository.PagamentoRepository) { s.pagamentos = p }

func (s *RepasseService) resumoMetodos(eventoID int64) (*ResumoMetodos, error) {
	if s.pagamentos == nil {
		return nil, nil
	}
	linhas, err := s.pagamentos.ResumoPorMetodo(eventoID)
	if err != nil {
		return nil, err
	}
	r := &ResumoMetodos{}
	for _, l := range linhas {
		if l.Metodo == pagamento.MetodoPix {
			r.PixQuantidade, r.PixValorCentavos, r.PixTaxaCentavos = l.Quantidade, l.ValorCentavos, l.TaxaCentavos
		} else {
			r.CartaoQuantidade += l.Quantidade
			r.CartaoValorCentavos += l.ValorCentavos
			r.CartaoTaxaCentavos += l.TaxaCentavos
		}
	}
	referencia, err := s.config.BuscarInt64(domain.ChaveTaxaReferenciaCartaoBP)
	if err != nil {
		referencia = 398
	}
	r.EconomiaPixCentavos, r.TaxaCartaoBP, r.TaxaCartaoEstimada = calcularEconomiaPix(
		r.PixValorCentavos, r.PixTaxaCentavos, r.CartaoValorCentavos, r.CartaoTaxaCentavos, referencia)
	return r, nil
}
