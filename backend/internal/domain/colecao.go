package domain

import "time"

// Colecao é uma lista de eventos montada pelo admin (item 6.2).
type Colecao struct {
	ID        int64 `gorm:"primaryKey"`
	Slug      string
	Titulo    string
	Descricao string
	Ordem     int
	Ativa     bool
	CriadoEm  time.Time
}

func (Colecao) TableName() string { return "colecoes" }
