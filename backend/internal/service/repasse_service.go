package service

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
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
	TaxaAbsorvidaCentavos   int64 // taxas da plataforma pagas pelo organizador (item 5.1)
	LiquidoCentavos         int64
	BrutoCartaoCentavos     int64 // parte do bruto vendida no cartão (base da reserva, item 5.4)
	ReservaCentavos         int64 // reserva de risco segurada do líquido (item 5.4)
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
	db            *gorm.DB
	eventos       *repository.EventoRepository
	itensPedido   *repository.ItemPedidoRepository
	repasses      *repository.RepasseRepository
	config        *repository.ConfigPlataformaRepository
	pagamentos    *repository.PagamentoRepository
	organizadores *repository.OrganizadorRepository
	// Item 9.1: envio automático por Pix.
	payout          Payouter
	chaveAssinatura ed25519.PrivateKey
	mailCliente     *mail.Cliente
	nomePlataforma  string
}

// DefinirOrganizadores liga a reserva de risco (item 5.4).
func (s *RepasseService) DefinirOrganizadores(o *repository.OrganizadorRepository) {
	s.organizadores = o
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
		if it.TaxaAbsorvida {
			f.TaxaAbsorvidaCentavos += it.TaxaPlataformaCentavos
		}
		if it.PagamentoMetodo == "cartao" {
			f.BrutoCartaoCentavos += it.PrecoCentavos
		}
	}
	f.LiquidoCentavos = f.BrutoCentavos - f.TaxaProcessadorCentavos - f.TaxaAbsorvidaCentavos
	return f, nil
}

type FinanceiroEvento struct {
	Financeiro
	LiberarEm *time.Time
	Repasse   *domain.Repasse
	Metodos   *ResumoMetodos
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
	if res.Metodos, err = s.resumoMetodos(eventoID); err != nil {
		return nil, err
	}
	if rep, err := s.repasses.BuscarAtivoPorEvento(eventoID); err == nil {
		res.Repasse = rep
		res.Financeiro = Financeiro{
			BrutoCentavos: rep.ValorBrutoCentavos, TaxaProcessadorCentavos: rep.TaxaProcessadorCentavos,
			TaxaAbsorvidaCentavos: rep.TaxaAbsorvidaCentavos, LiquidoCentavos: rep.ValorLiquidoCentavos,
			ReservaCentavos: rep.ReservaCentavos,
		}
		res.LiberarEm = &rep.LiberarEm
		return res, nil
	}

	if res.Financeiro, err = s.calcular(eventoID); err != nil {
		return nil, err
	}
	if fim, ok := fimDoEvento(evento); ok {
		l := adicionarDiasUteis(fim, s.diasParaRepasse(evento.OrganizadorID)) // 9.1: verificado recebe antes
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
	if _, err := s.config.BuscarInt64(domain.ChaveRepasseDiasUteis); err != nil {
		return 0, fmt.Errorf("config REPASSE_DIAS_UTEIS ausente: %w", err)
	}
	envio := EnvioManual
	if s.EnvioLigado() {
		envio = EnvioNaFila // item 9.1: vai sozinho (ou espera o admin, se não puder)
	}

	agora := time.Now()
	criados := 0
	for i := range eventos {
		e := &eventos[i]
		fim, ok := fimDoEvento(e)
		if !ok {
			continue
		}
		liberarEm := adicionarDiasUteis(fim, s.diasParaRepasse(e.OrganizadorID))
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
		reserva, reservaAte, err := s.reservaDoEvento(e, f, liberarEm)
		if err != nil {
			return criados, err
		}
		reservaStatus := domain.ReservaNenhuma
		if reserva > 0 {
			reservaStatus = domain.ReservaRetida
		}

		err = s.db.Transaction(func(tx *gorm.DB) error {
			rep := &domain.Repasse{
				EventoID: e.ID, OrganizadorID: e.OrganizadorID,
				ValorBrutoCentavos: f.BrutoCentavos, TaxaProcessadorCentavos: f.TaxaProcessadorCentavos,
				TaxaAbsorvidaCentavos: f.TaxaAbsorvidaCentavos,
				ReservaCentavos:       reserva, ReservaLiberarEm: reservaAte, ReservaStatus: reservaStatus,
				ValorLiquidoCentavos: f.LiquidoCentavos, Status: domain.StatusRepassePendente,
				LiberarEm: liberarEm, CriadoEm: agora, EnvioStatus: envio,
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
	// Item 9.1: com Pix automático no meio do caminho, marcar à mão pagaria duas vezes.
	if rep.EnvioStatus == EnvioEnviando || rep.EnvioStatus == EnvioProcessando {
		return nil, ErrEnvioEmAndamento
	}
	return s.marcarPago(rep, comprovanteURL, observacao)
}

func (s *RepasseService) marcarPago(rep *domain.Repasse, comprovanteURL, observacao string) (*domain.Repasse, error) {
	id := rep.ID

	err := s.db.Transaction(func(tx *gorm.DB) error {
		ok, err := s.repasses.MarcarPagoAtomico(tx, id, comprovanteURL, observacao)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRepasseNaoPendente
		}
		// Paga agora o líquido menos a reserva de risco (item 5.4).
		return tx.Create(&domain.Lancamento{
			Tipo: domain.LancamentoRepasse, ValorCentavos: ValorAPagarAgora(rep), Sinal: "-",
			EventoID: &rep.EventoID, OrganizadorID: &rep.OrganizadorID, CriadoEm: time.Now(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.repasses.BuscarPorID(id)
}
