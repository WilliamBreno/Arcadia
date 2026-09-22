package domain

import "time"

type TipoPessoa string

const (
	TipoPessoaFisica   TipoPessoa = "pf"
	TipoPessoaJuridica TipoPessoa = "pj"
)

type StatusOrganizador string

const (
	StatusOrganizadorPendente StatusOrganizador = "pendente"
	StatusOrganizadorAtivo    StatusOrganizador = "ativo"
	StatusOrganizadorSuspenso StatusOrganizador = "suspenso"
)

// Organizador é o perfil que um usuário assume para criar e gerenciar
// eventos. Um usuário tem no máximo um perfil de organizador (decisão
// registrada na seção 14 do plano).
type Organizador struct {
	ID           int64 `gorm:"primaryKey"`
	UsuarioID    int64
	NomePublico  string
	Slug         string
	Descricao    string
	LogoURL      string
	TipoPessoa   TipoPessoa
	Documento    string
	ChavePix     string
	TipoChavePix string
	Instagram    string
	Site         string
	Status       StatusOrganizador `gorm:"default:pendente"`
	CriadoEm     time.Time
}

func (Organizador) TableName() string {
	return "organizadores"
}
