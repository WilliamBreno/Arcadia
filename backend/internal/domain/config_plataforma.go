package domain

import "time"

// ConfigPlataforma é um par chave/valor de configuração global (taxas,
// garantia, dias de repasse, minutos de reserva). Valores são guardados
// como texto e convertidos pelo chamador conforme o tipo esperado.
type ConfigPlataforma struct {
	Chave        string    `gorm:"primaryKey;column:chave"`
	Valor        string    `gorm:"column:valor"`
	Descricao    string    `gorm:"column:descricao"`
	AtualizadoEm time.Time `gorm:"column:atualizado_em"`
}

func (ConfigPlataforma) TableName() string {
	return "config_plataforma"
}

// Chaves conhecidas de config_plataforma (seção 2.1 e 7.6 do plano).
const (
	// Taxa por faixa de preço: "limite:taxa,...,*:taxa" em centavos (item 5.1).
	ChaveTaxaFaixas             = "TAXA_FAIXAS"
	ChaveGarantiaPercentualBP   = "GARANTIA_PERCENTUAL_BP" // 500 = 5%
	ChaveGarantiaMinCentavos    = "GARANTIA_MIN_CENTAVOS"
	ChaveGarantiaMaxCentavos    = "GARANTIA_MAX_CENTAVOS"
	ChaveRepasseDiasUteis       = "REPASSE_DIAS_UTEIS"
	ChaveTaxaReferenciaCartaoBP = "TAXA_REFERENCIA_CARTAO_BP" // só para estimativa no painel
	ChaveReservaMinutos         = "RESERVA_MINUTOS"
	ChaveCDCReembolsaTaxa       = "CDC_REEMBOLSA_TAXA"
)
