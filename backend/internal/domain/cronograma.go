package domain

import "time"

// CronogramaItem é uma atividade da programação do evento (palestra,
// abertura, desfile...). Público: aparece na página do evento.
type CronogramaItem struct {
	ID        int64 `gorm:"primaryKey"`
	EventoID  int64
	Titulo    string
	Descricao string
	Local     string
	InicioEm  time.Time
	FimEm     *time.Time
	CriadoEm  time.Time
}

func (CronogramaItem) TableName() string {
	return "cronograma_itens"
}
