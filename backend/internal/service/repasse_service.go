package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrRepasseNaoEncontrado = errors.New("repasse não encontrado")
	ErrRepasseNaoPendente   = errors.New("repasse não está pendente (já pago ou cancelado)")
)

// Financeiro é o cálculo da seção 7.6/7.7 para um evento.
type Financeiro struct {
	BrutoCentavos           int64
	TaxaProcessadorCentavos int64
	LiquidoCentavos         int64
}

// alocarTaxaProcessador rateia a taxa real do pagamento pelo peso do item
// no total pago (seção 7.7). Divisão inteira: sobras de centavos ficam
// com o organizador (registrado na seção 14).
func alocarTaxaProcessador(taxaPagamento, totalItem, valorPagamento int64) int64 {
	if valorPagamento <= 0 {
		return 0
	}
	return taxaPagamento * totalItem / valorPagamento
}

// adicionarDiasUteis soma n dias úteis (seg-sex, sem feriados).
func adicionarDiasUteis(t time.Time, n int) time.Time {
	for n > 0 {
		t = t.AddDate(0, 0, 1)
		if t.Weekday() != time.Saturday && t.Weekday() != time.Sunday {
			n--
		}
	}
	return t
}

// fimDoEvento é o fim do último dia (seção 7.6): fim_em, ou início se não
// houver fim, até 23:59:59 do dia.
func fimDoEvento(e *domain.Evento) (time.Time, bool) {
	ref := e.FimEm
	if ref == nil {
		ref = e.InicioEm
	}
	if ref == nil {
		return time.Time{}, false
	}
	t := ref.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location()), true
}

// RepasseService concentra o arranjo de custódia (seção 2.3) num único
// lugar, para poder migrar para outro modelo depois.
type RepasseService struct {
	db          *gorm.DB
	eventos     *repository.EventoRepository
	itensPedido *repository.ItemPedidoRepository
	repasses    *repository.RepasseRepository
	config      *repository.ConfigPlataformaRepository
}

func NovoRepasseService(
	db *gorm.DB,
	eventos *repository.EventoRepository,
	itensPedido *repository.ItemPedidoRepository,
	repasses *repository.RepasseRepository,
	config *repository.ConfigPlataformaRepository,
) *RepasseService {
	return &RepasseService{db: db, eventos: eventos, itensPedido: itensPedido, repasses: repasses, config: config}
}

func (s *RepasseService) calcular(eventoID int64) (Financeiro, error) {
	itens, err := s.itensPedido.ListarAtivosComPagamento(eventoID)
	if err != nil {
		return Financeiro{}, err
	}
	var f Financeiro
	for _, it := range itens {
		f.BrutoCentavos += it.PrecoCentavos
		f.TaxaProcessadorCentavos += alocarTaxaProcessador(it.PagamentoTaxaProcCentavos, it.TotalCentavos, it.PagamentoValorCentavos)
	}
	f.LiquidoCentavos = f.BrutoCentavos - f.TaxaProcessadorCentavos
	return f, nil
}

type FinanceiroEvento struct {
	Financeiro
	LiberarEm *time.Time
	Repasse   *domain.Repasse
}

// FinanceiroDoEvento é o painel do organizador (seção 3): valores ao vivo
// enquanto não há repasse, e o repasse congelado depois.
func (s *RepasseService) FinanceiroDoEvento(organizadorID, eventoID int64) (*FinanceiroEvento, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, err
	}
	if evento.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}

	res := &FinanceiroEvento{}
	if rep, err := s.repasses.BuscarAtivoPorEvento(eventoID); err == nil {
		res.Repasse = rep
		res.Financeiro = Financeiro{rep.ValorBrutoCentavos, rep.TaxaProcessadorCentavos, rep.ValorLiquidoCentavos}
		res.LiberarEm = &rep.LiberarEm
		return res, nil
	}

	if res.Financeiro, err = s.calcular(eventoID); err != nil {
		return nil, err
	}
	if fim, ok := fimDoEvento(evento); ok {
		dias, _ := s.config.BuscarInt64(domain.ChaveRepasseDiasUteis)
		l := adicionarDiasUteis(fim, int(dias))
		res.LiberarEm = &l
	}
	return res, nil
}

// GerarRepasses é o job (seção 7.6): idempotente — o índice único em
// repasses(evento_id) impede duplicar, e a criação + lançamento da taxa
// do processador andam na mesma transação.
func (s *RepasseService) GerarRepasses() (int, error) {
	eventos, err := s.eventos.ListarSemRepasse()
	if err != nil {
		return 0, err
	}
	dias, err := s.config.BuscarInt64(domain.ChaveRepasseDiasUteis)
	if err != nil {
		return 0, fmt.Errorf("config REPASSE_DIAS_UTEIS ausente: %w", err)
	}

	agora := time.Now()
	criados := 0
	for i := range eventos {
		e := &eventos[i]
		fim, ok := fimDoEvento(e)
		if !ok {
			continue
		}
		liberarEm := adicionarDiasUteis(fim, int(dias))
		if agora.Before(liberarEm) {
			continue
		}
		f, err := s.calcular(e.ID)
		if err != nil {
			return criados, err
		}
		if f.BrutoCentavos == 0 {
			continue
		}

		err = s.db.Transaction(func(tx *gorm.DB) error {
			rep := &domain.Repasse{
				EventoID: e.ID, OrganizadorID: e.OrganizadorID,
				ValorBrutoCentavos: f.BrutoCentavos, TaxaProcessadorCentavos: f.TaxaProcessadorCentavos,
				ValorLiquidoCentavos: f.LiquidoCentavos, Status: domain.StatusRepassePendente,
				LiberarEm: liberarEm, CriadoEm: agora,
			}
			if err := s.repasses.Criar(tx, rep); err != nil {
				return err
			}
			if f.TaxaProcessadorCentavos > 0 {
				return tx.Create(&domain.Lancamento{
					Tipo: domain.LancamentoTaxaProcessador, ValorCentavos: f.TaxaProcessadorCentavos, Sinal: "-",
					EventoID: &e.ID, OrganizadorID: &e.OrganizadorID, CriadoEm: agora,
				}).Error
			}
			return nil
		})
		if err != nil {
			// duplicata por corrida entre dois jobs: o outro venceu
			continue
		}
		criados++
	}
	return criados, nil
}

func (s *RepasseService) ListarDoOrganizador(organizadorID int64) ([]repository.RepasseDetalhado, error) {
	return s.repasses.ListarPorOrganizador(organizadorID)
}

func (s *RepasseService) ListarAdmin(status string) ([]repository.RepasseDetalhado, error) {
	return s.repasses.ListarPorStatus(status)
}

// MarcarPago registra o Pix feito manualmente pelo admin. Transação +
// UPDATE condicional: segunda chamada não lança o repasse em dobro.
func (s *RepasseService) MarcarPago(id int64, comprovanteURL, observacao string) (*domain.Repasse, error) {
	rep, err := s.repasses.BuscarPorID(id)
	if err != nil {
		return nil, ErrRepasseNaoEncontrado
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		ok, err := s.repasses.MarcarPagoAtomico(tx, id, comprovanteURL, observacao)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRepasseNaoPendente
		}
		return tx.Create(&domain.Lancamento{
			Tipo: domain.LancamentoRepasse, ValorCentavos: rep.ValorLiquidoCentavos, Sinal: "-",
			EventoID: &rep.EventoID, OrganizadorID: &rep.OrganizadorID, CriadoEm: time.Now(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.repasses.BuscarPorID(id)
}
