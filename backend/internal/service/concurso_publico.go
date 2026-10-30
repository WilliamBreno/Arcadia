package service

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Voto do público e devolutiva dos jurados (item 8.2). Decisões do dono
// (30/10/2026): o voto gera um "Prêmio do público" separado (a nota dos
// jurados não muda); 1 voto por ingresso que passou no check-in, em cada
// categoria, trocável enquanto a votação está aberta; o organizador abre e
// fecha; a contagem só aparece com o resultado liberado e a votação fechada;
// a devolutiva mostra média e comentários por critério, com jurado anônimo.

var (
	ErrVotacaoFechada    = errors.New("a votação do público não está aberta")
	ErrSemCheckin        = errors.New("só quem entrou no evento (check-in) pode votar")
	ErrCandidatoInvalido = errors.New("participante inválido para esta categoria")
	ErrDevolutivaOculta  = errors.New("a devolutiva ainda não foi liberada pelo organizador")
	ErrSemDevolutiva     = errors.New("você não tem apresentação avaliada neste evento")
)

func VotacaoAberta(e *domain.Evento) bool {
	return e.VotacaoAbertaEm != nil && e.VotacaoFechadaEm == nil
}

// PremioPublicoVisivel: resultado liberado e votação encerrada. Testada.
func PremioPublicoVisivel(e *domain.Evento) bool {
	return e.ResultadoLiberadoEm != nil && e.VotacaoAbertaEm != nil && e.VotacaoFechadaEm != nil
}

type ConcursoPublicoService struct {
	db *gorm.DB
}

func NovoConcursoPublicoService(db *gorm.DB) *ConcursoPublicoService {
	return &ConcursoPublicoService{db: db}
}

// --- Organizador ---

func (s *ConcursoPublicoService) DefinirVotacao(eventoID int64, abrir bool) error {
	agora := time.Now()
	if abrir {
		return s.db.Model(&domain.Evento{}).Where("id = ?", eventoID).
			Updates(map[string]any{"votacao_aberta_em": agora, "votacao_fechada_em": nil}).Error
	}
	return s.db.Model(&domain.Evento{}).Where("id = ? AND votacao_aberta_em IS NOT NULL AND votacao_fechada_em IS NULL", eventoID).
		Update("votacao_fechada_em", agora).Error
}

func (s *ConcursoPublicoService) DefinirDevolutiva(eventoID int64, liberar bool) error {
	var v any
	if liberar {
		v = time.Now()
	}
	return s.db.Model(&domain.Evento{}).Where("id = ?", eventoID).Update("devolutiva_liberada_em", v).Error
}

// --- Candidatos ---

type Candidato struct {
	FichaID    int64  `json:"ficha_id"`
	Nome       string `json:"nome"`
	Personagem string `json:"personagem,omitempty"`
	usuarioID  int64
}

func personagemDaFicha(f *domain.FichaParticipacao) string {
	var d map[string]any
	_ = json.Unmarshal(f.Dados, &d)
	v, _ := d["personagem"].(string)
	return limpar(v)
}

func (s *ConcursoPublicoService) fichasAprovadas(eventoID int64) ([]domain.FichaParticipacao, error) {
	var fs []domain.FichaParticipacao
	err := s.db.Where("evento_id = ? AND papel = ? AND status = ?", eventoID, domain.PapelParticipante, domain.StatusFichaAprovado).
		Order("ordem_apresentacao NULLS LAST, id").Find(&fs).Error
	return fs, err
}

// Candidatos por categoria, com o nome que pode aparecer em público.
func (s *ConcursoPublicoService) Candidatos(eventoID int64) (map[string][]Candidato, error) {
	fs, err := s.fichasAprovadas(eventoID)
	if err != nil {
		return nil, err
	}
	out := map[string][]Candidato{}
	for i := range fs {
		f := &fs[i]
		t := tipoDaFicha(f)
		nome := NomePublico(LinhaRanking{Nome: f.Nome, NomeArtistico: f.NomeArtistico, Menor: f.MenorDeIdade()})
		out[t] = append(out[t], Candidato{FichaID: f.ID, Nome: nome, Personagem: personagemDaFicha(f), usuarioID: f.UsuarioID})
	}
	return out, nil
}

// --- Votar ---

// ingressosQueVotam: ingressos da pessoa neste evento que passaram no check-in.
func (s *ConcursoPublicoService) ingressosQueVotam(eventoID, usuarioID int64) ([]int64, error) {
	var ids []int64
	err := s.db.Table("itens_pedido").Select("itens_pedido.id").
		Joins("JOIN pedidos ON pedidos.id = itens_pedido.pedido_id").
		Where("pedidos.evento_id = ? AND pedidos.usuario_id = ? AND itens_pedido.status = ?", eventoID, usuarioID, domain.StatusItemUtilizado).
		Order("itens_pedido.id").Pluck("itens_pedido.id", &ids).Error
	return ids, err
}

// Votar registra (ou troca) o voto de todos os ingressos usados da pessoa
// naquela categoria. Devolve quantos votos isso representa.
func (s *ConcursoPublicoService) Votar(e *domain.Evento, usuarioID int64, tipo string, fichaID int64) (int, error) {
	if !VotacaoAberta(e) {
		return 0, ErrVotacaoFechada
	}
	var f domain.FichaParticipacao
	if err := s.db.First(&f, fichaID).Error; err != nil || f.EventoID != e.ID || f.Papel != domain.PapelParticipante ||
		f.Status != domain.StatusFichaAprovado || tipoDaFicha(&f) != tipo {
		return 0, ErrCandidatoInvalido
	}
	itens, err := s.ingressosQueVotam(e.ID, usuarioID)
	if err != nil {
		return 0, err
	}
	if len(itens) == 0 {
		return 0, ErrSemCheckin
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range itens {
			if err := tx.Exec(`INSERT INTO votos_publico (item_id, evento_id, tipo_apresentacao, ficha_id) VALUES (?, ?, ?, ?)
				ON CONFLICT (item_id, tipo_apresentacao) DO UPDATE SET ficha_id = EXCLUDED.ficha_id, atualizado_em = now()`,
				item, e.ID, tipo, fichaID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return len(itens), err
}

// MeusVotos: em quem a pessoa votou em cada categoria e quantos ingressos dela votam.
func (s *ConcursoPublicoService) MeusVotos(eventoID, usuarioID int64) (map[string]int64, int, error) {
	itens, err := s.ingressosQueVotam(eventoID, usuarioID)
	if err != nil || len(itens) == 0 {
		return map[string]int64{}, 0, err
	}
	var linhas []struct {
		TipoApresentacao string
		FichaID          int64
	}
	err = s.db.Table("votos_publico").Select("DISTINCT ON (tipo_apresentacao) tipo_apresentacao, ficha_id").
		Where("item_id IN ?", itens).Order("tipo_apresentacao, atualizado_em DESC").Scan(&linhas).Error
	out := map[string]int64{}
	for _, l := range linhas {
		out[l.TipoApresentacao] = l.FichaID
	}
	return out, len(itens), err
}

// --- Contagem ---

type LinhaVoto struct {
	Posicao    int    `json:"posicao"`
	FichaID    int64  `json:"ficha_id"`
	Nome       string `json:"nome"`
	Votos      int    `json:"votos"`
	PerfilSlug string `json:"perfil_slug,omitempty"`
	usuarioID  int64
}

// OrdenarVotos: mais votos primeiro; empate divide a posição. Testada.
func OrdenarVotos(linhas []LinhaVoto) []LinhaVoto {
	sort.SliceStable(linhas, func(i, j int) bool {
		if linhas[i].Votos != linhas[j].Votos {
			return linhas[i].Votos > linhas[j].Votos
		}
		return linhas[i].Nome < linhas[j].Nome
	})
	for i := range linhas {
		if i > 0 && linhas[i].Votos == linhas[i-1].Votos {
			linhas[i].Posicao = linhas[i-1].Posicao
		} else {
			linhas[i].Posicao = i + 1
		}
	}
	return linhas
}

// Contagem por categoria (todos os candidatos, inclusive com zero votos).
func (s *ConcursoPublicoService) Contagem(eventoID int64, perfis *PerfilTalentoService) (map[string][]LinhaVoto, error) {
	cands, err := s.Candidatos(eventoID)
	if err != nil {
		return nil, err
	}
	var brutos []struct {
		FichaID int64
		Votos   int
	}
	if err := s.db.Table("votos_publico").Select("ficha_id, COUNT(*) AS votos").Where("evento_id = ?", eventoID).
		Group("ficha_id").Scan(&brutos).Error; err != nil {
		return nil, err
	}
	votos := map[int64]int{}
	for _, b := range brutos {
		votos[b.FichaID] = b.Votos
	}
	var usuarios []int64
	for _, cs := range cands {
		for _, c := range cs {
			usuarios = append(usuarios, c.usuarioID)
		}
	}
	slugs := map[int64]string{}
	if perfis != nil {
		slugs = perfis.SlugsVisiveis(usuarios)
	}
	out := map[string][]LinhaVoto{}
	for tipo, cs := range cands {
		linhas := make([]LinhaVoto, 0, len(cs))
		for _, c := range cs {
			linhas = append(linhas, LinhaVoto{FichaID: c.FichaID, Nome: c.Nome, Votos: votos[c.FichaID], PerfilSlug: slugs[c.usuarioID], usuarioID: c.usuarioID})
		}
		out[tipo] = OrdenarVotos(linhas)
	}
	return out, nil
}

// --- Devolutiva ---

type ComentarioJurado struct {
	Jurado string `json:"jurado"`
	Texto  string `json:"texto"`
}

type CriterioDevolutiva struct {
	Nome        string             `json:"nome"`
	Media       float64            `json:"media"`
	NotaMin     float64            `json:"nota_min"`
	NotaMax     float64            `json:"nota_max"`
	Jurados     int                `json:"jurados"`
	Comentarios []ComentarioJurado `json:"comentarios"`
}

type Devolutiva struct {
	FichaID    int64                `json:"ficha_id"`
	Tipo       string               `json:"tipo"`
	Personagem string               `json:"personagem,omitempty"`
	Posicao    int                  `json:"posicao,omitempty"`
	NotaFinal  float64              `json:"nota_final,omitempty"`
	Criterios  []CriterioDevolutiva `json:"criterios"`
}

// apelidosJurados: "Jurado 1", "Jurado 2"... na ordem dos ids, o mesmo
// número para o mesmo jurado em todo o evento (sem revelar quem é). Testada.
func apelidosJurados(avs []domain.Avaliacao) map[int64]string {
	vistos := map[int64]bool{}
	var ids []int64
	for _, a := range avs {
		if !vistos[a.JuradoUsuarioID] {
			vistos[a.JuradoUsuarioID] = true
			ids = append(ids, a.JuradoUsuarioID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := map[int64]string{}
	for i, id := range ids {
		out[id] = "Jurado " + strconv.Itoa(i+1)
	}
	return out
}

// MinhaDevolutiva monta a devolutiva de cada apresentação da pessoa.
func (s *ConcursoPublicoService) MinhaDevolutiva(e *domain.Evento, usuarioID int64, ranking map[string][]LinhaRanking) ([]Devolutiva, error) {
	if e.DevolutivaLiberadaEm == nil {
		return nil, ErrDevolutivaOculta
	}
	var minhas []domain.FichaParticipacao
	if err := s.db.Where("evento_id = ? AND usuario_id = ? AND papel = ? AND status = ?", e.ID, usuarioID, domain.PapelParticipante, domain.StatusFichaAprovado).
		Find(&minhas).Error; err != nil {
		return nil, err
	}
	if len(minhas) == 0 {
		return nil, ErrSemDevolutiva
	}
	var criterios []domain.CriterioAvaliacao
	if err := s.db.Where("evento_id = ?", e.ID).Order("ordem, id").Find(&criterios).Error; err != nil {
		return nil, err
	}
	var todas []domain.Avaliacao
	if err := s.db.Table("avaliacoes").Select("avaliacoes.*").Joins("JOIN fichas_participacao f ON f.id = avaliacoes.ficha_id").
		Where("f.evento_id = ? AND avaliacoes.finalizada", e.ID).Order("avaliacoes.id").Scan(&todas).Error; err != nil {
		return nil, err
	}
	apelido := apelidosJurados(todas)
	var out []Devolutiva
	for i := range minhas {
		f := &minhas[i]
		tipo := tipoDaFicha(f)
		d := Devolutiva{FichaID: f.ID, Tipo: tipo, Personagem: personagemDaFicha(f), Criterios: []CriterioDevolutiva{}}
		for _, l := range ranking[tipo] {
			if l.FichaID == f.ID {
				d.Posicao, d.NotaFinal = l.Posicao, l.NotaFinal
			}
		}
		for ci := range criterios {
			c := &criterios[ci]
			if !criterioAplica(c, tipo) {
				continue
			}
			cd := CriterioDevolutiva{Nome: c.Nome, NotaMin: c.NotaMin, NotaMax: c.NotaMax, Comentarios: []ComentarioJurado{}}
			soma := 0.0
			for _, a := range todas {
				if a.FichaID != f.ID || a.CriterioID != c.ID {
					continue
				}
				soma += a.Nota
				cd.Jurados++
				if t := limpar(a.Comentario); t != "" {
					cd.Comentarios = append(cd.Comentarios, ComentarioJurado{Jurado: apelido[a.JuradoUsuarioID], Texto: t})
				}
			}
			if cd.Jurados == 0 {
				continue
			}
			cd.Media = arredondar2(soma / float64(cd.Jurados))
			d.Criterios = append(d.Criterios, cd)
		}
		out = append(out, d)
	}
	return out, nil
}
