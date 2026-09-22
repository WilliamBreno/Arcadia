package repository

import (
	"strconv"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// ConfigPlataformaRepository lê configuração global (taxas, garantia,
// dias de repasse, minutos de reserva) da tabela config_plataforma.
type ConfigPlataformaRepository struct {
	db *gorm.DB
}

func NovoConfigPlataformaRepository(db *gorm.DB) *ConfigPlataformaRepository {
	return &ConfigPlataformaRepository{db: db}
}

// BuscarInt64 lê uma chave de configuração e converte para int64.
// Usado para valores financeiros, sempre em centavos.
func (r *ConfigPlataformaRepository) BuscarInt64(chave string) (int64, error) {
	var registro domain.ConfigPlataforma
	if err := r.db.First(&registro, "chave = ?", chave).Error; err != nil {
		return 0, err
	}
	return strconv.ParseInt(registro.Valor, 10, 64)
}
