package service

// Item 9.2: comissão de divulgadores. Decisões do dono (30/10/2026):
//   - por divulgador, % do preço (0 a 50%) ou valor fixo por ingresso;
//   - a Evve calcula e mostra; o organizador paga por fora e registra o
//     pagamento (o repasse do organizador não muda);
//   - conta ingresso pago e não cancelado; cancelou ou reembolsou, a
//     comissão sai sozinha; só fica "a pagar" depois do fim do evento;
//   - o divulgador vê as próprias vendas por um link secreto, sem dados
//     dos compradores.
// A comissão é gravada no item na hora da compra (comissao_afiliado_centavos):
// mudar a regra vale para as próximas vendas.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

var (
	ErrRegraComissao      = errors.New("regra de comissão inválida: percentual de 0 a 50% ou valor fixo de até R$ 1.000,00")
	ErrComissaoAntesDoFim = errors.New("a comissão só pode ser paga depois que o evento termina")
	ErrComissaoAcimaSaldo = errors.New("o valor é maior que a comissão a pagar")
	ErrPainelDivulgador   = errors.New("link do divulgador inválido ou trocado")
)

// RegraComissao vem do painel do organizador.
type RegraComissao struct {
	Tipo         string
	BP           int
	FixaCentavos int64
}

func validarRegra(r RegraComissao) error {
	switch r.Tipo {
	case domain.ComissaoNenhuma:
		return nil
	case domain.ComissaoPercentual:
		if r.BP < 1 || r.BP > 5000 {
			return ErrRegraComissao
		}
	case domain.ComissaoFixo:
		if r.FixaCentavos < 1 || r.FixaCentavos > 100000 {
			return ErrRegraComissao
		}
	default:
		return ErrRegraComissao
	}
	return nil
}

// DefinirComissao muda a regra (vale para as próximas vendas).
func (s *AfiliadoService) DefinirComissao(organizadorID, eventoID, afiliadoID int64, r RegraComissao) (*domain.Afiliado, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	if err := validarRegra(r); err != nil {
		return nil, err
	}
	a, err := s.afiliados.BuscarPorID(afiliadoID)
	if err != nil || a.EventoID != eventoID {
		return nil, ErrAfiliadoNaoEncontrado
	}
	a.ComissaoTipo, a.ComissaoBP, a.ComissaoFixaCentavos = r.Tipo, 0, 0
	switch r.Tipo {
	case domain.ComissaoPercentual:
		a.ComissaoBP = r.BP
	case domain.ComissaoFixo:
		a.ComissaoFixaCentavos = r.FixaCentavos
	}
	return a, s.afiliados.Salvar(a)
}

// SaldoComissao: comissão dos ingressos válidos, o já pago e o que falta.
type SaldoComissao struct {
	Ingressos        int64
	ComissaoCentavos int64
	PagoCentavos     int64
	APagarCentavos   int64
	Liberado         bool // o evento já terminou
}

func saldoDoAfiliado(db *gorm.DB, afiliadoID int64, liberado bool) (SaldoComissao, error) {
	var s SaldoComissao
	err := db.Raw(`SELECT COUNT(ip.id) AS ingressos, COALESCE(SUM(ip.comissao_afiliado_centavos), 0) AS comissao_centavos
		FROM pedidos p JOIN itens_pedido ip ON ip.pedido_id = p.id
		WHERE p.afiliado_id = ? AND ip.status IN ('pago', 'utilizado') AND NOT ip.cortesia
		  AND ip.ficha_id IS NULL AND ip.expositor_id IS NULL`, afiliadoID).Scan(&s).Error
	if err != nil {
		return s, err
	}
	db.Raw(`SELECT COALESCE(SUM(valor_centavos), 0) FROM comissoes_pagas WHERE afiliado_id = ?`, afiliadoID).Scan(&s.PagoCentavos)
	s.APagarCentavos = s.ComissaoCentavos - s.PagoCentavos
	if s.APagarCentavos < 0 {
		// Cancelamento depois de pago: fica negativo (crédito do organizador); mostramos 0.
		s.APagarCentavos = 0
	}
	s.Liberado = liberado
	return s, nil
}

func eventoTerminou(e *domain.Evento, agora time.Time) bool {
	fim, ok := fimDoEvento(e)
	return ok && agora.After(fim)
}

// SaldoDe é usado na lista do organizador.
func (s *AfiliadoService) SaldoDe(eventoID, afiliadoID int64) (SaldoComissao, error) {
	e, err := s.eventos.eventos.BuscarPorID(eventoID)
	if err != nil {
		return SaldoComissao{}, err
	}
	return saldoDoAfiliado(s.db, afiliadoID, eventoTerminou(e, time.Now()))
}

// RegistrarPagamento: o organizador pagou o divulgador por fora. Trava a
// linha do divulgador para dois registros ao mesmo tempo não passarem do saldo.
func (s *AfiliadoService) RegistrarPagamento(organizadorID, usuarioID, eventoID, afiliadoID, valor int64, observacao string) error {
	e, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return err
	}
	if !eventoTerminou(e, time.Now()) {
		return ErrComissaoAntesDoFim
	}
	if valor <= 0 {
		return ErrComissaoAcimaSaldo
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var a domain.Afiliado
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, afiliadoID).Error; err != nil || a.EventoID != eventoID {
			return ErrAfiliadoNaoEncontrado
		}
		saldo, err := saldoDoAfiliado(tx, a.ID, true)
		if err != nil {
			return err
		}
		if valor > saldo.APagarCentavos {
			return fmt.Errorf("%w (a pagar: %s)", ErrComissaoAcimaSaldo, formatarReais(saldo.APagarCentavos))
		}
		obs := strings.TrimSpace(observacao)
		if len([]rune(obs)) > 300 {
			obs = string([]rune(obs)[:300])
		}
		return tx.Exec(`INSERT INTO comissoes_pagas (afiliado_id, valor_centavos, observacao, registrado_por, pago_em) VALUES (?, ?, ?, ?, now())`,
			a.ID, valor, obs, usuarioID).Error
	})
}

// PagamentoComissao é uma linha do histórico.
type PagamentoComissao struct {
	ValorCentavos int64
	Observacao    string
	PagoEm        time.Time
}

func (s *AfiliadoService) pagamentosDe(afiliadoID int64) []PagamentoComissao {
	var ps []PagamentoComissao
	s.db.Raw(`SELECT valor_centavos, observacao, pago_em FROM comissoes_pagas WHERE afiliado_id = ? ORDER BY pago_em`, afiliadoID).Scan(&ps)
	return ps
}

func hashPainel(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GerarLinkPainel cria (ou troca) o link secreto do divulgador. O token só
// existe na resposta; guardamos o hash.
func (s *AfiliadoService) GerarLinkPainel(organizadorID, eventoID, afiliadoID int64) (string, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return "", err
	}
	a, err := s.afiliados.BuscarPorID(afiliadoID)
	if err != nil || a.EventoID != eventoID {
		return "", ErrAfiliadoNaoEncontrado
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	h := hashPainel(token)
	a.PainelTokenHash = &h
	return token, s.afiliados.Salvar(a)
}

// PainelDivulgador é o que o divulgador vê pelo link (sem compradores).
type PainelDivulgador struct {
	Nome         string
	EventoTitulo string
	EventoSlug   string
	InicioEm     *time.Time
	Codigo       string
	Regra        RegraComissao
	Saldo        SaldoComissao
	Pagamentos   []PagamentoComissao
}

func (s *AfiliadoService) Painel(token string) (*PainelDivulgador, error) {
	token = strings.TrimSpace(token)
	if len(token) != 48 {
		return nil, ErrPainelDivulgador
	}
	var a domain.Afiliado
	if err := s.db.Where("painel_token_hash = ?", hashPainel(token)).First(&a).Error; err != nil {
		return nil, ErrPainelDivulgador
	}
	e, err := s.eventos.eventos.BuscarPorID(a.EventoID)
	if err != nil {
		return nil, ErrPainelDivulgador
	}
	saldo, err := saldoDoAfiliado(s.db, a.ID, eventoTerminou(e, time.Now()))
	if err != nil {
		return nil, err
	}
	return &PainelDivulgador{
		Nome: a.Nome, EventoTitulo: e.Titulo, EventoSlug: e.Slug, InicioEm: e.InicioEm, Codigo: a.Codigo,
		Regra: RegraComissao{Tipo: a.ComissaoTipo, BP: a.ComissaoBP, FixaCentavos: a.ComissaoFixaCentavos},
		Saldo: saldo, Pagamentos: s.pagamentosDe(a.ID),
	}, nil
}

// PagamentosDe expõe o histórico ao painel do organizador.
func (s *AfiliadoService) PagamentosDe(organizadorID, eventoID, afiliadoID int64) ([]PagamentoComissao, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	a, err := s.afiliados.BuscarPorID(afiliadoID)
	if err != nil || a.EventoID != eventoID {
		return nil, ErrAfiliadoNaoEncontrado
	}
	return s.pagamentosDe(a.ID), nil
}
