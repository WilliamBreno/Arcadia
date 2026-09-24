package domain

import "time"

// CriterioAvaliacao é um quesito de nota (seção 3.3): peso, escala
// (nota_min..nota_max de passo em passo) e, opcionalmente, restrito a um
// tipo de apresentação (ex.: "fidelidade" só para cosplay).
type CriterioAvaliacao struct {
	ID               int64 `gorm:"primaryKey"`
	EventoID         int64
	Nome             string
	Peso             float64
	NotaMin          float64
	NotaMax          float64
	Passo            float64
	TipoApresentacao *TipoApresentacao
	Ordem            int
	CriadoEm         time.Time
}

func (CriterioAvaliacao) TableName() string {
	return "criterios_avaliacao"
}

type Avaliacao struct {
	ID              int64 `gorm:"primaryKey"`
	FichaID         int64
	JuradoUsuarioID int64
	CriterioID      int64
	Nota            float64
	Comentario      string
	Finalizada      bool
	CriadoEm        time.Time
}

func (Avaliacao) TableName() string {
	return "avaliacoes"
}
