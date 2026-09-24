package service

import (
	"crypto/subtle"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

type QRInfo struct {
	Payload  string
	Rotativo bool
	RenovaEm int // segundos até valer a pena buscar de novo (0 = estático)
}

// QRService entrega ao portador o texto do QR — fixo, ou rotativo se o
// evento ligou qr_rotativo.
type QRService struct {
	itens   *repository.ItemPedidoRepository
	pedidos *repository.PedidoRepository
	tipos   *repository.TipoIngressoRepository
	eventos *repository.EventoRepository
	segredo string
}

func NovoQRService(i *repository.ItemPedidoRepository, p *repository.PedidoRepository, t *repository.TipoIngressoRepository, e *repository.EventoRepository, segredo string) *QRService {
	return &QRService{itens: i, pedidos: p, tipos: t, eventos: e, segredo: segredo}
}

func (s *QRService) info(item *domain.ItemPedido) (*QRInfo, error) {
	if item.Status != domain.StatusItemPago && item.Status != domain.StatusItemUtilizado {
		return nil, ErrItemNaoEncontrado
	}
	tipo, err := s.tipos.BuscarPorID(item.TipoIngressoID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	evento, err := s.eventos.BuscarPorID(tipo.EventoID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	agora := time.Now()
	info := &QRInfo{Payload: PayloadQR(s.segredo, item.Codigo, item.QRToken, evento.QRRotativo, agora), Rotativo: evento.QRRotativo}
	if evento.QRRotativo {
		info.RenovaEm = JanelaQRSegundos - int(agora.Unix()%JanelaQRSegundos)
	}
	return info, nil
}

// DoComprador: só o dono do pedido.
func (s *QRService) DoComprador(usuarioID, itemID int64) (*QRInfo, error) {
	item, err := s.itens.BuscarPorID(itemID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	pedido, err := s.pedidos.BuscarPorID(item.PedidoID)
	if err != nil {
		return nil, ErrItemNaoEncontrado
	}
	if pedido.UsuarioID != usuarioID {
		return nil, ErrItemNaoPertenceAoUsuario
	}
	return s.info(item)
}

// PorLink: portador que recebeu o link por e-mail (cortesia/transferência);
// o segredo é o token estático na URL.
func (s *QRService) PorLink(codigo, token string) (*QRInfo, error) {
	item, err := s.itens.BuscarPorCodigo(codigo)
	if err != nil || subtle.ConstantTimeCompare([]byte(item.QRToken), []byte(token)) != 1 {
		return nil, ErrItemNaoEncontrado
	}
	return s.info(item)
}
