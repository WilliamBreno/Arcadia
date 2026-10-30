package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Modo palco/telão e camarim (item 8.3). Decisões do dono (30/10/2026):
// telão por link secreto (trocável); foto de referência sempre e foto do
// participante só de 18+; camarim com QR próprio do participante e busca.

const (
	PalcoAguardando = "aguardando"
	PalcoPresente   = "presente"
	PalcoNoPalco    = "no_palco"
	PalcoApresentou = "apresentou"
	PalcoAusente    = "ausente"

	TelaoIntervalo    = "intervalo"
	TelaoApresentacao = "apresentacao"
	TelaoResultado    = "resultado"

	prefixoQRParticipante = "EVVEP"
)

var (
	ErrQRParticipante  = errors.New("QR de participante inválido")
	ErrQROutroEvento   = errors.New("este participante é de outro evento")
	ErrStatusPalco     = errors.New("situação inválida")
	ErrTelaoInvalido   = errors.New("link do telão inválido ou trocado")
	ErrFichaNaoNaFila  = errors.New("participante não está na fila deste evento")
	statusPalcoManuais = map[string]bool{PalcoAguardando: true, PalcoPresente: true, PalcoApresentou: true, PalcoAusente: true}
	modosTelao         = map[string]bool{TelaoIntervalo: true, TelaoApresentacao: true, TelaoResultado: true}
)

// --- QR do participante (assinado, nada guardado) ---

func tokenParticipante(segredo string, fichaID int64) string {
	m := hmac.New(sha256.New, []byte(segredo))
	fmt.Fprintf(m, "participante:%d", fichaID)
	return hex.EncodeToString(m.Sum(nil))[:24]
}

// PayloadParticipante é o conteúdo do QR: EVVEP:<ficha>:<assinatura>. Testada.
func PayloadParticipante(segredo string, fichaID int64) string {
	return prefixoQRParticipante + ":" + strconv.FormatInt(fichaID, 10) + ":" + tokenParticipante(segredo, fichaID)
}

// LerPayloadParticipante confere a assinatura e devolve a ficha. Testada.
func LerPayloadParticipante(segredo, payload string) (int64, error) {
	partes := strings.Split(strings.TrimSpace(payload), ":")
	if len(partes) != 3 || partes[0] != prefixoQRParticipante {
		return 0, ErrQRParticipante
	}
	id, err := strconv.ParseInt(partes[1], 10, 64)
	if err != nil || !hmac.Equal([]byte(partes[2]), []byte(tokenParticipante(segredo, id))) {
		return 0, ErrQRParticipante
	}
	return id, nil
}

// --- Fila ---

type ItemFila struct {
	FichaID          int64  `json:"ficha_id"`
	Ordem            *int   `json:"ordem"`
	Nome             string `json:"nome"`
	Personagem       string `json:"personagem,omitempty"`
	Obra             string `json:"obra,omitempty"`
	Tipo             string `json:"tipo"`
	Status           string `json:"status"`
	Menor            bool   `json:"menor"`
	FotoReferencia   string `json:"foto_referencia,omitempty"`
	FotoParticipante string `json:"foto_participante,omitempty"` // só 18+ (decisão do dono)
}

type linhaFicha struct {
	ID                int64
	Nome              string
	NomeArtistico     string
	DataNascimento    *time.Time
	TipoApresentacao  *string
	Dados             []byte
	FotoURL           string
	OrdemApresentacao *int
	StatusPalco       string
}

func (l *linhaFicha) item() ItemFila {
	menor := l.DataNascimento != nil && l.DataNascimento.After(time.Now().AddDate(-18, 0, 0))
	var d map[string]any
	_ = json.Unmarshal(l.Dados, &d)
	texto := func(k string) string {
		v, _ := d[k].(string)
		return limpar(v)
	}
	tipo := ""
	if l.TipoApresentacao != nil {
		tipo = *l.TipoApresentacao
	}
	it := ItemFila{FichaID: l.ID, Ordem: l.OrdemApresentacao, Tipo: tipo, Status: l.StatusPalco, Menor: menor,
		Nome:       NomePublico(LinhaRanking{Nome: l.Nome, NomeArtistico: l.NomeArtistico, Menor: menor}),
		Personagem: texto("personagem"), Obra: texto("obra_origem"), FotoReferencia: texto("foto_referencia")}
	if !menor && strings.HasPrefix(l.FotoURL, "http") {
		it.FotoParticipante = l.FotoURL
	}
	if !strings.HasPrefix(it.FotoReferencia, "http") {
		it.FotoReferencia = ""
	}
	return it
}

type PalcoService struct {
	db        *gorm.DB
	segredo   string
	avaliacao *AvaliacaoService
	concurso  *ConcursoPublicoService
	perfis    *PerfilTalentoService
}

func NovoPalcoService(db *gorm.DB, segredo string, a *AvaliacaoService, c *ConcursoPublicoService, p *PerfilTalentoService) *PalcoService {
	return &PalcoService{db: db, segredo: segredo, avaliacao: a, concurso: c, perfis: p}
}

func (s *PalcoService) fila(tx *gorm.DB, eventoID int64) ([]ItemFila, error) {
	var ls []linhaFicha
	err := tx.Table("fichas_participacao").
		Select("id, nome, nome_artistico, data_nascimento, tipo_apresentacao, dados, foto_url, ordem_apresentacao, status_palco").
		Where("evento_id = ? AND papel = ? AND status = ?", eventoID, domain.PapelParticipante, domain.StatusFichaAprovado).
		Order("ordem_apresentacao NULLS LAST, id").Scan(&ls).Error
	out := make([]ItemFila, 0, len(ls))
	for i := range ls {
		out = append(out, ls[i].item())
	}
	return out, err
}

type estadoEvento struct {
	ID                  int64
	Titulo              string
	PalcoFichaAtualID   *int64
	TelaoModo           string
	TelaoTokenHash      *string
	ResultadoLiberadoEm *time.Time
}

func (s *PalcoService) evento(tx *gorm.DB, eventoID int64, travar bool) (*estadoEvento, error) {
	var e estadoEvento
	q := tx.Table("eventos").Select("id, titulo, palco_ficha_atual_id, telao_modo, telao_token_hash, resultado_liberado_em").Where("id = ?", eventoID)
	if travar {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.Take(&e).Error
	return &e, err
}

type EstadoPalco struct {
	AtualID    *int64     `json:"atual_id"`
	Modo       string     `json:"modo"`
	TelaoAtivo bool       `json:"telao_ativo"`
	Fila       []ItemFila `json:"fila"`
}

func (s *PalcoService) Estado(eventoID int64) (*EstadoPalco, error) {
	e, err := s.evento(s.db, eventoID, false)
	if err != nil {
		return nil, err
	}
	f, err := s.fila(s.db, eventoID)
	return &EstadoPalco{AtualID: e.PalcoFichaAtualID, Modo: e.TelaoModo, TelaoAtivo: e.TelaoTokenHash != nil, Fila: f}, err
}

// --- Camarim ---

func (s *PalcoService) fichaDoEvento(tx *gorm.DB, eventoID, fichaID int64) (string, error) {
	var st struct{ StatusPalco string }
	err := tx.Table("fichas_participacao").Select("status_palco").
		Where("id = ? AND evento_id = ? AND papel = ? AND status = ?", fichaID, eventoID, domain.PapelParticipante, domain.StatusFichaAprovado).
		Take(&st).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrFichaNaoNaFila
	}
	return st.StatusPalco, err
}

func (s *PalcoService) MarcarStatus(eventoID, fichaID int64, status string) error {
	if !statusPalcoManuais[status] {
		return ErrStatusPalco
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		e, err := s.evento(tx, eventoID, true)
		if err != nil {
			return err
		}
		if _, err := s.fichaDoEvento(tx, eventoID, fichaID); err != nil {
			return err
		}
		upd := map[string]any{"status_palco": status}
		switch status {
		case PalcoPresente:
			upd["presente_em"] = time.Now()
		case PalcoApresentou:
			upd["apresentou_em"] = time.Now()
		}
		if err := tx.Table("fichas_participacao").Where("id = ?", fichaID).Updates(upd).Error; err != nil {
			return err
		}
		// Tirou do palco quem estava nele: o palco fica vazio.
		if e.PalcoFichaAtualID != nil && *e.PalcoFichaAtualID == fichaID {
			return tx.Table("eventos").Where("id = ?", eventoID).Updates(map[string]any{"palco_ficha_atual_id": nil, "palco_atualizado_em": time.Now()}).Error
		}
		return nil
	})
}

// CheckinQR: QR do participante lido no camarim. Marca presente (se ainda
// não chegou) e devolve quem é.
func (s *PalcoService) CheckinQR(eventoID int64, payload string) (*ItemFila, bool, error) {
	fichaID, err := LerPayloadParticipante(s.segredo, payload)
	if err != nil {
		return nil, false, err
	}
	var dono struct{ EventoID int64 }
	if s.db.Table("fichas_participacao").Select("evento_id").Where("id = ?", fichaID).Take(&dono).Error != nil {
		return nil, false, ErrQRParticipante
	}
	if dono.EventoID != eventoID {
		return nil, false, ErrQROutroEvento
	}
	status, err := s.fichaDoEvento(s.db, eventoID, fichaID)
	if err != nil {
		return nil, false, err
	}
	novo := status == PalcoAguardando || status == PalcoAusente
	if novo {
		if err := s.MarcarStatus(eventoID, fichaID, PalcoPresente); err != nil {
			return nil, false, err
		}
	}
	f, err := s.fila(s.db, eventoID)
	if err != nil {
		return nil, false, err
	}
	for i := range f {
		if f[i].FichaID == fichaID {
			return &f[i], novo, nil
		}
	}
	return nil, false, ErrFichaNaoNaFila
}

// --- Palco ---

// chamar põe a ficha no palco; quem estava nele passa a "apresentou".
func (s *PalcoService) chamar(tx *gorm.DB, e *estadoEvento, fichaID int64) error {
	agora := time.Now()
	if e.PalcoFichaAtualID != nil && *e.PalcoFichaAtualID != fichaID {
		if err := tx.Table("fichas_participacao").Where("id = ? AND status_palco = ?", *e.PalcoFichaAtualID, PalcoNoPalco).
			Updates(map[string]any{"status_palco": PalcoApresentou, "apresentou_em": agora}).Error; err != nil {
			return err
		}
	}
	if err := tx.Table("fichas_participacao").Where("id = ?", fichaID).Update("status_palco", PalcoNoPalco).Error; err != nil {
		return err
	}
	return tx.Table("eventos").Where("id = ?", e.ID).
		Updates(map[string]any{"palco_ficha_atual_id": fichaID, "telao_modo": TelaoApresentacao, "palco_atualizado_em": agora}).Error
}

func (s *PalcoService) Chamar(eventoID, fichaID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		e, err := s.evento(tx, eventoID, true)
		if err != nil {
			return err
		}
		if _, err := s.fichaDoEvento(tx, eventoID, fichaID); err != nil {
			return err
		}
		return s.chamar(tx, e, fichaID)
	})
}

// ProximoDaFila: o primeiro, na ordem de apresentação, que ainda não se
// apresentou nem faltou (pode estar presente ou ainda aguardando). Testada.
func ProximoDaFila(fila []ItemFila, atual *int64) *ItemFila {
	for i := range fila {
		if atual != nil && fila[i].FichaID == *atual {
			continue
		}
		if fila[i].Status == PalcoPresente || fila[i].Status == PalcoAguardando {
			return &fila[i]
		}
	}
	return nil
}

// Proximo chama o próximo da fila. Sem ninguém, encerra a apresentação atual
// e o telão volta ao intervalo. Devolve quem foi chamado (ou nil).
func (s *PalcoService) Proximo(eventoID int64) (*ItemFila, error) {
	var chamado *ItemFila
	err := s.db.Transaction(func(tx *gorm.DB) error {
		e, err := s.evento(tx, eventoID, true)
		if err != nil {
			return err
		}
		f, err := s.fila(tx, eventoID)
		if err != nil {
			return err
		}
		if p := ProximoDaFila(f, e.PalcoFichaAtualID); p != nil {
			chamado = p
			return s.chamar(tx, e, p.FichaID)
		}
		return s.encerrar(tx, e)
	})
	return chamado, err
}

func (s *PalcoService) encerrar(tx *gorm.DB, e *estadoEvento) error {
	agora := time.Now()
	if e.PalcoFichaAtualID != nil {
		if err := tx.Table("fichas_participacao").Where("id = ? AND status_palco = ?", *e.PalcoFichaAtualID, PalcoNoPalco).
			Updates(map[string]any{"status_palco": PalcoApresentou, "apresentou_em": agora}).Error; err != nil {
			return err
		}
	}
	return tx.Table("eventos").Where("id = ?", e.ID).
		Updates(map[string]any{"palco_ficha_atual_id": nil, "telao_modo": TelaoIntervalo, "palco_atualizado_em": agora}).Error
}

func (s *PalcoService) EncerrarAtual(eventoID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		e, err := s.evento(tx, eventoID, true)
		if err != nil {
			return err
		}
		return s.encerrar(tx, e)
	})
}

func (s *PalcoService) DefinirModo(eventoID int64, modo string) error {
	if !modosTelao[modo] {
		return ErrStatusPalco
	}
	return s.db.Table("eventos").Where("id = ?", eventoID).Updates(map[string]any{"telao_modo": modo, "palco_atualizado_em": time.Now()}).Error
}

// --- Telão ---

// NovoLinkTelao gera um link secreto novo (o anterior para de funcionar).
func (s *PalcoService) NovoLinkTelao(eventoID int64) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	return token, s.db.Table("eventos").Where("id = ?", eventoID).Update("telao_token_hash", hashToken(token)).Error
}

type PessoaTelao struct {
	Nome       string `json:"nome"`
	Personagem string `json:"personagem,omitempty"`
}

type PodioTelao struct {
	Posicao int    `json:"posicao"`
	Nome    string `json:"nome"`
	Votos   int    `json:"votos,omitempty"`
}

type EstadoTelao struct {
	Evento    string                  `json:"evento"`
	Modo      string                  `json:"modo"`
	Atual     *ItemFila               `json:"atual,omitempty"`
	Proximos  []PessoaTelao           `json:"proximos"`
	Resultado map[string][]PodioTelao `json:"resultado,omitempty"`
	Publico   map[string][]PodioTelao `json:"publico,omitempty"`
}

func (s *PalcoService) Telao(token string) (*EstadoTelao, error) {
	if len(strings.TrimSpace(token)) < 40 {
		return nil, ErrTelaoInvalido
	}
	var id int64
	if err := s.db.Table("eventos").Select("id").Where("telao_token_hash = ?", hashToken(token)).Take(&id).Error; err != nil {
		return nil, ErrTelaoInvalido
	}
	e, err := s.evento(s.db, id, false)
	if err != nil {
		return nil, err
	}
	f, err := s.fila(s.db, id)
	if err != nil {
		return nil, err
	}
	out := &EstadoTelao{Evento: e.Titulo, Modo: e.TelaoModo, Proximos: []PessoaTelao{}}
	for i := range f {
		if e.PalcoFichaAtualID != nil && f[i].FichaID == *e.PalcoFichaAtualID {
			atual := f[i]
			atual.Ordem = nil
			out.Atual = &atual
		}
	}
	for i := range f {
		if len(out.Proximos) == 3 {
			break
		}
		if (f[i].Status == PalcoPresente || f[i].Status == PalcoAguardando) && (e.PalcoFichaAtualID == nil || f[i].FichaID != *e.PalcoFichaAtualID) {
			out.Proximos = append(out.Proximos, PessoaTelao{Nome: f[i].Nome, Personagem: f[i].Personagem})
		}
	}
	if e.TelaoModo == TelaoResultado && e.ResultadoLiberadoEm != nil && s.avaliacao != nil {
		var ev domain.Evento
		if s.db.First(&ev, id).Error == nil {
			if r, err := s.avaliacao.ResultadoPublico(&ev); err == nil {
				out.Resultado = map[string][]PodioTelao{}
				for tipo, ls := range r {
					for _, l := range ls {
						if l.Posicao <= 3 {
							out.Resultado[tipo] = append(out.Resultado[tipo], PodioTelao{Posicao: l.Posicao, Nome: NomePublico(l)})
						}
					}
				}
			}
			if s.concurso != nil && PremioPublicoVisivel(&ev) {
				if cont, err := s.concurso.Contagem(id, nil); err == nil {
					out.Publico = map[string][]PodioTelao{}
					for tipo, ls := range cont {
						for _, l := range ls {
							if l.Posicao == 1 && l.Votos > 0 {
								out.Publico[tipo] = append(out.Publico[tipo], PodioTelao{Posicao: 1, Nome: l.Nome, Votos: l.Votos})
							}
						}
					}
				}
			}
		}
	}
	return out, nil
}

// --- QR do próprio participante ---

type MeuQRParticipante struct {
	FichaID    int64  `json:"ficha_id"`
	Tipo       string `json:"tipo"`
	Personagem string `json:"personagem,omitempty"`
	Payload    string `json:"payload"`
	Status     string `json:"status"`
}

func (s *PalcoService) MeusQRs(eventoID, usuarioID int64) ([]MeuQRParticipante, error) {
	var ls []linhaFicha
	if err := s.db.Table("fichas_participacao").
		Select("id, nome, nome_artistico, data_nascimento, tipo_apresentacao, dados, foto_url, ordem_apresentacao, status_palco").
		Where("evento_id = ? AND usuario_id = ? AND papel = ? AND status = ?", eventoID, usuarioID, domain.PapelParticipante, domain.StatusFichaAprovado).
		Order("id").Scan(&ls).Error; err != nil {
		return nil, err
	}
	out := make([]MeuQRParticipante, 0, len(ls))
	for i := range ls {
		it := ls[i].item()
		out = append(out, MeuQRParticipante{FichaID: it.FichaID, Tipo: it.Tipo, Personagem: it.Personagem, Payload: PayloadParticipante(s.segredo, it.FichaID), Status: it.Status})
	}
	return out, nil
}
