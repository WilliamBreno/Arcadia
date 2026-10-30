package domain

import "time"

type Visibilidade string

const (
	VisibilidadePublico    Visibilidade = "publico"
	VisibilidadeNaoListado Visibilidade = "nao_listado"
	VisibilidadePrivado    Visibilidade = "privado"
)

// Quem paga a taxa da plataforma (seção 2.1).
const (
	TaxaPagaPorComprador   = "comprador"
	TaxaPagaPorOrganizador = "organizador"
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
	ID                      int64 `gorm:"primaryKey"`
	OrganizadorID           int64
	LocalID                 *int64
	Titulo                  string
	Slug                    string
	Descricao               string
	Categoria               string
	CapaURL                 string
	InicioEm                *time.Time
	FimEm                   *time.Time
	Timezone                string
	ClassificacaoEtaria     string
	Visibilidade            Visibilidade
	TipoAcesso              TipoAcesso
	Status                  StatusEvento
	ModoParticipantes       ModoParticipantes
	InscricaoTalentosInicio *time.Time
	InscricaoTalentosFim    *time.Time
	CapacidadeTotal         *int
	GarantiaHabilitada      bool
	TaxaPagaPor             string // comprador | organizador (item 5.1)
	AprovacaoManual         bool
	QRRotativo              bool
	CorTema                 string
	// Item 7.5: pixels de anúncio do organizador (só os IDs).
	PixelMeta                 string
	PixelGA4                  string `gorm:"column:pixel_ga4"`
	PixelTikTok               string `gorm:"column:pixel_tiktok"`
	ResultadoLiberadoEm       *time.Time
	PoliticaCancelamentoTexto string
	MaxItensPorPedido         int
	MaxIngressosPorConta      int // 0 = sem limite (item 5.3)
	PublicadoEm               *time.Time
	CanceladoEm               *time.Time
	MotivoCancelamento        string
	CriadoEm                  time.Time
}

func (Evento) TableName() string {
	return "eventos"
}
