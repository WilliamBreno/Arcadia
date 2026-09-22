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
	ChaveTaxaIngressoCentavos = "TAXA_INGRESSO_CENTAVOS"
	ChaveTaxaCadastroCentavos = "TAXA_CADASTRO_CENTAVOS"
	ChaveGarantiaCentavos     = "GARANTIA_CENTAVOS"
	ChaveRepasseDiasUteis     = "REPASSE_DIAS_UTEIS"
	ChaveReservaMinutos       = "RESERVA_MINUTOS"
)
