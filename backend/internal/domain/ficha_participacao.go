package domain

import (
	"time"

	"gorm.io/datatypes"
)

type StatusFicha string

const (
	StatusFichaRascunho    StatusFicha = "rascunho"
	StatusFichaPendente    StatusFicha = "pendente"
	StatusFichaAprovado    StatusFicha = "aprovado"
	StatusFichaRejeitado   StatusFicha = "rejeitado"
	StatusFichaListaEspera StatusFicha = "lista_espera"
	StatusFichaDesistiu    StatusFicha = "desistiu"
)

type TipoApresentacao string

const (
	TipoApresentacaoCosplay TipoApresentacao = "cosplay"
	TipoApresentacaoDanca   TipoApresentacao = "danca"
	TipoApresentacaoCanto   TipoApresentacao = "canto"
	TipoApresentacaoAtuacao TipoApresentacao = "atuacao"
)

// FichaParticipacao é a inscrição de um jurado ou participante num
// evento. Campos privados (Telefone, ResponsavelContato,
// AutorizacaoResponsavelURL) nunca são expostos a jurados — ver
// filtragem na área do jurado (item 1.6).
type FichaParticipacao struct {
	ID                        int64 `gorm:"primaryKey"`
	EventoID                  int64
	UsuarioID                 int64
	Papel                     Papel
	Origem                    OrigemPapel
	Nome                      string
	NomeArtistico             string
	Instagram                 string
	DataNascimento            *time.Time
	FotoURL                   string
	Telefone                  string
	Status                    StatusFicha
	MotivoRejeicao            string
	OrdemApresentacao         *int
	TipoApresentacao          *TipoApresentacao
	Dados                     datatypes.JSON
	ResponsavelNome           string
	ResponsavelContato        string
	AutorizacaoResponsavelURL string
	CriadoEm                  time.Time
}

func (FichaParticipacao) TableName() string {
	return "fichas_participacao"
}

// MenorDeIdade calcula a partir de DataNascimento (regra da seção 7.12).
func (f *FichaParticipacao) MenorDeIdade() bool {
	if f.DataNascimento == nil {
		return false
	}
	dezoitoAnosAtras := time.Now().AddDate(-18, 0, 0)
	return f.DataNascimento.After(dezoitoAnosAtras)
}
