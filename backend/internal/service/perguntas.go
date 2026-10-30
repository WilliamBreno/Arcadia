package service

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

// Perguntas personalizadas no checkout (item 7.6). O organizador cria
// perguntas (texto curto ou escolha entre opções), diz para que pede e para
// quais tipos de ingresso valem; cada ingresso comprado responde. As
// respostas são apagadas 90 dias depois do fim do evento (decisão do dono).

const (
	PerguntaTexto       = "texto"
	PerguntaOpcao       = "opcao"
	MaxPerguntas        = 10
	MaxRespostaChars    = 300
	DiasGuardarResposta = 90
)

var (
	ErrPerguntaInvalida = errors.New("pergunta inválida")
	ErrRespostaInvalida = errors.New("resposta inválida")
	ErrMuitasPerguntas  = fmt.Errorf("no máximo %d perguntas por evento", MaxPerguntas)
)

type Pergunta struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	EventoID        int64     `json:"-"`
	Texto           string    `json:"texto"`
	Tipo            string    `json:"tipo"`
	Opcoes          []string  `gorm:"serializer:json" json:"opcoes"`
	Obrigatoria     bool      `json:"obrigatoria"`
	Finalidade      string    `json:"finalidade"`
	TipoIngressoIDs []int64   `gorm:"serializer:json;column:tipo_ingresso_ids" json:"tipo_ingresso_ids"`
	Ordem           int       `json:"ordem"`
	Ativa           bool      `json:"-"`
	CriadoEm        time.Time `json:"-"`
}

func (Pergunta) TableName() string { return "perguntas_evento" }

// vale diz se a pergunta se aplica a este tipo de ingresso.
func (p *Pergunta) vale(tipoID int64) bool {
	return len(p.TipoIngressoIDs) == 0 || slices.Contains(p.TipoIngressoIDs, tipoID)
}

func limpar(s string) string { return strings.Join(strings.Fields(s), " ") }

// NormalizarPergunta valida o que o organizador mandou. Testada.
func NormalizarPergunta(p Pergunta) (Pergunta, error) {
	p.Texto, p.Finalidade = limpar(p.Texto), limpar(p.Finalidade)
	switch {
	case utf8.RuneCountInString(p.Texto) < 3 || utf8.RuneCountInString(p.Texto) > 200:
		return p, fmt.Errorf("%w: escreva a pergunta (de 3 a 200 caracteres)", ErrPerguntaInvalida)
	case utf8.RuneCountInString(p.Finalidade) < 5 || utf8.RuneCountInString(p.Finalidade) > 200:
		return p, fmt.Errorf("%w: explique para que você pede essa informação (o comprador vê)", ErrPerguntaInvalida)
	case p.Tipo != PerguntaTexto && p.Tipo != PerguntaOpcao:
		return p, fmt.Errorf("%w: tipo deve ser texto ou opção", ErrPerguntaInvalida)
	}
	if p.Tipo == PerguntaTexto {
		p.Opcoes = []string{}
	} else {
		vistas := map[string]bool{}
		var ops []string
		for _, o := range p.Opcoes {
			o = limpar(o)
			if o == "" || vistas[strings.ToLower(o)] {
				continue
			}
			if utf8.RuneCountInString(o) > 60 {
				return p, fmt.Errorf("%w: cada opção tem no máximo 60 caracteres", ErrPerguntaInvalida)
			}
			vistas[strings.ToLower(o)] = true
			ops = append(ops, o)
		}
		if len(ops) < 2 || len(ops) > 20 {
			return p, fmt.Errorf("%w: dê de 2 a 20 opções", ErrPerguntaInvalida)
		}
		p.Opcoes = ops
	}
	if p.TipoIngressoIDs == nil {
		p.TipoIngressoIDs = []int64{}
	}
	return p, nil
}

// ValidarRespostas confere as respostas de UM ingresso do tipo informado e
// devolve só as que valem (limpas). Testada.
func ValidarRespostas(perguntas []Pergunta, tipoID int64, respostas map[int64]string) (map[int64]string, error) {
	out := map[int64]string{}
	for i := range perguntas {
		p := &perguntas[i]
		if !p.vale(tipoID) {
			continue
		}
		r := limpar(respostas[p.ID])
		if r == "" {
			if p.Obrigatoria {
				return nil, fmt.Errorf("%w: responda \"%s\"", ErrRespostaInvalida, p.Texto)
			}
			continue
		}
		if utf8.RuneCountInString(r) > MaxRespostaChars {
			return nil, fmt.Errorf("%w: \"%s\" aceita no máximo %d caracteres", ErrRespostaInvalida, p.Texto, MaxRespostaChars)
		}
		if p.Tipo == PerguntaOpcao && !slices.Contains(p.Opcoes, r) {
			return nil, fmt.Errorf("%w: escolha uma das opções em \"%s\"", ErrRespostaInvalida, p.Texto)
		}
		out[p.ID] = r
	}
	return out, nil
}

// --- Banco ---

func PerguntasAtivas(db *gorm.DB, eventoID int64) ([]Pergunta, error) {
	var ps []Pergunta
	err := db.Where("evento_id = ? AND ativa", eventoID).Order("ordem, id").Find(&ps).Error
	return ps, err
}

type PerguntasService struct{ db *gorm.DB }

func NovoPerguntasService(db *gorm.DB) *PerguntasService { return &PerguntasService{db: db} }

func (s *PerguntasService) Listar(eventoID int64) ([]Pergunta, error) {
	ps, err := PerguntasAtivas(s.db, eventoID)
	if ps == nil {
		ps = []Pergunta{}
	}
	return ps, err
}

// tiposDoEvento garante que os tipos citados são deste evento.
func (s *PerguntasService) tiposDoEvento(eventoID int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var n int64
	if err := s.db.Table("tipos_ingresso").Where("evento_id = ? AND id IN ?", eventoID, ids).Count(&n).Error; err != nil {
		return err
	}
	if int(n) != len(ids) {
		return fmt.Errorf("%w: tipo de ingresso de outro evento", ErrPerguntaInvalida)
	}
	return nil
}

func (s *PerguntasService) Salvar(eventoID int64, p Pergunta) (*Pergunta, error) {
	p, err := NormalizarPergunta(p)
	if err != nil {
		return nil, err
	}
	if err := s.tiposDoEvento(eventoID, p.TipoIngressoIDs); err != nil {
		return nil, err
	}
	if p.ID == 0 {
		var n int64
		s.db.Model(&Pergunta{}).Where("evento_id = ? AND ativa", eventoID).Count(&n)
		if n >= MaxPerguntas {
			return nil, ErrMuitasPerguntas
		}
		p.EventoID, p.Ativa, p.CriadoEm = eventoID, true, time.Now()
		if err := s.db.Create(&p).Error; err != nil {
			return nil, err
		}
		return &p, nil
	}
	res := s.db.Model(&Pergunta{ID: p.ID}).Where("evento_id = ? AND ativa", eventoID).
		Select("texto", "tipo", "opcoes", "obrigatoria", "finalidade", "tipo_ingresso_ids", "ordem").Updates(&p)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

// Remover tira a pergunta do checkout; respostas já dadas ficam até o prazo
// de apagar.
func (s *PerguntasService) Remover(eventoID, id int64) error {
	res := s.db.Model(&Pergunta{}).Where("id = ? AND evento_id = ?", id, eventoID).Update("ativa", false)
	if res.Error == nil && res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return res.Error
}

// salvarRespostas grava as respostas dos itens recém-criados (na transação
// da reserva). respostasPorUnidade[i] são as respostas do i-ésimo ingresso.
func salvarRespostas(tx *gorm.DB, perguntas []Pergunta, tipoID int64, itens []int64, respostasPorUnidade []map[int64]string) error {
	if len(perguntas) == 0 {
		return nil
	}
	for i, itemID := range itens {
		var r map[int64]string
		if i < len(respostasPorUnidade) {
			r = respostasPorUnidade[i]
		}
		validas, err := ValidarRespostas(perguntas, tipoID, r)
		if err != nil {
			return fmt.Errorf("ingresso %d: %w", i+1, err)
		}
		for pid, resp := range validas {
			if err := tx.Exec("INSERT INTO respostas_item (item_id, pergunta_id, resposta) VALUES (?, ?, ?)", itemID, pid, resp).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// Planilha: uma linha por ingresso pago (ou usado), uma coluna por pergunta.
type LinhaRespostas struct {
	Codigo, Tipo, Titular, Email, Status string
	Respostas                            map[int64]string
}

func (s *PerguntasService) Respostas(eventoID int64) ([]Pergunta, []LinhaRespostas, error) {
	var perguntas []Pergunta
	if err := s.db.Where("evento_id = ?", eventoID).Order("ordem, id").Find(&perguntas).Error; err != nil {
		return nil, nil, err
	}
	var brutas []struct {
		ItemID                               int64
		Codigo, Tipo, Titular, Email, Status string
		PerguntaID                           *int64
		Resposta                             *string
	}
	err := s.db.Raw(`
		SELECT i.id AS item_id, i.codigo, t.nome AS tipo, i.titular_nome AS titular, i.titular_email AS email, i.status,
		       r.pergunta_id, r.resposta
		FROM itens_pedido i
		JOIN tipos_ingresso t ON t.id = i.tipo_ingresso_id
		LEFT JOIN respostas_item r ON r.item_id = i.id
		WHERE t.evento_id = ? AND i.status IN ('pago', 'utilizado')
		ORDER BY i.id`, eventoID).Scan(&brutas).Error
	if err != nil {
		return nil, nil, err
	}
	var linhas []LinhaRespostas
	idx := map[int64]int{}
	for _, b := range brutas {
		k, ok := idx[b.ItemID]
		if !ok {
			linhas = append(linhas, LinhaRespostas{Codigo: b.Codigo, Tipo: b.Tipo, Titular: b.Titular, Email: b.Email, Status: b.Status, Respostas: map[int64]string{}})
			k = len(linhas) - 1
			idx[b.ItemID] = k
		}
		if b.PerguntaID != nil && b.Resposta != nil {
			linhas[k].Respostas[*b.PerguntaID] = *b.Resposta
		}
	}
	return perguntas, linhas, nil
}

// ApagarVencidas apaga as respostas de eventos que acabaram há mais de 90
// dias. Roda no job de reservas.
func (s *PerguntasService) ApagarVencidas(agora time.Time) (int64, error) {
	res := s.db.Exec(`DELETE FROM respostas_item r USING itens_pedido i, tipos_ingresso t, eventos e
		WHERE r.item_id = i.id AND i.tipo_ingresso_id = t.id AND t.evento_id = e.id
		  AND COALESCE(e.fim_em, e.inicio_em) < ?`, agora.AddDate(0, 0, -DiasGuardarResposta))
	return res.RowsAffected, res.Error
}

// EscreverRespostasCSV: uma linha por ingresso, uma coluna por pergunta.
// Células protegidas contra fórmula (celulaCSV).
func EscreverRespostasCSV(w io.Writer, perguntas []Pergunta, linhas []LinhaRespostas) error {
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cab := []string{"codigo", "ingresso", "titular", "email", "situacao"}
	for _, p := range perguntas {
		cab = append(cab, celulaCSV(p.Texto))
	}
	if err := cw.Write(cab); err != nil {
		return err
	}
	for _, l := range linhas {
		linha := []string{celulaCSV(l.Codigo), celulaCSV(l.Tipo), celulaCSV(l.Titular), celulaCSV(l.Email), l.Status}
		for _, p := range perguntas {
			linha = append(linha, celulaCSV(l.Respostas[p.ID]))
		}
		if err := cw.Write(linha); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
