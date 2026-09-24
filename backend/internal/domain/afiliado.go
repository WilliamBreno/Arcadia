package domain

import "time"

// Afiliado é um divulgador do evento: um link com ?ref=<codigo> que atribui
// os pedidos a ele (só atribuição e estatística; não há comissão automática).
type Afiliado struct {
	ID       int64 `gorm:"primaryKey"`
	EventoID int64
	Nome     string
	Codigo   string
	Ativo    bool
	CriadoEm time.Time
}

func (Afiliado) TableName() string { return "afiliados" }
