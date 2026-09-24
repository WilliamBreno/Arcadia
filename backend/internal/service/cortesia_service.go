package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrCortesiaEventoNaoPublicado = errors.New("cortesias só podem ser emitidas em evento publicado")
	ErrCortesiaQuantidade         = errors.New("quantidade de cortesias deve ser entre 1 e 50")
	ErrCortesiaNaoRevogavel       = errors.New("cortesia não encontrada ou já utilizada/cancelada")
)

// CortesiaService emite ingressos gratuitos, sem taxa da plataforma e sem
// pagamento (seção 2.5 do checklist). O pedido pertence ao organizador, mas
// a cortesia não aparece em "Meus ingressos" dele (itens com cortesia=true
// ficam fora das consultas do comprador).
type CortesiaService struct {
	db          *gorm.DB
	eventos     *EventoService
	itensPedido *repository.ItemPedidoRepository
	pedidos     *repository.PedidoRepository
	mailCliente *mail.Cliente
	qrSecret    string
	frontendURL string
	plataforma  string
	promotor    PromotorListaEspera
}

func NovoCortesiaService(
	db *gorm.DB, eventos *EventoService, itensPedido *repository.ItemPedidoRepository, pedidos *repository.PedidoRepository,
	mailCliente *mail.Cliente, qrSecret, frontendURL, plataforma string,
) *CortesiaService {
	return &CortesiaService{db: db, eventos: eventos, itensPedido: itensPedido, pedidos: pedidos,
		mailCliente: mailCliente, qrSecret: qrSecret, frontendURL: frontendURL, plataforma: plataforma}
}

// Emitir cria N itens já `pago`, total 0, respeitando o estoque do tipo.
func (s *CortesiaService) Emitir(organizadorID, usuarioID, eventoID, tipoID int64, nome, email string, quantidade int) ([]domain.ItemPedido, error) {
	evento, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID)
	if err != nil {
		return nil, err
	}
	if evento.Status != domain.StatusEventoPublicado {
		return nil, ErrCortesiaEventoNaoPublicado
	}
	if quantidade < 1 || quantidade > 50 {
		return nil, ErrCortesiaQuantidade
	}

	var itens []domain.ItemPedido
	err = s.db.Transaction(func(tx *gorm.DB) error {
		tipo, err := s.itensPedido.BuscarTipoIngressoParaAtualizar(tx, tipoID)
		if err != nil || tipo.EventoID != eventoID {
			return fmt.Errorf("tipo de ingresso não encontrado")
		}
		ativos, err := s.itensPedido.ContarAtivosPorTipo(tx, tipo.ID)
		if err != nil {
			return err
		}
		if int64(quantidade) > int64(tipo.Quantidade)-ativos {
			return fmt.Errorf("%w: %s", ErrEstoqueInsuficiente, tipo.Nome)
		}

		agora := time.Now()
		pedido := &domain.Pedido{UsuarioID: usuarioID, EventoID: eventoID, Status: domain.StatusPedidoPago, CriadoEm: agora}
		if err := s.pedidos.Criar(tx, pedido); err != nil {
			return err
		}
		for i := 0; i < quantidade; i++ {
			codigo, err := gerarCodigoItem()
			if err != nil {
				return err
			}
			item := domain.ItemPedido{
				PedidoID: pedido.ID, TipoIngressoID: tipo.ID, TitularNome: nome, TitularEmail: email,
				Status: domain.StatusItemPago, Codigo: codigo, QRToken: assinarQR(s.qrSecret, codigo),
				Cortesia: true, CriadoEm: agora,
			}
			if err := s.itensPedido.Criar(tx, &item); err != nil {
				return err
			}
			itens = append(itens, item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.enviarEmail(evento, itens)
	return itens, nil
}

func (s *CortesiaService) enviarEmail(evento *domain.Evento, itens []domain.ItemPedido) {
	if len(itens) == 0 || itens[0].TitularEmail == "" {
		return
	}
	intro := fmt.Sprintf("Você recebeu %d ingresso(s) de cortesia.", len(itens))
	corpo, imagens := corpoEmailIngressos("Olá, "+itens[0].TitularNome+"!", intro, evento, itens, s.qrSecret, s.frontendURL)
	_ = s.mailCliente.EnviarComImagens(itens[0].TitularEmail, "Ingresso de cortesia — "+evento.Titulo+" — "+s.plataforma, corpo, imagens)
}

func (s *CortesiaService) Listar(organizadorID, eventoID int64) ([]repository.ItemVenda, error) {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return nil, err
	}
	return s.itensPedido.ListarCortesiasPorEvento(eventoID)
}

// Revogar cancela uma cortesia ainda não utilizada (sem reembolso — não há
// pagamento); a vaga volta ao estoque pelo status `cancelado`.
func (s *CortesiaService) Revogar(organizadorID, eventoID, itemID int64) error {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	ok, err := s.itensPedido.RevogarCortesia(eventoID, itemID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCortesiaNaoRevogavel
	}
	if s.promotor != nil {
		s.promotor.PromoverListaEspera(eventoID)
	}
	return nil
}

// DefinirPromotor liga a promoção da lista de espera (item 3.2).
func (s *CortesiaService) DefinirPromotor(p PromotorListaEspera) { s.promotor = p }
