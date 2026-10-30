package service

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Item 9.5: mapa de assentos e mesas. Cada cadeira é um lugar; o setor
// aceita um ou mais tipos de ingresso, que dão o preço. O índice único
// itens_pedido_lugar_ativo impede dois ingressos ativos na mesma cadeira.

var (
	ErrMapaNaoEncontrado = errors.New("evento não encontrado")
	ErrLugarOcupado      = errors.New("um dos lugares escolhidos acabou de ser ocupado; escolha outro")
	ErrLugarInvalido     = errors.New("lugar inválido para este ingresso")
	ErrEscolhaLugares    = errors.New("escolha no mapa um lugar para cada ingresso")
	ErrMesaInteira       = errors.New("esta mesa é vendida inteira: escolha todas as cadeiras dela")
	ErrSemLugarLivre     = errors.New("não há lugar livre para este ingresso")
	ErrDadosSetor        = errors.New("dados do setor inválidos")
	ErrSetorComIngressos = errors.New("há ingressos neste setor; não dá para apagar nem mudar as cadeiras vendidas")
)

const (
	FormatoFileiras = "fileiras"
	FormatoMesas    = "mesas"

	maxLugaresEvento   = 5000
	maxLugaresPorGrupo = 60
	maxFileiras        = 52
	maxMesas           = 300
)

type SetorMapa struct {
	ID       int64 `gorm:"primaryKey"`
	EventoID int64
	Nome     string
	Formato  string
	Ordem    int
	CriadoEm time.Time `gorm:"default:now()"`
}

func (SetorMapa) TableName() string { return "setores_mapa" }

type Lugar struct {
	ID          int64 `gorm:"primaryKey"`
	SetorID     int64
	Grupo       string
	Numero      int
	OrdemGrupo  int
	MesaInteira bool
}

func (Lugar) TableName() string { return "lugares" }

// RotuloLugar é o texto gravado no ingresso e mostrado na portaria.
func RotuloLugar(setor, formato, grupo string, numero int) string {
	if formato == FormatoMesas {
		return fmt.Sprintf("%s · Mesa %s · Cadeira %d", setor, grupo, numero)
	}
	return fmt.Sprintf("%s · Fileira %s · Lugar %d", setor, grupo, numero)
}

// NomeFileira: 1 → A, 26 → Z, 27 → AA.
func NomeFileira(n int) string {
	nome := ""
	for n > 0 {
		n--
		nome = string(rune('A'+n%26)) + nome
		n /= 26
	}
	return nome
}

// ---------- leitura do mapa ----------

type LugarMapa struct {
	ID      int64 `json:"id"`
	Numero  int   `json:"numero"`
	Ocupado bool  `json:"ocupado"`
}

type GrupoMapa struct {
	Grupo       string      `json:"grupo"`
	MesaInteira bool        `json:"mesa_inteira"`
	Lugares     []LugarMapa `json:"lugares"`
}

type SetorView struct {
	ID              int64       `json:"id"`
	Nome            string      `json:"nome"`
	Formato         string      `json:"formato"`
	TipoIngressoIDs []int64     `json:"tipo_ingresso_ids"`
	Grupos          []GrupoMapa `json:"grupos"`
	Total           int         `json:"total"`
	Livres          int         `json:"livres"`
}

var statusAtivosLugar = []domain.StatusItemPedido{domain.StatusItemReservado, domain.StatusItemPago, domain.StatusItemUtilizado}

// MapaDoEvento monta os setores com as cadeiras livres e ocupadas (sem
// dizer quem ocupa).
func MapaDoEvento(db *gorm.DB, eventoID int64) ([]SetorView, error) {
	var setores []SetorMapa
	if err := db.Where("evento_id = ?", eventoID).Order("ordem, id").Find(&setores).Error; err != nil {
		return nil, err
	}
	if len(setores) == 0 {
		return []SetorView{}, nil
	}
	ids := make([]int64, len(setores))
	for i := range setores {
		ids[i] = setores[i].ID
	}
	var vinculos []struct{ SetorID, TipoIngressoID int64 }
	if err := db.Table("setor_tipos").Where("setor_id IN ?", ids).Order("tipo_ingresso_id").Scan(&vinculos).Error; err != nil {
		return nil, err
	}
	var lugares []Lugar
	if err := db.Where("setor_id IN ?", ids).Order("setor_id, ordem_grupo, grupo, numero").Find(&lugares).Error; err != nil {
		return nil, err
	}
	var ocupados []int64
	if err := db.Model(&domain.ItemPedido{}).Joins("JOIN lugares ON lugares.id = itens_pedido.lugar_id").
		Where("lugares.setor_id IN ? AND itens_pedido.status IN ?", ids, statusAtivosLugar).
		Pluck("itens_pedido.lugar_id", &ocupados).Error; err != nil {
		return nil, err
	}
	ocupado := map[int64]bool{}
	for _, id := range ocupados {
		ocupado[id] = true
	}
	views := make([]SetorView, len(setores))
	pos := map[int64]int{}
	for i, s := range setores {
		views[i] = SetorView{ID: s.ID, Nome: s.Nome, Formato: s.Formato, TipoIngressoIDs: []int64{}, Grupos: []GrupoMapa{}}
		pos[s.ID] = i
	}
	for _, v := range vinculos {
		views[pos[v.SetorID]].TipoIngressoIDs = append(views[pos[v.SetorID]].TipoIngressoIDs, v.TipoIngressoID)
	}
	for _, l := range lugares {
		v := &views[pos[l.SetorID]]
		if n := len(v.Grupos); n == 0 || v.Grupos[n-1].Grupo != l.Grupo {
			v.Grupos = append(v.Grupos, GrupoMapa{Grupo: l.Grupo, MesaInteira: l.MesaInteira})
		}
		g := &v.Grupos[len(v.Grupos)-1]
		g.Lugares = append(g.Lugares, LugarMapa{ID: l.ID, Numero: l.Numero, Ocupado: ocupado[l.ID]})
		v.Total++
		if !ocupado[l.ID] {
			v.Livres++
		}
	}
	return views, nil
}

// TiposComLugar diz quais tipos pedem cadeira marcada.
func TiposComLugar(db *gorm.DB, tipoIDs []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if len(tipoIDs) == 0 {
		return out, nil
	}
	var ids []int64
	if err := db.Table("setor_tipos").Where("tipo_ingresso_id IN ?", tipoIDs).Distinct().Pluck("tipo_ingresso_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// ---------- checkout ----------

type lugarEscolhido struct {
	Lugar
	SetorNome string
	Formato   string
	Rotulo    string
}

type linhaLugar struct {
	Lugar
	SetorNome string
	Formato   string
}

// conferirLugares valida as cadeiras pedidas: um lugar por ingresso nos
// tipos com mapa, cadeira do setor que aceita aquele tipo, sem repetir,
// livre agora e, nas mesas inteiras, todas as cadeiras da mesa juntas.
// Devolve as cadeiras por tipo, na ordem pedida.
func conferirLugares(tx *gorm.DB, eventoID int64, itens []ItemRequisitado) (map[int]([]lugarEscolhido), error) {
	tipoIDs := make([]int64, 0, len(itens))
	for _, ir := range itens {
		tipoIDs = append(tipoIDs, ir.TipoIngressoID)
	}
	comLugar, err := TiposComLugar(tx, tipoIDs)
	if err != nil {
		return nil, err
	}
	pedidos := map[int64]bool{}
	todos := []int64{}
	for _, ir := range itens {
		if !comLugar[ir.TipoIngressoID] {
			if len(ir.LugarIDs) > 0 {
				return nil, ErrLugarInvalido
			}
			continue
		}
		if len(ir.LugarIDs) != ir.Quantidade {
			return nil, ErrEscolhaLugares
		}
		for _, id := range ir.LugarIDs {
			if pedidos[id] {
				return nil, ErrLugarInvalido
			}
			pedidos[id] = true
			todos = append(todos, id)
		}
	}
	out := map[int][]lugarEscolhido{}
	if len(todos) == 0 {
		return out, nil
	}
	var linhas []linhaLugar
	if err := tx.Table("lugares").
		Select("lugares.*, setores_mapa.nome AS setor_nome, setores_mapa.formato").
		Joins("JOIN setores_mapa ON setores_mapa.id = lugares.setor_id").
		Where("lugares.id IN ? AND setores_mapa.evento_id = ?", todos, eventoID).Scan(&linhas).Error; err != nil {
		return nil, err
	}
	porID := map[int64]linhaLugar{}
	for _, l := range linhas {
		porID[l.ID] = l
	}
	var vinculos []struct{ SetorID, TipoIngressoID int64 }
	if err := tx.Table("setor_tipos").Where("tipo_ingresso_id IN ?", tipoIDs).Scan(&vinculos).Error; err != nil {
		return nil, err
	}
	aceita := map[[2]int64]bool{}
	for _, v := range vinculos {
		aceita[[2]int64{v.SetorID, v.TipoIngressoID}] = true
	}
	for idx, ir := range itens {
		for _, id := range ir.LugarIDs {
			l, ok := porID[id]
			if !ok || !aceita[[2]int64{l.SetorID, ir.TipoIngressoID}] {
				return nil, ErrLugarInvalido
			}
			out[idx] = append(out[idx], lugarEscolhido{Lugar: l.Lugar, SetorNome: l.SetorNome, Formato: l.Formato,
				Rotulo: RotuloLugar(l.SetorNome, l.Formato, l.Grupo, l.Numero)})
		}
	}
	// Mesa inteira: todas as cadeiras daquela mesa precisam estar no pedido.
	conferidas := map[string]bool{}
	for _, l := range porID {
		chave := strconv.FormatInt(l.SetorID, 10) + "|" + l.Grupo
		if !l.MesaInteira || conferidas[chave] {
			continue
		}
		conferidas[chave] = true
		var daMesa []int64
		if err := tx.Model(&Lugar{}).Where("setor_id = ? AND grupo = ?", l.SetorID, l.Grupo).Pluck("id", &daMesa).Error; err != nil {
			return nil, err
		}
		for _, id := range daMesa {
			if !pedidos[id] {
				return nil, fmt.Errorf("%w (mesa %s, %d cadeiras)", ErrMesaInteira, l.Grupo, len(daMesa))
			}
		}
	}
	var ocupados int64
	if err := tx.Model(&domain.ItemPedido{}).Where("lugar_id IN ? AND status IN ?", todos, statusAtivosLugar).Count(&ocupados).Error; err != nil {
		return nil, err
	}
	if ocupados > 0 {
		return nil, ErrLugarOcupado
	}
	return out, nil
}

// lugaresLivres escolhe sozinho as primeiras cadeiras livres do tipo (para
// a cortesia e a venda na porta). Pula as mesas vendidas inteiras.
func lugaresLivres(tx *gorm.DB, tipoID int64, n int) ([]lugarEscolhido, error) {
	var linhas []linhaLugar
	err := tx.Table("lugares").
		Select("lugares.*, setores_mapa.nome AS setor_nome, setores_mapa.formato").
		Joins("JOIN setores_mapa ON setores_mapa.id = lugares.setor_id").
		Joins("JOIN setor_tipos ON setor_tipos.setor_id = lugares.setor_id AND setor_tipos.tipo_ingresso_id = ?", tipoID).
		Where("lugares.mesa_inteira = false").
		Where("NOT EXISTS (SELECT 1 FROM itens_pedido ip WHERE ip.lugar_id = lugares.id AND ip.status IN ?)", statusAtivosLugar).
		Order("setores_mapa.ordem, setores_mapa.id, lugares.ordem_grupo, lugares.grupo, lugares.numero").
		Limit(n).Scan(&linhas).Error
	if err != nil {
		return nil, err
	}
	if len(linhas) < n {
		return nil, ErrSemLugarLivre
	}
	out := make([]lugarEscolhido, len(linhas))
	for i, l := range linhas {
		out[i] = lugarEscolhido{Lugar: l.Lugar, SetorNome: l.SetorNome, Formato: l.Formato, Rotulo: RotuloLugar(l.SetorNome, l.Formato, l.Grupo, l.Numero)}
	}
	return out, nil
}

// lugarJaOcupado reconhece a violação do índice único (corrida entre dois
// pedidos para a mesma cadeira).
func lugarJaOcupado(err error) bool {
	return err != nil && strings.Contains(err.Error(), "itens_pedido_lugar_ativo")
}

// ---------- organizador ----------

type LugaresService struct {
	db      *gorm.DB
	eventos *repository.EventoRepository
}

func NovoLugaresService(db *gorm.DB, eventos *repository.EventoRepository) *LugaresService {
	return &LugaresService{db: db, eventos: eventos}
}

func (s *LugaresService) doOrganizador(organizadorID, eventoID int64) (*domain.Evento, error) {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return e, nil
}

// Publico: mapa de um evento publicado (sem dados de quem comprou).
func (s *LugaresService) Publico(slug string) ([]SetorView, error) {
	e, err := s.eventos.BuscarPorSlug(slug)
	if err != nil || e.Status != domain.StatusEventoPublicado {
		return nil, ErrMapaNaoEncontrado
	}
	return MapaDoEvento(s.db, e.ID)
}

func (s *LugaresService) Painel(organizadorID, eventoID int64) ([]SetorView, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return MapaDoEvento(s.db, eventoID)
}

type DadosSetor struct {
	Nome              string
	Formato           string
	TipoIngressoIDs   []int64
	Fileiras          int // fileiras: quantas (A, B, C…)
	LugaresPorFileira int
	Mesas             int // mesas: quantas, numeradas a partir de PrimeiraMesa
	PrimeiraMesa      int
	CadeirasPorMesa   int
	MesaInteira       bool
}

// conferirTipos: tipos do próprio evento, de plateia (não internos).
func (s *LugaresService) conferirTipos(tx *gorm.DB, eventoID int64, ids []int64) ([]int64, error) {
	unicos := map[int64]bool{}
	lista := []int64{}
	for _, id := range ids {
		if !unicos[id] {
			unicos[id] = true
			lista = append(lista, id)
		}
	}
	if len(lista) == 0 {
		return nil, fmt.Errorf("%w: escolha pelo menos um tipo de ingresso para o setor", ErrDadosSetor)
	}
	var n int64
	if err := tx.Model(&domain.TipoIngresso{}).
		Where("id IN ? AND evento_id = ? AND inscricao_talento = false AND espaco_expositor = false", lista, eventoID).
		Count(&n).Error; err != nil {
		return nil, err
	}
	if int(n) != len(lista) {
		return nil, fmt.Errorf("%w: tipo de ingresso não encontrado neste evento", ErrDadosSetor)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i] < lista[j] })
	return lista, nil
}

func (s *LugaresService) CriarSetor(organizadorID, eventoID int64, d DadosSetor) (int64, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return 0, err
	}
	d.Nome = strings.TrimSpace(d.Nome)
	if d.Nome == "" || len([]rune(d.Nome)) > 40 {
		return 0, fmt.Errorf("%w: dê um nome de até 40 letras ao setor", ErrDadosSetor)
	}
	type nova struct {
		grupo  string
		ordem  int
		numero int
	}
	var cadeiras []nova
	switch d.Formato {
	case FormatoFileiras:
		if d.Fileiras < 1 || d.Fileiras > maxFileiras || d.LugaresPorFileira < 1 || d.LugaresPorFileira > maxLugaresPorGrupo {
			return 0, fmt.Errorf("%w: de 1 a %d fileiras, com 1 a %d lugares cada", ErrDadosSetor, maxFileiras, maxLugaresPorGrupo)
		}
		for f := 1; f <= d.Fileiras; f++ {
			for n := 1; n <= d.LugaresPorFileira; n++ {
				cadeiras = append(cadeiras, nova{NomeFileira(f), f, n})
			}
		}
		d.MesaInteira = false
	case FormatoMesas:
		if d.PrimeiraMesa < 1 {
			d.PrimeiraMesa = 1
		}
		if d.Mesas < 1 || d.Mesas > maxMesas || d.CadeirasPorMesa < 1 || d.CadeirasPorMesa > 20 {
			return 0, fmt.Errorf("%w: de 1 a %d mesas, com 1 a 20 cadeiras cada", ErrDadosSetor, maxMesas)
		}
		for m := d.PrimeiraMesa; m < d.PrimeiraMesa+d.Mesas; m++ {
			for n := 1; n <= d.CadeirasPorMesa; n++ {
				cadeiras = append(cadeiras, nova{strconv.Itoa(m), m, n})
			}
		}
	default:
		return 0, fmt.Errorf("%w: formato deve ser fileiras ou mesas", ErrDadosSetor)
	}
	var setorID int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		tipos, err := s.conferirTipos(tx, eventoID, d.TipoIngressoIDs)
		if err != nil {
			return err
		}
		// Trava o evento para o limite de cadeiras valer sob concorrência.
		if err := tx.Exec("SELECT id FROM eventos WHERE id = ? FOR UPDATE", eventoID).Error; err != nil {
			return err
		}
		var existentes int64
		if err := tx.Table("lugares").Joins("JOIN setores_mapa ON setores_mapa.id = lugares.setor_id").
			Where("setores_mapa.evento_id = ?", eventoID).Count(&existentes).Error; err != nil {
			return err
		}
		if int(existentes)+len(cadeiras) > maxLugaresEvento {
			return fmt.Errorf("%w: o mapa aceita até %d lugares por evento", ErrDadosSetor, maxLugaresEvento)
		}
		var ordem int
		tx.Model(&SetorMapa{}).Where("evento_id = ?", eventoID).Select("COALESCE(MAX(ordem), 0) + 1").Scan(&ordem)
		setor := SetorMapa{EventoID: eventoID, Nome: d.Nome, Formato: d.Formato, Ordem: ordem}
		if err := tx.Create(&setor).Error; err != nil {
			if strings.Contains(err.Error(), "setores_mapa_evento_id_nome_key") {
				return fmt.Errorf("%w: já existe um setor com esse nome", ErrDadosSetor)
			}
			return err
		}
		setorID = setor.ID
		for _, t := range tipos {
			if err := tx.Exec("INSERT INTO setor_tipos (setor_id, tipo_ingresso_id) VALUES (?, ?)", setor.ID, t).Error; err != nil {
				return err
			}
		}
		linhas := make([]Lugar, len(cadeiras))
		for i, c := range cadeiras {
			linhas[i] = Lugar{SetorID: setor.ID, Grupo: c.grupo, Numero: c.numero, OrdemGrupo: c.ordem, MesaInteira: d.MesaInteira}
		}
		return tx.CreateInBatches(linhas, 500).Error
	})
	return setorID, err
}

func (s *LugaresService) setorDoEvento(tx *gorm.DB, eventoID, setorID int64) (*SetorMapa, error) {
	var setor SetorMapa
	if err := tx.Where("id = ? AND evento_id = ?", setorID, eventoID).First(&setor).Error; err != nil {
		return nil, fmt.Errorf("%w: setor não encontrado", ErrDadosSetor)
	}
	return &setor, nil
}

// AtualizarSetor troca o nome e os tipos aceitos. Ingressos já vendidos
// guardam o rótulo e o preço da compra.
func (s *LugaresService) AtualizarSetor(organizadorID, eventoID, setorID int64, nome string, tipoIDs []int64) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	nome = strings.TrimSpace(nome)
	if nome == "" || len([]rune(nome)) > 40 {
		return fmt.Errorf("%w: dê um nome de até 40 letras ao setor", ErrDadosSetor)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.setorDoEvento(tx, eventoID, setorID); err != nil {
			return err
		}
		tipos, err := s.conferirTipos(tx, eventoID, tipoIDs)
		if err != nil {
			return err
		}
		if err := tx.Model(&SetorMapa{}).Where("id = ?", setorID).Update("nome", nome).Error; err != nil {
			if strings.Contains(err.Error(), "setores_mapa_evento_id_nome_key") {
				return fmt.Errorf("%w: já existe um setor com esse nome", ErrDadosSetor)
			}
			return err
		}
		if err := tx.Exec("DELETE FROM setor_tipos WHERE setor_id = ?", setorID).Error; err != nil {
			return err
		}
		for _, t := range tipos {
			if err := tx.Exec("INSERT INTO setor_tipos (setor_id, tipo_ingresso_id) VALUES (?, ?)", setorID, t).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ingressosAtivosNoSetor(tx *gorm.DB, setorID int64, grupo string) (int64, error) {
	q := tx.Model(&domain.ItemPedido{}).Joins("JOIN lugares ON lugares.id = itens_pedido.lugar_id").
		Where("lugares.setor_id = ? AND itens_pedido.status IN ?", setorID, statusAtivosLugar)
	if grupo != "" {
		q = q.Where("lugares.grupo = ?", grupo)
	}
	var n int64
	return n, q.Count(&n).Error
}

func (s *LugaresService) ExcluirSetor(organizadorID, eventoID, setorID int64) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.setorDoEvento(tx, eventoID, setorID); err != nil {
			return err
		}
		// Trava os tipos do setor: nenhuma compra nova entra no meio.
		if err := tx.Exec(`SELECT id FROM tipos_ingresso WHERE id IN (SELECT tipo_ingresso_id FROM setor_tipos WHERE setor_id = ?) FOR UPDATE`, setorID).Error; err != nil {
			return err
		}
		n, err := ingressosAtivosNoSetor(tx, setorID, "")
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrSetorComIngressos
		}
		return tx.Delete(&SetorMapa{}, setorID).Error
	})
}

// DefinirMesaInteira liga ou desliga a venda da mesa toda (só sem
// ingresso ativo naquela mesa).
func (s *LugaresService) DefinirMesaInteira(organizadorID, eventoID, setorID int64, grupo string, inteira bool) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		setor, err := s.setorDoEvento(tx, eventoID, setorID)
		if err != nil {
			return err
		}
		if setor.Formato != FormatoMesas {
			return fmt.Errorf("%w: só setores de mesas", ErrDadosSetor)
		}
		if err := tx.Exec(`SELECT id FROM tipos_ingresso WHERE id IN (SELECT tipo_ingresso_id FROM setor_tipos WHERE setor_id = ?) FOR UPDATE`, setorID).Error; err != nil {
			return err
		}
		n, err := ingressosAtivosNoSetor(tx, setorID, grupo)
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrSetorComIngressos
		}
		res := tx.Model(&Lugar{}).Where("setor_id = ? AND grupo = ?", setorID, grupo).Update("mesa_inteira", inteira)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("%w: mesa não encontrada", ErrDadosSetor)
		}
		return nil
	})
}

// lugaresAutomaticos: nil se o tipo não tem mapa; senão as primeiras
// cadeiras livres (cortesia e venda na porta não abrem o mapa).
func lugaresAutomaticos(tx *gorm.DB, tipoID int64, n int) ([]lugarEscolhido, error) {
	com, err := TiposComLugar(tx, []int64{tipoID})
	if err != nil || !com[tipoID] {
		return nil, err
	}
	return lugaresLivres(tx, tipoID, n)
}

// marcarLugar põe a cadeira i (se houver) no ingresso.
func marcarLugar(item *domain.ItemPedido, lugares []lugarEscolhido, i int) {
	if i < len(lugares) {
		item.LugarID = &lugares[i].ID
		item.LugarRotulo = lugares[i].Rotulo
	}
}
