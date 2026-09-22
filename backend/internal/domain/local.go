package domain

import "time"

// Local é um endereço físico cadastrado por um organizador, usado por
// eventos presenciais (evento.local_id nulo = evento online).
type Local struct {
	ID            int64 `gorm:"primaryKey"`
	OrganizadorID int64
	Nome          string
	Logradouro    string
	Numero        string
	Bairro        string
	Cidade        string
	UF            string `gorm:"column:uf"`
	CEP           string `gorm:"column:cep"`
	Latitude      *float64
	Longitude     *float64
	Capacidade    *int
	Observacoes   string
	CriadoEm      time.Time
}

func (Local) TableName() string {
	return "locais"
}
