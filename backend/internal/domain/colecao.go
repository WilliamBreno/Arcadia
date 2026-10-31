package domain

import "time"

// Colecao é uma lista de eventos (item 6.2), que no item 11.7 virou também
// tema da época e coleção automática.
type Colecao struct {
	ID        int64 `gorm:"primaryKey"`
	Slug      string
	Titulo    string
	Descricao string
	Ordem     int
	Ativa     bool
	CriadoEm  time.Time
	// Item 11.7.
	Icone             string
	ExibirDe          *time.Time // data (sem hora), no fuso de Brasília
	ExibirAte         *time.Time
	Regra             string // manual | gratuitos | esta_semana | online | categoria
	RegraValor        string // categoria, na regra "categoria"
	AbertaOrganizador bool   // o organizador pode marcar o próprio evento (tema)
	NaHome            bool   // aparece como trilho na página inicial
}

func (Colecao) TableName() string { return "colecoes" }

const (
	RegraColecaoManual     = "manual"
	RegraColecaoGratuitos  = "gratuitos"
	RegraColecaoEstaSemana = "esta_semana"
	RegraColecaoOnline     = "online"
	RegraColecaoCategoria  = "categoria"
)

// RegrasColecao são as regras aceitas.
var RegrasColecao = map[string]bool{
	RegraColecaoManual: true, RegraColecaoGratuitos: true, RegraColecaoEstaSemana: true,
	RegraColecaoOnline: true, RegraColecaoCategoria: true,
}

// IconesColecao são os ícones desenhados no frontend (components/icone-colecao.tsx).
var IconesColecao = map[string]bool{
	"estrela": true, "abobora": true, "arvore": true, "tacas": true, "sol": true, "mascara": true,
	"fogueira": true, "pipa": true, "controle": true, "ingresso": true, "calendario": true, "tela": true,
	"microfone": true, "nota": true, "teatro": true, "danca": true, "cosplay": true, "comida": true,
	"livro": true, "coracao": true,
}

var fusoBrasilia = func() *time.Location {
	if l, err := time.LoadLocation("America/Sao_Paulo"); err == nil {
		return l
	}
	return time.FixedZone("BRT", -3*3600)
}()

// diaBR devolve a data (ano, mês, dia) em Brasília como número comparável.
func diaBR(t time.Time) int {
	y, m, d := t.In(fusoBrasilia).Date()
	return y*10000 + int(m)*100 + d
}

// diaGuardado lê uma coluna DATE (o Postgres devolve meia-noite UTC).
func diaGuardado(t time.Time) int {
	y, m, d := t.UTC().Date()
	return y*10000 + int(m)*100 + d
}

// VisivelEm diz se a coleção aparece no momento dado: ativa e dentro do
// período (as duas pontas contam o dia inteiro, no horário de Brasília).
func (c *Colecao) VisivelEm(agora time.Time) bool {
	if !c.Ativa {
		return false
	}
	hoje := diaBR(agora)
	if c.ExibirDe != nil && hoje < diaGuardado(*c.ExibirDe) {
		return false
	}
	if c.ExibirAte != nil && hoje > diaGuardado(*c.ExibirAte) {
		return false
	}
	return true
}

// AceitaMarcacaoEm diz se o organizador ainda pode marcar o tema: aberto e
// sem ter terminado (pode marcar antes da época começar).
func (c *Colecao) AceitaMarcacaoEm(agora time.Time) bool {
	if !c.Ativa || !c.AbertaOrganizador || c.Regra != RegraColecaoManual {
		return false
	}
	return c.ExibirAte == nil || diaBR(agora) <= diaGuardado(*c.ExibirAte)
}
