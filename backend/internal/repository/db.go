package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Conectar abre a conexão com o Postgres via GORM a partir da URL de conexão.
func Conectar(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
}
