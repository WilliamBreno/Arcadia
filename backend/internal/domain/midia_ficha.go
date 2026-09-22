package domain

import "time"

type TipoMidiaFicha string

const (
	TipoMidiaFotoReferencia TipoMidiaFicha = "foto_referencia"
	TipoMidiaFotoCosplay    TipoMidiaFicha = "foto_cosplay"
	TipoMidiaAudio          TipoMidiaFicha = "audio"
)

type MidiaFicha struct {
	ID       int64 `gorm:"primaryKey"`
	FichaID  int64
	Tipo     TipoMidiaFicha
	URL      string
	Ordem    int
	CriadoEm time.Time
}

func (MidiaFicha) TableName() string {
	return "midias_ficha"
}
