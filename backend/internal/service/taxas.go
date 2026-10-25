package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Taxas da plataforma (seção 2.1, tabela aprovada em 24/10/2026):
// valor fixo por faixa de preço, garantia percentual com mínimo e teto,
// e opção de o organizador absorver a taxa. Todo cálculo de taxa passa
// por aqui — checkout, página pública e publicação usam as mesmas regras.

// FaixaTaxa: preços até AteCentavos (inclusive) pagam TaxaCentavos.
// AteCentavos < 0 significa "sem limite" (última faixa).
type FaixaTaxa struct {
	AteCentavos  int64 `json:"ate_centavos"`
	TaxaCentavos int64 `json:"taxa_centavos"`
}

// RegraTaxas é a configuração global (config_plataforma).
type RegraTaxas struct {
	Faixas               []FaixaTaxa
	GarantiaPercentualBP int64 // pontos-base: 500 = 5%
	GarantiaMinCentavos  int64
	GarantiaMaxCentavos  int64
}

// ParseFaixas lê o formato "3000:99,8000:199,20000:399,*:699".
// As faixas precisam estar em ordem crescente e a última ser "*".
func ParseFaixas(s string) ([]FaixaTaxa, error) {
	var faixas []FaixaTaxa
	var anterior int64 = -1
	partes := strings.Split(strings.TrimSpace(s), ",")
	for i, p := range partes {
		limite, taxa, ok := strings.Cut(strings.TrimSpace(p), ":")
		if !ok {
			return nil, fmt.Errorf("faixa inválida %q", p)
		}
		t, err := strconv.ParseInt(strings.TrimSpace(taxa), 10, 64)
		if err != nil || t < 0 {
			return nil, fmt.Errorf("taxa inválida na faixa %q", p)
		}
		if strings.TrimSpace(limite) == "*" {
			if i != len(partes)-1 {
				return nil, fmt.Errorf("a faixa \"*\" precisa ser a última")
			}
			faixas = append(faixas, FaixaTaxa{AteCentavos: -1, TaxaCentavos: t})
			continue
		}
		l, err := strconv.ParseInt(strings.TrimSpace(limite), 10, 64)
		if err != nil || l <= anterior {
			return nil, fmt.Errorf("limite inválido ou fora de ordem na faixa %q", p)
		}
		anterior = l
		faixas = append(faixas, FaixaTaxa{AteCentavos: l, TaxaCentavos: t})
	}
	if len(faixas) == 0 || faixas[len(faixas)-1].AteCentavos != -1 {
		return nil, fmt.Errorf("a última faixa precisa ser \"*\"")
	}
	return faixas, nil
}

// TaxaPorFaixa: preço zero (gratuito) nunca paga taxa.
func (r RegraTaxas) TaxaPorFaixa(precoCentavos int64) int64 {
	if precoCentavos <= 0 {
		return 0
	}
	for _, f := range r.Faixas {
		if f.AteCentavos < 0 || precoCentavos <= f.AteCentavos {
			return f.TaxaCentavos
		}
	}
	return 0
}

// Garantia: percentual do preço arredondado ao centavo mais próximo,
// depois mínimo e teto. Item gratuito não tem garantia (não há o que
// devolver).
func (r RegraTaxas) Garantia(precoCentavos int64) int64 {
	if precoCentavos <= 0 {
		return 0
	}
	g := (precoCentavos*r.GarantiaPercentualBP + 5000) / 10000
	if g < r.GarantiaMinCentavos {
		g = r.GarantiaMinCentavos
	}
	if r.GarantiaMaxCentavos > 0 && g > r.GarantiaMaxCentavos {
		g = r.GarantiaMaxCentavos
	}
	return g
}

// TaxasEvento é a regra aplicada a um evento específico: a global mais
// o limite regional (seção 7.8) e quem paga a taxa.
type TaxasEvento struct {
	Regra RegraTaxas
	// LimiteBP: teto da taxa em pontos-base do preço (ex.: AL = 1000 = 10%).
	// Só vale quando o comprador paga a taxa.
	LimiteBP  *int64
	Absorvida bool // organizador paga a taxa (taxa_paga_por = organizador)
}

// ValoresItem é o resultado do cálculo de um item.
type ValoresItem struct {
	TaxaCentavos     int64 // receita da plataforma
	TaxaAbsorvida    bool  // true: a taxa sai do repasse, não do comprador
	GarantiaCentavos int64 // 0 se não contratada ou item gratuito
	TotalCentavos    int64 // o que o comprador paga
}

// Calcular aplica a seção 7.1 a um item com o preço já com desconto.
func (t TaxasEvento) Calcular(precoCentavos int64, garantiaContratada bool) ValoresItem {
	taxa := t.Regra.TaxaPorFaixa(precoCentavos)
	if !t.Absorvida && t.LimiteBP != nil {
		maximo := precoCentavos * *t.LimiteBP / 10000
		if taxa > maximo {
			taxa = maximo
		}
	}
	v := ValoresItem{TaxaCentavos: taxa, TaxaAbsorvida: t.Absorvida && taxa > 0}
	if garantiaContratada {
		v.GarantiaCentavos = t.Regra.Garantia(precoCentavos)
	}
	v.TotalCentavos = precoCentavos + v.GarantiaCentavos
	if !v.TaxaAbsorvida {
		v.TotalCentavos += taxa
	}
	return v
}

// TaxasService carrega a regra global e a regional de um evento.
type TaxasService struct {
	config *repository.ConfigPlataformaRepository
	locais *repository.LocalRepository
	regras *repository.RegraRegionalRepository
}

func NovoTaxasService(config *repository.ConfigPlataformaRepository, locais *repository.LocalRepository, regras *repository.RegraRegionalRepository) *TaxasService {
	return &TaxasService{config: config, locais: locais, regras: regras}
}

func (s *TaxasService) Regra() (RegraTaxas, error) {
	texto, err := s.config.Buscar(domain.ChaveTaxaFaixas)
	if err != nil {
		return RegraTaxas{}, fmt.Errorf("config %s ausente: %w", domain.ChaveTaxaFaixas, err)
	}
	faixas, err := ParseFaixas(texto)
	if err != nil {
		return RegraTaxas{}, fmt.Errorf("config %s inválida: %w", domain.ChaveTaxaFaixas, err)
	}
	r := RegraTaxas{Faixas: faixas}
	for chave, destino := range map[string]*int64{
		domain.ChaveGarantiaPercentualBP: &r.GarantiaPercentualBP,
		domain.ChaveGarantiaMinCentavos:  &r.GarantiaMinCentavos,
		domain.ChaveGarantiaMaxCentavos:  &r.GarantiaMaxCentavos,
	} {
		v, err := s.config.BuscarInt64(chave)
		if err != nil {
			return RegraTaxas{}, fmt.Errorf("config %s ausente: %w", chave, err)
		}
		*destino = v
	}
	return r, nil
}

// RegraRegionalDoEvento devolve a regra da UF/cidade do local do evento,
// ou nil se não houver (evento online, sem regra, ou dispensado pelo
// tamanho do público).
func (s *TaxasService) RegraRegionalDoEvento(evento *domain.Evento) *domain.RegraRegional {
	if evento.LocalID == nil || s.locais == nil || s.regras == nil {
		return nil
	}
	local, err := s.locais.BuscarPorID(*evento.LocalID)
	if err != nil {
		return nil
	}
	regra, err := s.regras.Buscar(local.UF, local.Cidade)
	if err != nil {
		return nil
	}
	if regra.ExcecaoPublicoAte != nil && evento.CapacidadeTotal != nil && *evento.CapacidadeTotal <= *regra.ExcecaoPublicoAte {
		return nil
	}
	return regra
}

func (s *TaxasService) ParaEvento(evento *domain.Evento) (TaxasEvento, error) {
	regra, err := s.Regra()
	if err != nil {
		return TaxasEvento{}, err
	}
	t := TaxasEvento{Regra: regra, Absorvida: evento.TaxaPagaPor == domain.TaxaPagaPorOrganizador}
	if rr := s.RegraRegionalDoEvento(evento); rr != nil && rr.TaxaMaximaPercentual != nil {
		bp := int64(math.Round(*rr.TaxaMaximaPercentual * 100))
		t.LimiteBP = &bp
	}
	return t, nil
}
