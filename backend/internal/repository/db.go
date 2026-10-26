package repository

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Conectar abre a conexão com o Postgres via GORM a partir da URL de conexão.
// maxConexoes limita o pool (item 5.5): sem limite, um pico de compras abria
// uma conexão por requisição até o Postgres recusar ("too many clients").
// As requisições excedentes esperam na fila do pool em vez de falhar.
func Conectar(databaseURL string, maxConexoes int) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if maxConexoes <= 0 {
		maxConexoes = 20
	}
	sqlDB.SetMaxOpenConns(maxConexoes)
	sqlDB.SetMaxIdleConns(maxConexoes / 2)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	return db, nil
}
