package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Reserva de risco e verificação do organizador (item 5.4, decisões do
// dono em 25/10/2026):
//   - organizador NOVO = não verificado OU com menos de 3 repasses pagos;
//   - do organizador novo, segura 10% das vendas no CARTÃO do evento
//     (Pix não tem contestação desse tipo) por 60 dias após o repasse;
//   - se houver contestação perdida, o ADMIN decide se usa a reserva
//     (nada automático); senão a reserva é liberada para pagamento no fim
//     do prazo.

var (
	ErrReservaIndisponivel  = errors.New("este repasse não tem reserva retida")
	ErrReservaNoPrazo       = errors.New("a reserva ainda está no prazo de retenção")
	ErrRepasseAindaPendente = errors.New("pague o repasse antes de liberar a reserva")
)

// calcularReserva: percentual (pontos-base) das vendas no cartão, só para
// organizador novo, nunca maior que o líquido. Função pura, testada.
func calcularReserva(brutoCartao, liquido, bp int64, novo bool) int64 {
	if !novo || brutoCartao <= 0 || bp <= 0 || liquido <= 0 {
		return 0
	}
	r := brutoCartao * bp / 10000
	if r > liquido {
		r = liquido
	}
	return r
}

// organizadorNovo: sem verificação ou com poucos repasses pagos.
func organizadorNovo(verificado bool, repassesPagos, minimo int64) bool {
	return !verificado || repassesPagos < minimo
}

// ValorAPagarAgora é o que o admin envia no repasse (líquido − reserva).
func ValorAPagarAgora(r *domain.Repasse) int64 {
	return r.ValorLiquidoCentavos - r.ReservaCentavos
}

// regraReserva lê a config (com padrões seguros se faltar).
func (s *RepasseService) regraReserva() (bp, dias, minimo int64) {
	bp, dias, minimo = 1000, 60, 3
	if v, err := s.config.BuscarInt64(domain.ChaveReservaRiscoBP); err == nil {
		bp = v
	}
	if v, err := s.config.BuscarInt64(domain.ChaveReservaRiscoDias); err == nil {
		dias = v
	}
	if v, err := s.config.BuscarInt64(domain.ChaveReservaRiscoEventosMin); err == nil {
		minimo = v
	}
	return
}

// reservaDoEvento decide a reserva no momento de gerar o repasse.
func (s *RepasseService) reservaDoEvento(e *domain.Evento, f Financeiro, liberarEm time.Time) (int64, *time.Time, error) {
	if s.organizadores == nil {
		return 0, nil, nil
	}
	org, err := s.organizadores.BuscarPorID(e.OrganizadorID)
	if err != nil {
		return 0, nil, err
	}
	pagos, err := s.repasses.ContarPagosDoOrganizador(e.OrganizadorID)
	if err != nil {
		return 0, nil, err
	}
	bp, dias, minimo := s.regraReserva()
	reserva := calcularReserva(f.BrutoCartaoCentavos, f.LiquidoCentavos, bp, organizadorNovo(org.VerificadoEm != nil, pagos, minimo))
	if reserva == 0 {
		return 0, nil, nil
	}
	ate := liberarEm.AddDate(0, 0, int(dias))
	return reserva, &ate, nil
}

// PagarReserva: o admin enviou o Pix da reserva (só depois do prazo e do
// repasse principal pago).
func (s *RepasseService) PagarReserva(id int64, comprovanteURL, observacao string) (*domain.Repasse, error) {
	return s.finalizarReserva(id, domain.ReservaPaga, comprovanteURL, observacao)
}

// UsarReserva: o admin usa a reserva para cobrir uma contestação perdida
// (decisão manual — nunca automática).
func (s *RepasseService) UsarReserva(id int64, observacao string) (*domain.Repasse, error) {
	return s.finalizarReserva(id, domain.ReservaUsada, "", observacao)
}

func (s *RepasseService) finalizarReserva(id int64, destino, comprovanteURL, observacao string) (*domain.Repasse, error) {
	rep, err := s.repasses.BuscarPorID(id)
	if err != nil {
		return nil, ErrRepasseNaoEncontrado
	}
	if rep.ReservaStatus != domain.ReservaRetida {
		return nil, ErrReservaIndisponivel
	}
	if destino == domain.ReservaPaga {
		if rep.Status != domain.StatusRepassePago {
			return nil, ErrRepasseAindaPendente
		}
		if rep.ReservaLiberarEm != nil && time.Now().Before(*rep.ReservaLiberarEm) {
			return nil, ErrReservaNoPrazo
		}
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		ok, err := s.repasses.FinalizarReservaAtomico(tx, id, destino, comprovanteURL, observacao)
		if err != nil {
			return err
		}
		if !ok {
			return ErrReservaIndisponivel
		}
		tipo := domain.LancamentoRepasse // Pix da reserva enviado ao organizador
		if destino == domain.ReservaUsada {
			tipo = domain.LancamentoReservaUsada // reserva fica com a plataforma
		}
		return tx.Create(&domain.Lancamento{
			Tipo: tipo, ValorCentavos: rep.ReservaCentavos, Sinal: "-",
			EventoID: &rep.EventoID, OrganizadorID: &rep.OrganizadorID, CriadoEm: time.Now(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.repasses.BuscarPorID(id)
}
