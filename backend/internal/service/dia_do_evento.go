package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Item 9.6: painel ao vivo do dia do evento, ingressos importados de
// outras plataformas e credenciais impressas.

var (
	ErrDadosDiaEvento   = errors.New("dados inválidos")
	ErrCredencialEvento = errors.New("credencial inválida ou desativada")
)

const (
	OrigemEntradaIngresso   = "ingresso"
	OrigemEntradaImportado  = "importado"
	OrigemEntradaCredencial = "credencial"

	maxLinhasImportacao = 20000
	maxPortoes          = 20
	prefixoQRCredEvento = "EVVEC"
)

var bomUTF8 = string(rune(0xFEFF))

type PortaoEvento struct {
	ID       int64 `gorm:"primaryKey"`
	EventoID int64
	Nome     string
}

func (PortaoEvento) TableName() string { return "portoes_evento" }

type IngressoImportado struct {
	ID          int64 `gorm:"primaryKey"`
	EventoID    int64
	Codigo      string
	Nome        string
	Email       string
	Tipo        string
	Origem      string
	ImportadoEm time.Time `gorm:"default:now()"`
	UtilizadoEm *time.Time
}

func (IngressoImportado) TableName() string { return "ingressos_importados" }

type CredencialEvento struct {
	ID                int64 `gorm:"primaryKey"`
	EventoID          int64
	Nome              string
	Funcao            string
	PapelID           *int64
	Ativa             bool
	PrimeiraEntradaEm *time.Time
	UltimaEntradaEm   *time.Time
	CriadoEm          time.Time `gorm:"default:now()"`
}

func (CredencialEvento) TableName() string { return "credenciais_evento" }

// --- QR da credencial do evento (assinado, nada guardado) ---

func tokenCredencialEvento(segredo string, id int64) string {
	m := hmac.New(sha256.New, []byte(segredo))
	fmt.Fprintf(m, "credencial-evento:%d", id)
	return hex.EncodeToString(m.Sum(nil))[:24]
}

// PayloadCredencialEvento: EVVEC:<id>:<assinatura>.
func PayloadCredencialEvento(segredo string, id int64) string {
	return prefixoQRCredEvento + ":" + strconv.FormatInt(id, 10) + ":" + tokenCredencialEvento(segredo, id)
}

func LerPayloadCredencialEvento(segredo, payload string) (int64, error) {
	partes := strings.Split(strings.TrimSpace(payload), ":")
	if len(partes) != 3 || partes[0] != prefixoQRCredEvento {
		return 0, ErrCredencialEvento
	}
	id, err := strconv.ParseInt(partes[1], 10, 64)
	if err != nil || !hmac.Equal([]byte(partes[2]), []byte(tokenCredencialEvento(segredo, id))) {
		return 0, ErrCredencialEvento
	}
	return id, nil
}

// ---------- check-in (equipe ou organizador) ----------

// DefinirDiaDoEvento liga o registro de entradas e o painel (item 9.6).
func (s *CheckinService) DefinirDiaDoEvento(db *gorm.DB) { s.db = db }

// RegistrarEntrada grava uma entrada liberada para o painel ao vivo.
// Falha aqui não barra a entrada (o painel é informativo).
func (s *CheckinService) RegistrarEntrada(eventoID int64, origem string, refID int64, portao string, usuarioID int64, quando time.Time) {
	if s.db == nil {
		return
	}
	var uid *int64
	if usuarioID > 0 {
		uid = &usuarioID
	}
	s.db.Exec(`INSERT INTO entradas_evento (evento_id, origem, ref_id, portao, usuario_id, criado_em) VALUES (?, ?, ?, ?, ?, ?)`,
		eventoID, origem, refID, limitarTexto(strings.TrimSpace(portao), 40), uid, quando)
}

func limitarTexto(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func (s *CheckinService) Portoes(usuarioID, eventoID int64) ([]string, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	return listarPortoes(s.db, eventoID)
}

func listarPortoes(db *gorm.DB, eventoID int64) ([]string, error) {
	nomes := []string{}
	err := db.Model(&PortaoEvento{}).Where("evento_id = ?", eventoID).Order("id").Pluck("nome", &nomes).Error
	return nomes, err
}

type ResultadoImportado struct {
	Resultado   ResultadoValidacao
	Nome        string
	Tipo        string
	Origem      string
	UtilizadoEm *time.Time
}

// ValidarImportado procura o código (o texto lido do QR de outra
// plataforma) entre os importados do evento e marca a entrada.
func (s *CheckinService) ValidarImportado(usuarioID, eventoID int64, codigo, portao string) (*ResultadoImportado, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	codigo = strings.TrimSpace(codigo)
	var imp IngressoImportado
	if codigo == "" || s.db.Where("evento_id = ? AND codigo = ?", eventoID, codigo).First(&imp).Error != nil {
		return &ResultadoImportado{Resultado: ResultadoNaoEncontrado}, nil
	}
	r := &ResultadoImportado{Nome: imp.Nome, Tipo: imp.Tipo, Origem: imp.Origem}
	agora := time.Now()
	res := s.db.Model(&IngressoImportado{}).Where("id = ? AND utilizado_em IS NULL", imp.ID).Update("utilizado_em", agora)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		s.db.First(&imp, imp.ID)
		r.Resultado, r.UtilizadoEm = ResultadoJaUtilizado, imp.UtilizadoEm
		return r, nil
	}
	r.Resultado, r.UtilizadoEm = ResultadoValido, &agora
	s.RegistrarEntrada(eventoID, OrigemEntradaImportado, imp.ID, portao, usuarioID, agora)
	return r, nil
}

// RotuloImportado: "Importado de Sympla · Pista".
func RotuloImportado(origem, tipo string) string {
	r := "Importado"
	if origem != "" {
		r += " de " + origem
	}
	if tipo != "" {
		r += " · " + tipo
	}
	return r
}

// BuscarImportados entra na busca manual da portaria.
func (s *CheckinService) BuscarImportados(usuarioID, eventoID int64, texto string) ([]IngressoImportado, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	texto = strings.TrimSpace(texto)
	lista := []IngressoImportado{}
	if len([]rune(texto)) < 2 || s.db == nil {
		return lista, nil
	}
	like := "%" + strings.ToLower(texto) + "%"
	err := s.db.Where("evento_id = ? AND (lower(nome) LIKE ? OR lower(email) LIKE ? OR lower(codigo) LIKE ?)", eventoID, like, like, like).
		Order("nome").Limit(20).Find(&lista).Error
	return lista, err
}

type ResultadoCredencialEvento struct {
	Nome        string
	Funcao      string
	PrimeiraVez bool
}

// LerCredencialEvento: a credencial entra e sai à vontade (registra a
// primeira entrada e avisa a reentrada).
func (s *CheckinService) LerCredencialEvento(usuarioID, eventoID int64, payload, portao string) (*ResultadoCredencialEvento, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	id, err := LerPayloadCredencialEvento(s.qrSecret, payload)
	if err != nil {
		return nil, err
	}
	var c CredencialEvento
	if s.db.Where("id = ? AND evento_id = ? AND ativa", id, eventoID).First(&c).Error != nil {
		return nil, ErrCredencialEvento
	}
	agora := time.Now()
	primeira := c.PrimeiraEntradaEm == nil
	upd := map[string]any{"ultima_entrada_em": agora}
	if primeira {
		upd["primeira_entrada_em"] = agora
	}
	if err := s.db.Model(&CredencialEvento{}).Where("id = ?", c.ID).Updates(upd).Error; err != nil {
		return nil, err
	}
	if primeira {
		s.RegistrarEntrada(eventoID, OrigemEntradaCredencial, c.ID, portao, usuarioID, agora)
	}
	return &ResultadoCredencialEvento{Nome: c.Nome, Funcao: c.Funcao, PrimeiraVez: primeira}, nil
}

// ---------- painel ao vivo ----------

type MinutoEntradas struct {
	Minuto   time.Time `json:"minuto"`
	Entradas int64     `json:"entradas"`
}

type PortaoEntradas struct {
	Portao    string `json:"portao"`
	Entradas  int64  `json:"entradas"`
	Ultimos10 int64  `json:"ultimos_10"`
}

type PainelAoVivo struct {
	Presentes            int64            `json:"presentes"`
	Esperados            int64            `json:"esperados"`
	Faltam               int64            `json:"faltam"`
	ImportadosTotal      int64            `json:"importados_total"`
	ImportadosEntraram   int64            `json:"importados_entraram"`
	CredenciaisTotal     int64            `json:"credenciais_total"`
	CredenciaisEntraram  int64            `json:"credenciais_entraram"`
	PorMinuto            []MinutoEntradas `json:"por_minuto"`
	RitmoPorMinuto       float64          `json:"ritmo_por_minuto"`
	PrevisaoMinutos      *int             `json:"previsao_minutos"`
	PorPortao            []PortaoEntradas `json:"por_portao"`
	PortaPedidos         int64            `json:"porta_pedidos"`
	PortaIngressos       int64            `json:"porta_ingressos"`
	PortaDinheiroCentavo int64            `json:"porta_dinheiro_centavos"`
	PortaPixCentavos     int64            `json:"porta_pix_centavos"`
	AtualizadoEm         time.Time        `json:"atualizado_em"`
}

func (s *CheckinService) AoVivo(usuarioID, eventoID int64) (*PainelAoVivo, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	agora := time.Now()
	p := &PainelAoVivo{AtualizadoEm: agora, PorMinuto: []MinutoEntradas{}, PorPortao: []PortaoEntradas{}}

	resumo, err := s.Resumo(usuarioID, eventoID) // já soma os importados
	if err != nil {
		return nil, err
	}
	p.Presentes, p.Esperados = resumo.TotalUtilizados, resumo.TotalPagos
	if p.Faltam = p.Esperados - p.Presentes; p.Faltam < 0 {
		p.Faltam = 0
	}
	s.db.Model(&IngressoImportado{}).Where("evento_id = ?", eventoID).Count(&p.ImportadosTotal)
	s.db.Model(&IngressoImportado{}).Where("evento_id = ? AND utilizado_em IS NOT NULL", eventoID).Count(&p.ImportadosEntraram)

	// Credenciais: do evento (equipe, jurados, imprensa) e de expositores.
	var cred struct{ Total, Entraram int64 }
	s.db.Raw(`SELECT
		(SELECT count(*) FROM credenciais_evento WHERE evento_id = ? AND ativa) +
		(SELECT count(*) FROM credenciais_expositor c JOIN expositores x ON x.id = c.expositor_id WHERE x.evento_id = ? AND c.ativa) AS total,
		(SELECT count(*) FROM credenciais_evento WHERE evento_id = ? AND ativa AND primeira_entrada_em IS NOT NULL) +
		(SELECT count(*) FROM credenciais_expositor c JOIN expositores x ON x.id = c.expositor_id WHERE x.evento_id = ? AND c.ativa AND c.primeira_entrada_em IS NOT NULL) AS entraram`,
		eventoID, eventoID, eventoID, eventoID).Scan(&cred)
	p.CredenciaisTotal, p.CredenciaisEntraram = cred.Total, cred.Entraram

	// Entradas por minuto na última hora (minutos sem entrada = 0).
	inicio := agora.Truncate(time.Minute).Add(-59 * time.Minute)
	var linhas []struct {
		Minuto   time.Time
		Entradas int64
	}
	s.db.Raw(`SELECT date_trunc('minute', criado_em) AS minuto, count(*) AS entradas FROM entradas_evento
		WHERE evento_id = ? AND criado_em >= ? GROUP BY 1`, eventoID, inicio).Scan(&linhas)
	porMinuto := map[int64]int64{}
	for _, l := range linhas {
		porMinuto[l.Minuto.Unix()] = l.Entradas
	}
	var ultimos10 int64
	for i := 0; i < 60; i++ {
		m := inicio.Add(time.Duration(i) * time.Minute)
		n := porMinuto[m.Unix()]
		p.PorMinuto = append(p.PorMinuto, MinutoEntradas{Minuto: m, Entradas: n})
		if i >= 50 {
			ultimos10 += n
		}
	}
	p.RitmoPorMinuto = float64(ultimos10) / 10
	if p.RitmoPorMinuto > 0 && p.Faltam > 0 {
		min := int(float64(p.Faltam)/p.RitmoPorMinuto + 0.5)
		p.PrevisaoMinutos = &min
	}

	s.db.Raw(`SELECT portao, count(*) AS entradas, count(*) FILTER (WHERE criado_em >= ?) AS ultimos10
		FROM entradas_evento WHERE evento_id = ? GROUP BY portao ORDER BY entradas DESC`, agora.Add(-10*time.Minute), eventoID).Scan(&p.PorPortao)

	var porta []struct {
		Metodo    string
		Pedidos   int64
		Ingressos int64
		Total     int64
	}
	s.db.Raw(`SELECT p.metodo_porta AS metodo, count(DISTINCT p.id) AS pedidos, count(i.id) AS ingressos, COALESCE(sum(i.preco_centavos), 0) AS total
		FROM pedidos p JOIN itens_pedido i ON i.pedido_id = p.id
		WHERE p.evento_id = ? AND p.canal = 'porta' AND p.status = ? GROUP BY p.metodo_porta`, eventoID, domain.StatusPedidoPago).Scan(&porta)
	for _, l := range porta {
		p.PortaPedidos += l.Pedidos
		p.PortaIngressos += l.Ingressos
		if l.Metodo == MetodoPortaDinheiro {
			p.PortaDinheiroCentavo += l.Total
		} else {
			p.PortaPixCentavos += l.Total
		}
	}
	return p, nil
}

// ---------- organizador ----------

type DiaDoEventoService struct {
	db      *gorm.DB
	eventos *repository.EventoRepository
	segredo string
}

func NovoDiaDoEventoService(db *gorm.DB, eventos *repository.EventoRepository, segredo string) *DiaDoEventoService {
	return &DiaDoEventoService{db: db, eventos: eventos, segredo: segredo}
}

func (s *DiaDoEventoService) doOrganizador(organizadorID, eventoID int64) (*domain.Evento, error) {
	e, err := s.eventos.BuscarPorID(eventoID)
	if err != nil || e.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}
	return e, nil
}

func (s *DiaDoEventoService) Portoes(organizadorID, eventoID int64) ([]string, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return listarPortoes(s.db, eventoID)
}

// DefinirPortoes troca a lista inteira (as entradas antigas guardam o nome).
func (s *DiaDoEventoService) DefinirPortoes(organizadorID, eventoID int64, nomes []string) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	vistos := map[string]bool{}
	limpos := []string{}
	for _, n := range nomes {
		n = strings.TrimSpace(n)
		if n == "" || vistos[strings.ToLower(n)] {
			continue
		}
		if utf8.RuneCountInString(n) > 40 {
			return fmt.Errorf("%w: nome de portão com até 40 letras", ErrDadosDiaEvento)
		}
		vistos[strings.ToLower(n)] = true
		limpos = append(limpos, n)
	}
	if len(limpos) > maxPortoes {
		return fmt.Errorf("%w: até %d portões", ErrDadosDiaEvento, maxPortoes)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("evento_id = ?", eventoID).Delete(&PortaoEvento{}).Error; err != nil {
			return err
		}
		for _, n := range limpos {
			if err := tx.Create(&PortaoEvento{EventoID: eventoID, Nome: n}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// --- importação ---

type ResultadoImportacao struct {
	Importados int `json:"importados"`
	Repetidos  int `json:"repetidos"`
	SemCodigo  int `json:"sem_codigo"`
}

type colunasCSV struct{ codigo, nome, email, tipo int }

func normalizarCabecalho(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(s, bomUTF8)))
	r := strings.NewReplacer("á", "a", "à", "a", "ã", "a", "â", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "õ", "o", "ô", "o", "ú", "u", "ç", "c", "_", " ", "-", " ")
	return r.Replace(s)
}

// detectarColunas reconhece os cabeçalhos mais comuns (Sympla, Even3,
// Eventbrite, planilhas). Sem cabeçalho reconhecido: a 1ª coluna é o código.
func detectarColunas(cab []string) (colunasCSV, bool) {
	c := colunasCSV{-1, -1, -1, -1}
	for i, bruto := range cab {
		h := normalizarCabecalho(bruto)
		switch {
		case c.codigo < 0 && (strings.Contains(h, "codigo") || strings.Contains(h, "code") || strings.Contains(h, "qr") ||
			strings.Contains(h, "localizador") || h == "ticket id" || h == "ticket" || strings.Contains(h, "numero do ingresso") || h == "id do ingresso"):
			c.codigo = i
		case c.email < 0 && strings.Contains(h, "mail"):
			c.email = i
		case c.nome < 0 && (strings.Contains(h, "nome") || strings.Contains(h, "name") || strings.Contains(h, "participante") || strings.Contains(h, "titular")) &&
			!strings.Contains(h, "evento") && !strings.Contains(h, "ingresso"):
			c.nome = i
		case c.tipo < 0 && (strings.Contains(h, "tipo") || strings.Contains(h, "ingresso") || strings.Contains(h, "categoria") ||
			strings.Contains(h, "lote") || strings.Contains(h, "ticket type") || strings.Contains(h, "setor")):
			c.tipo = i
		}
	}
	if c.codigo < 0 {
		return colunasCSV{0, -1, -1, -1}, false
	}
	return c, true
}

func separadorCSV(primeira string) rune {
	melhor, n := ',', strings.Count(primeira, ",")
	for _, r := range []rune{';', '\t'} {
		if k := strings.Count(primeira, string(r)); k > n {
			melhor, n = r, k
		}
	}
	return melhor
}

func celula(linha []string, i, max int) string {
	if i < 0 || i >= len(linha) {
		return ""
	}
	return limitarTexto(strings.TrimSpace(linha[i]), max)
}

// Importar lê o CSV de outra plataforma. Código repetido (no arquivo ou já
// importado) é ignorado, então dá para mandar o arquivo de novo atualizado.
func (s *DiaDoEventoService) Importar(organizadorID, eventoID int64, conteudo, origem string) (*ResultadoImportacao, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	conteudo = strings.TrimPrefix(conteudo, bomUTF8)
	if strings.TrimSpace(conteudo) == "" {
		return nil, fmt.Errorf("%w: o arquivo está vazio", ErrDadosDiaEvento)
	}
	primeira := conteudo
	if i := strings.IndexAny(conteudo, "\r\n"); i >= 0 {
		primeira = conteudo[:i]
	}
	leitor := csv.NewReader(strings.NewReader(conteudo))
	leitor.Comma = separadorCSV(primeira)
	leitor.FieldsPerRecord = -1
	leitor.LazyQuotes = true
	leitor.TrimLeadingSpace = true
	origem = limitarTexto(strings.TrimSpace(origem), 60)

	var cols colunasCSV
	var linhas []IngressoImportado
	vistos := map[string]bool{}
	r := &ResultadoImportacao{}
	for n := 0; ; n++ {
		linha, err := leitor.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: não consegui ler a linha %d do arquivo", ErrDadosDiaEvento, n+1)
		}
		if n == 0 {
			var temCabecalho bool
			if cols, temCabecalho = detectarColunas(linha); temCabecalho {
				continue
			}
		}
		codigo := celula(linha, cols.codigo, 200)
		if codigo == "" {
			r.SemCodigo++
			continue
		}
		if vistos[codigo] {
			r.Repetidos++
			continue
		}
		vistos[codigo] = true
		if len(linhas) >= maxLinhasImportacao {
			return nil, fmt.Errorf("%w: até %d ingressos por arquivo", ErrDadosDiaEvento, maxLinhasImportacao)
		}
		linhas = append(linhas, IngressoImportado{EventoID: eventoID, Codigo: codigo, Nome: celula(linha, cols.nome, 160),
			Email: strings.ToLower(celula(linha, cols.email, 200)), Tipo: celula(linha, cols.tipo, 120), Origem: origem})
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(linhas); i += 500 {
			fim := min(i+500, len(linhas))
			res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(linhas[i:fim])
			if res.Error != nil {
				return res.Error
			}
			r.Importados += int(res.RowsAffected)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	r.Repetidos += len(linhas) - r.Importados
	return r, nil
}

type ResumoImportados struct {
	Total    int64            `json:"total"`
	Entraram int64            `json:"entraram"`
	Origens  []map[string]any `json:"origens"`
}

func (s *DiaDoEventoService) ResumoImportados(organizadorID, eventoID int64) (*ResumoImportados, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	r := &ResumoImportados{Origens: []map[string]any{}}
	var linhas []struct {
		Origem          string
		Total, Entraram int64
	}
	s.db.Raw(`SELECT origem, count(*) AS total, count(utilizado_em) AS entraram FROM ingressos_importados
		WHERE evento_id = ? GROUP BY origem ORDER BY origem`, eventoID).Scan(&linhas)
	for _, l := range linhas {
		r.Total += l.Total
		r.Entraram += l.Entraram
		r.Origens = append(r.Origens, map[string]any{"origem": l.Origem, "total": l.Total, "entraram": l.Entraram})
	}
	return r, nil
}

// ApagarImportados apaga os importados (todos ou só os que não entraram).
func (s *DiaDoEventoService) ApagarImportados(organizadorID, eventoID int64, todos bool) (int64, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return 0, err
	}
	q := s.db.Where("evento_id = ?", eventoID)
	if !todos {
		q = q.Where("utilizado_em IS NULL")
	}
	res := q.Delete(&IngressoImportado{})
	return res.RowsAffected, res.Error
}

// --- credenciais ---

type CredencialImpressa struct {
	ID       int64      `json:"id"`
	Nome     string     `json:"nome"`
	Funcao   string     `json:"funcao"` // equipe, jurado, imprensa, convidado, expositor
	Detalhe  string     `json:"detalhe,omitempty"`
	QR       string     `json:"qr"`
	Manual   bool       `json:"manual"`
	EntrouEm *time.Time `json:"entrou_em"`
}

// sincronizarPapeis cria a credencial de quem é equipe ou jurado
// confirmado e desliga a de quem saiu.
func (s *DiaDoEventoService) sincronizarPapeis(eventoID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO credenciais_evento (evento_id, nome, funcao, papel_id)
			SELECT p.evento_id, left(u.nome, 120), CASE WHEN p.papel = 'staff' THEN 'equipe' ELSE 'jurado' END, p.id
			FROM papeis_evento p JOIN usuarios u ON u.id = p.usuario_id
			WHERE p.evento_id = ? AND p.papel IN ('staff', 'jurado') AND p.status = 'confirmado'
			ON CONFLICT (papel_id) DO UPDATE SET ativa = true`, eventoID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE credenciais_evento c SET ativa = false
			WHERE c.evento_id = ? AND c.funcao IN ('equipe', 'jurado') AND c.ativa AND NOT EXISTS (
				SELECT 1 FROM papeis_evento p WHERE p.id = c.papel_id AND p.status = 'confirmado')`, eventoID).Error
	})
}

func (s *DiaDoEventoService) Credenciais(organizadorID, eventoID int64) ([]CredencialImpressa, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if err := s.sincronizarPapeis(eventoID); err != nil {
		return nil, err
	}
	var doEvento []CredencialEvento
	if err := s.db.Where("evento_id = ? AND ativa", eventoID).Order("funcao, nome, id").Find(&doEvento).Error; err != nil {
		return nil, err
	}
	lista := make([]CredencialImpressa, 0, len(doEvento))
	for _, c := range doEvento {
		lista = append(lista, CredencialImpressa{ID: c.ID, Nome: c.Nome, Funcao: c.Funcao, QR: PayloadCredencialEvento(s.segredo, c.ID),
			Manual: c.PapelID == nil, EntrouEm: c.PrimeiraEntradaEm})
	}
	// Expositores confirmados: a mesma credencial EVVEX do item 8.6.
	var exps []struct {
		ID                int64
		Nome              string
		Marca             string
		Mesa              string
		PrimeiraEntradaEm *time.Time
	}
	s.db.Raw(`SELECT c.id, c.nome, x.marca, COALESCE(e.codigo, '') AS mesa, c.primeira_entrada_em
		FROM credenciais_expositor c JOIN expositores x ON x.id = c.expositor_id LEFT JOIN espacos e ON e.id = x.espaco_id
		WHERE x.evento_id = ? AND c.ativa AND x.status = 'aprovado' ORDER BY x.marca, c.id`, eventoID).Scan(&exps)
	for _, x := range exps {
		detalhe := x.Marca
		if x.Mesa != "" {
			detalhe += " · mesa " + x.Mesa
		}
		lista = append(lista, CredencialImpressa{ID: x.ID, Nome: x.Nome, Funcao: "expositor", Detalhe: detalhe,
			QR: PayloadCredencial(s.segredo, x.ID), EntrouEm: x.PrimeiraEntradaEm})
	}
	return lista, nil
}

// CriarCredencial: imprensa e convidados, cadastrados à mão.
func (s *DiaDoEventoService) CriarCredencial(organizadorID, eventoID int64, nome, funcao string) (*CredencialEvento, error) {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	nome = strings.TrimSpace(nome)
	if nome == "" || utf8.RuneCountInString(nome) > 120 || (funcao != "imprensa" && funcao != "convidado") {
		return nil, fmt.Errorf("%w: informe o nome e a função (imprensa ou convidado)", ErrDadosDiaEvento)
	}
	var n int64
	s.db.Model(&CredencialEvento{}).Where("evento_id = ? AND papel_id IS NULL AND ativa", eventoID).Count(&n)
	if n >= 2000 {
		return nil, fmt.Errorf("%w: até 2.000 credenciais avulsas por evento", ErrDadosDiaEvento)
	}
	c := &CredencialEvento{EventoID: eventoID, Nome: nome, Funcao: funcao, Ativa: true}
	return c, s.db.Create(c).Error
}

// DesativarCredencial: avulsa some; a de equipe/jurado volta enquanto o
// papel existir (tire a pessoa da equipe para desligar de vez).
func (s *DiaDoEventoService) DesativarCredencial(organizadorID, eventoID, credID int64) error {
	if _, err := s.doOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	res := s.db.Model(&CredencialEvento{}).Where("id = ? AND evento_id = ? AND papel_id IS NULL", credID, eventoID).Update("ativa", false)
	if res.Error == nil && res.RowsAffected == 0 {
		return fmt.Errorf("%w: credencial não encontrada (a da equipe e a dos jurados saem quando a pessoa sai do evento)", ErrDadosDiaEvento)
	}
	return res.Error
}
