package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Item 10.3: adicionais (camiseta, estacionamento, kit, fichas de bar).
// São tipos de ingresso marcados como adicional: preço, estoque, taxa,
// garantia, reembolso e repasse funcionam como no ingresso; na portaria
// o adicional é entregue (status utilizado = entregue), não dá entrada.

var (
	ErrDadosAdicional         = errors.New("dados do adicional inválidos")
	ErrAdicionalNaoEncontrado = errors.New("adicional não encontrado")
	ErrAdicionalJaEntregue    = errors.New("este adicional já foi entregue")
	ErrAdicionalNaoPago       = errors.New("este adicional não está pago")
	prefixoQRAdicional        = "EVVEA"
)

type DadosAdicional struct {
	Nome          string
	Descricao     string
	PrecoCentavos int64
	Quantidade    int
	MaxPorPedido  int
	Ativo         bool
}

type AdicionalView struct {
	ID            int64  `json:"id"`
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	PrecoCentavos int64  `json:"preco_centavos"`
	Quantidade    int    `json:"quantidade"`
	Vendidos      int64  `json:"vendidos"`
	Disponivel    int64  `json:"disponivel"`
	MaxPorPedido  int    `json:"max_por_pedido"`
	Ativo         bool   `json:"ativo"`
}

func validarAdicional(d *DadosAdicional) error {
	d.Nome, d.Descricao = strings.TrimSpace(d.Nome), strings.TrimSpace(d.Descricao)
	if d.Nome == "" || utf8.RuneCountInString(d.Nome) > 80 || utf8.RuneCountInString(d.Descricao) > 300 {
		return fmt.Errorf("%w: nome até 80 letras e descrição até 300", ErrDadosAdicional)
	}
	if d.PrecoCentavos < 0 || d.PrecoCentavos > 10_000_000 || d.Quantidade < 1 || d.Quantidade > 100000 {
		return fmt.Errorf("%w: preço de 0 a R$ 100 mil e quantidade de 1 a 100 mil", ErrDadosAdicional)
	}
	if d.MaxPorPedido < 1 || d.MaxPorPedido > 20 {
		d.MaxPorPedido = 10
	}
	return nil
}

func (s *CheckoutService) adicionalDoEvento(tx *gorm.DB, eventoID, id int64) (*domain.TipoIngresso, error) {
	var t domain.TipoIngresso
	if tx.Where("id = ? AND evento_id = ? AND adicional", id, eventoID).First(&t).Error != nil {
		return nil, ErrAdicionalNaoEncontrado
	}
	return &t, nil
}

func (s *CheckoutService) SalvarAdicional(organizadorID, eventoID, id int64, d DadosAdicional) (int64, error) {
	if err := s.eventoDoOrganizador(organizadorID, eventoID); err != nil {
		return 0, err
	}
	if err := validarAdicional(&d); err != nil {
		return 0, err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if id > 0 {
			t, err := s.adicionalDoEvento(tx, eventoID, id)
			if err != nil {
				return err
			}
			ativos, err := s.itensPedido.ContarAtivosPorTipo(tx, t.ID)
			if err != nil {
				return err
			}
			if int64(d.Quantidade) < ativos {
				return fmt.Errorf("%w: já foram vendidos %d", ErrDadosAdicional, ativos)
			}
			if ativos > 0 && d.PrecoCentavos != t.PrecoCentavos {
				return fmt.Errorf("%w: com vendas, o preço não muda (crie outro adicional)", ErrDadosAdicional)
			}
			return tx.Model(t).Updates(map[string]any{"nome": d.Nome, "descricao": d.Descricao, "preco_centavos": d.PrecoCentavos,
				"quantidade": d.Quantidade, "max_por_pedido": d.MaxPorPedido, "ativo": d.Ativo}).Error
		}
		t := domain.TipoIngresso{EventoID: eventoID, Nome: d.Nome, Descricao: d.Descricao, PrecoCentavos: d.PrecoCentavos,
			Quantidade: d.Quantidade, MinPorPedido: 0, MaxPorPedido: d.MaxPorPedido, Ordem: 1000, Ativo: d.Ativo, Adicional: true, CriadoEm: time.Now()}
		if err := tx.Create(&t).Error; err != nil {
			return err
		}
		id = t.ID
		return nil
	})
	return id, err
}

func (s *CheckoutService) ExcluirAdicional(organizadorID, eventoID, id int64) error {
	if err := s.eventoDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		t, err := s.adicionalDoEvento(tx, eventoID, id)
		if err != nil {
			return err
		}
		var n int64
		tx.Model(&domain.ItemPedido{}).Where("tipo_ingresso_id = ?", t.ID).Count(&n)
		if n > 0 {
			return fmt.Errorf("%w: o adicional já tem vendas; desative em vez de apagar", ErrDadosAdicional)
		}
		var emPacote int64
		tx.Model(&PacoteItem{}).Where("tipo_ingresso_id = ?", t.ID).Count(&emPacote)
		if emPacote > 0 {
			return fmt.Errorf("%w: o adicional está em um pacote", ErrDadosAdicional)
		}
		return tx.Delete(t).Error
	})
}

// Adicionais do evento (soAtivos para a página pública), com o que resta.
func (s *CheckoutService) Adicionais(eventoID int64, soAtivos bool) ([]AdicionalView, error) {
	var tipos []domain.TipoIngresso
	q := s.db.Where("evento_id = ? AND adicional", eventoID).Order("ordem, id")
	if soAtivos {
		q = q.Where("ativo")
	}
	if err := q.Find(&tipos).Error; err != nil {
		return nil, err
	}
	out := make([]AdicionalView, 0, len(tipos))
	for _, t := range tipos {
		ativos, _ := s.itensPedido.ContarAtivosPorTipo(s.db, t.ID)
		out = append(out, AdicionalView{ID: t.ID, Nome: t.Nome, Descricao: t.Descricao, PrecoCentavos: t.PrecoCentavos, Quantidade: t.Quantidade,
			Vendidos: ativos, Disponivel: max(int64(t.Quantidade)-ativos, 0), MaxPorPedido: t.MaxPorPedido, Ativo: t.Ativo})
	}
	return out, nil
}

// ---------- comprador ----------

type MeuAdicional struct {
	ID           int64      `json:"id"`
	Nome         string     `json:"nome"`
	EventoTitulo string     `json:"evento_titulo"`
	EventoSlug   string     `json:"evento_slug"`
	EventoInicio *time.Time `json:"evento_inicio_em"`
	Codigo       string     `json:"codigo"`
	QR           string     `json:"qr"`
	Status       string     `json:"status"`
	EntregueEm   *time.Time `json:"entregue_em"`
}

func (s *CheckoutService) MeusAdicionais(usuarioID int64) ([]MeuAdicional, error) {
	var linhas []struct {
		ID           int64
		Nome         string
		EventoTitulo string
		EventoSlug   string
		EventoInicio *time.Time
		Codigo       string
		QRToken      string
		Status       string
		UtilizadoEm  *time.Time
	}
	err := s.db.Raw(`SELECT i.id, t.nome, e.titulo AS evento_titulo, e.slug AS evento_slug, e.inicio_em AS evento_inicio,
		i.codigo, i.qr_token, i.status, i.utilizado_em
		FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id JOIN tipos_ingresso t ON t.id = i.tipo_ingresso_id JOIN eventos e ON e.id = p.evento_id
		WHERE p.usuario_id = ? AND i.adicional AND i.status IN ('pago', 'utilizado') ORDER BY e.inicio_em DESC, i.id`, usuarioID).Scan(&linhas).Error
	out := make([]MeuAdicional, 0, len(linhas))
	for _, l := range linhas {
		out = append(out, MeuAdicional{ID: l.ID, Nome: l.Nome, EventoTitulo: l.EventoTitulo, EventoSlug: l.EventoSlug, EventoInicio: l.EventoInicio,
			Codigo: l.Codigo, QR: prefixoQRAdicional + ":" + l.Codigo + ":" + l.QRToken, Status: l.Status, EntregueEm: l.UtilizadoEm})
	}
	return out, err
}

// ---------- portaria: entrega ----------

type AdicionalEntrega struct {
	ID          int64      `json:"id"`
	Nome        string     `json:"nome"`
	Titular     string     `json:"titular"`
	Codigo      string     `json:"codigo"`
	Status      string     `json:"status"`
	EntregueEm  *time.Time `json:"entregue_em"`
	PrimeiraVez bool       `json:"primeira_vez"`
}

func (s *CheckoutService) adicionalPorCodigo(eventoID int64, codigo string) (*domain.ItemPedido, string, error) {
	var item domain.ItemPedido
	if s.db.Raw(`SELECT i.* FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id
		WHERE i.codigo = ? AND i.adicional AND p.evento_id = ?`, codigo, eventoID).Scan(&item).RowsAffected == 0 {
		return nil, "", ErrAdicionalNaoEncontrado
	}
	var nome string
	s.db.Table("tipos_ingresso").Where("id = ?", item.TipoIngressoID).Pluck("nome", &nome)
	return &item, nome, nil
}

// EntregarAdicional: o QR do adicional (EVVEA:CODIGO:TOKEN) ou, na busca
// manual, só o código. Marca entregue uma vez (idempotente: avisa).
func (s *CheckoutService) EntregarAdicional(eventoID int64, texto string, comToken bool) (*AdicionalEntrega, error) {
	partes := strings.Split(strings.TrimSpace(texto), ":")
	codigo := strings.ToUpper(partes[0])
	if len(partes) == 3 && partes[0] == prefixoQRAdicional {
		codigo = partes[1]
	} else if comToken {
		return nil, ErrAdicionalNaoEncontrado
	}
	item, nome, err := s.adicionalPorCodigo(eventoID, codigo)
	if err != nil {
		return nil, err
	}
	if comToken && (len(partes) != 3 || partes[2] != item.QRToken) {
		return nil, ErrAdicionalNaoEncontrado
	}
	r := &AdicionalEntrega{ID: item.ID, Nome: nome, Titular: item.TitularNome, Codigo: item.Codigo, Status: string(item.Status)}
	switch item.Status {
	case domain.StatusItemUtilizado:
		r.EntregueEm = item.UtilizadoEm
		return r, nil
	case domain.StatusItemPago:
	default:
		return r, ErrAdicionalNaoPago
	}
	marcou, err := s.itensPedido.MarcarUtilizadoAtomico(item.ID)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	r.Status, r.EntregueEm, r.PrimeiraVez = string(domain.StatusItemUtilizado), &agora, marcou
	return r, nil
}

// BuscarAdicionais: busca manual da portaria por nome, e-mail ou código.
func (s *CheckoutService) BuscarAdicionais(eventoID int64, q string) ([]AdicionalEntrega, error) {
	q = strings.TrimSpace(q)
	lista := []AdicionalEntrega{}
	if utf8.RuneCountInString(q) < 2 {
		return lista, nil
	}
	like := "%" + strings.ToLower(q) + "%"
	var linhas []struct {
		ID          int64
		Nome        string
		Titular     string
		Codigo      string
		Status      string
		UtilizadoEm *time.Time
	}
	err := s.db.Raw(`SELECT i.id, t.nome, i.titular_nome AS titular, i.codigo, i.status, i.utilizado_em
		FROM itens_pedido i JOIN pedidos p ON p.id = i.pedido_id JOIN tipos_ingresso t ON t.id = i.tipo_ingresso_id
		WHERE p.evento_id = ? AND i.adicional AND i.status IN ('pago', 'utilizado')
		AND (lower(i.titular_nome) LIKE ? OR lower(i.titular_email) LIKE ? OR lower(i.codigo) LIKE ?)
		ORDER BY i.titular_nome LIMIT 20`, eventoID, like, like, like).Scan(&linhas).Error
	for _, l := range linhas {
		lista = append(lista, AdicionalEntrega{ID: l.ID, Nome: l.Nome, Titular: l.Titular, Codigo: l.Codigo, Status: l.Status, EntregueEm: l.UtilizadoEm})
	}
	return lista, err
}
