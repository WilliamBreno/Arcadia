package domain

import "time"

type StatusSolicitacao string

const (
	SolicitacaoPendente    StatusSolicitacao = "pendente"
	SolicitacaoAprovada    StatusSolicitacao = "aprovada"
	SolicitacaoRejeitada   StatusSolicitacao = "rejeitada"
	SolicitacaoListaEspera StatusSolicitacao = "lista_espera"
)

// SolicitacaoPlateia é o pedido de participação de um usuário num evento
// com aprovacao_manual: só com status "aprovada" ele pode comprar/se
// cadastrar. "lista_espera" = aprovado pelo organizador, mas sem vaga agora.
type SolicitacaoPlateia struct {
	ID         int64 `gorm:"primaryKey"`
	EventoID   int64
	UsuarioID  int64
	Status     StatusSolicitacao
	CriadoEm   time.Time
	DecididoEm *time.Time
}

func (SolicitacaoPlateia) TableName() string {
	return "solicitacoes_plateia"
}
