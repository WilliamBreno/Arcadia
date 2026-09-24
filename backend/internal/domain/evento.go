package domain

import "time"

type Visibilidade string

const (
	VisibilidadePublico    Visibilidade = "publico"
	VisibilidadeNaoListado Visibilidade = "nao_listado"
	VisibilidadePrivado    Visibilidade = "privado"
)

type TipoAcesso string

const (
	TipoAcessoIngresso TipoAcesso = "ingresso"
	TipoAcessoCadastro TipoAcesso = "cadastro"
)

type StatusEvento string

const (
	StatusEventoRascunho  StatusEvento = "rascunho"
	StatusEventoPublicado StatusEvento = "publicado"
	StatusEventoEncerrado StatusEvento = "encerrado"
	StatusEventoCancelado StatusEvento = "cancelado"
)

type ModoParticipantes string

const (
	ModoParticipantesNenhum          ModoParticipantes = "nenhum"
	ModoParticipantesConvite         ModoParticipantes = "convite"
	ModoParticipantesInscricaoAberta ModoParticipantes = "inscricao_aberta"
	ModoParticipantesAmbos           ModoParticipantes = "ambos"
)

// Evento segue a máquina de estados da seção 6 do plano:
// rascunho -> publicado -> encerrado; cancelado é terminal a partir de
// qualquer estado não-encerrado. "Vendas abertas" e "em andamento" são
// derivados das datas, não guardados aqui.
type Evento struct {
	ID                        int64 `gorm:"primaryKey"`
	OrganizadorID             int64
	LocalID                   *int64
	Titulo                    string
	Slug                      string
	Descricao                 string
	Categoria                 string
	CapaURL                   string
	InicioEm                  *time.Time
	FimEm                     *time.Time
	Timezone                  string
	ClassificacaoEtaria       string
	Visibilidade              Visibilidade
	TipoAcesso                TipoAcesso
	Status                    StatusEvento
	ModoParticipantes         ModoParticipantes
	InscricaoTalentosInicio   *time.Time
	InscricaoTalentosFim      *time.Time
	CapacidadeTotal           *int
	GarantiaHabilitada        bool
	AprovacaoManual           bool
	QRRotativo                bool
	ResultadoLiberadoEm       *time.Time
	PoliticaCancelamentoTexto string
	MaxItensPorPedido         int
	PublicadoEm               *time.Time
	CanceladoEm               *time.Time
	MotivoCancelamento        string
	CriadoEm                  time.Time
}

func (Evento) TableName() string {
	return "eventos"
}
