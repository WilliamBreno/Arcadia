package service

// Item 9.4: venda na porta pelo celular. Decisões do dono (30/10/2026):
//   - Pix na tela (Mercado Pago, entra no repasse como venda online) ou
//     dinheiro registrado (o organizador fica com o dinheiro);
//   - mesma taxa: no Pix, somada ao preço (ou absorvida, se o evento for
//     assim); no dinheiro, o comprador paga só o preço e a taxa sai do
//     repasse do organizador (pode deixar saldo devedor);
//   - nome e e-mail opcionais; o ingresso já nasce usado (a pessoa entrou),
//     sem garantia e sem cancelamento pelo comprador;
//   - vendem o organizador e a equipe de check-in, os tipos ativos, no mesmo
//     estoque do online, até o fim do evento (mesmo com a venda online
//     encerrada).
// O pedido fica no usuário de sistema "Bilheteria" (nada aparece na conta de
// quem vendeu); vendido_por guarda a pessoa da equipe.

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

const (
	MetodoPortaPix      = "pix"
	MetodoPortaDinheiro = "dinheiro"
	maxIngressosPorta   = 20
)

var (
	ErrPortaSemAcesso   = errors.New("só o organizador e a equipe de check-in vendem na porta")
	ErrPortaFechada     = errors.New("a venda na porta só vale com o evento publicado e até o fim dele")
	ErrPortaInvalida    = errors.New("venda inválida: escolha os ingressos (até 20) e a forma de pagamento")
	ErrPortaEmail       = errors.New("e-mail do comprador inválido")
	ErrBilheteriaFaltou = errors.New("usuário de sistema da bilheteria não encontrado (rode as migrações)")
)

type ItemPorta struct {
	TipoIngressoID int64
	Quantidade     int
}

// DefinirAcessoPorta liga a venda na porta (quem tem acesso ao check-in).
func (s *CheckoutService) DefinirAcessoPorta(temAcesso func(usuarioID, eventoID int64) bool) {
	s.acessoPorta = temAcesso
}

func (s *CheckoutService) bilheteriaID() (int64, error) {
	var id int64
	if err := s.db.Table("usuarios").Where("email = ?", domain.EmailBilheteria).Pluck("id", &id).Error; err != nil || id == 0 {
		return 0, ErrBilheteriaFaltou
	}
	return id, nil
}

func (s *CheckoutService) podeVenderNaPorta(usuarioID, eventoID int64) bool {
	return s.acessoPorta != nil && s.acessoPorta(usuarioID, eventoID)
}

// VenderNaPorta cria o pedido da porta. Dinheiro: pago e usado na hora.
// Pix: reservado até o pagamento (o QR sai em GerarPixPorta).
func (s *CheckoutService) VenderNaPorta(vendedorID, eventoID int64, itensReq []ItemPorta, metodo, nome, email string) (*domain.Pedido, []domain.ItemPedido, error) {
	if !s.podeVenderNaPorta(vendedorID, eventoID) {
		return nil, nil, ErrPortaSemAcesso
	}
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, nil, ErrPortaFechada
	}
	agora := time.Now()
	if fim, ok := fimDoEvento(evento); evento.Status != domain.StatusEventoPublicado || (ok && agora.After(fim)) {
		return nil, nil, ErrPortaFechada
	}
	total := 0
	for _, ir := range itensReq {
		if ir.Quantidade < 0 {
			return nil, nil, ErrPortaInvalida
		}
		total += ir.Quantidade
	}
	if total < 1 || total > maxIngressosPorta || (metodo != MetodoPortaPix && metodo != MetodoPortaDinheiro) {
		return nil, nil, ErrPortaInvalida
	}
	nome, email = strings.TrimSpace(nome), strings.ToLower(strings.TrimSpace(email))
	if len([]rune(nome)) > 120 {
		nome = string([]rune(nome)[:120])
	}
	if email != "" {
		if a, err := mail.ParseAddress(email); err != nil || a.Address != email {
			return nil, nil, ErrPortaEmail
		}
	}
	if nome == "" {
		nome = "Venda na porta"
	}
	if metodo == MetodoPortaPix && !s.mp.Habilitado() {
		return nil, nil, ErrMercadoPagoNaoConfigurado
	}
	bilheteria, err := s.bilheteriaID()
	if err != nil {
		return nil, nil, err
	}
	taxas, err := s.taxas.ParaEvento(evento)
	if err != nil {
		return nil, nil, err
	}
	if metodo == MetodoPortaDinheiro {
		taxas.Absorvida = true // dinheiro: o comprador paga só o preço; a taxa sai do repasse
	}
	reservaMinutos, err := s.config.BuscarInt64(domain.ChaveReservaMinutos)
	if err != nil {
		return nil, nil, err
	}

	var pedido *domain.Pedido
	var itens []domain.ItemPedido
	err = s.db.Transaction(func(tx *gorm.DB) error {
		met := metodo
		pedido = &domain.Pedido{UsuarioID: bilheteria, EventoID: eventoID, Status: domain.StatusPedidoAguardandoPagamento,
			Canal: domain.CanalPorta, VendidoPor: &vendedorID, MetodoPorta: &met, CriadoEm: agora}
		if metodo == MetodoPortaDinheiro {
			pedido.Status = domain.StatusPedidoPago
		} else {
			expira := agora.Add(time.Duration(reservaMinutos) * time.Minute)
			pedido.ExpiraEm = &expira
		}
		if err := s.pedidos.Criar(tx, pedido); err != nil {
			return err
		}
		var lancamentos []domain.Lancamento
		for _, ir := range itensReq {
			if ir.Quantidade == 0 {
				continue
			}
			tipo, err := s.itensPedido.BuscarTipoIngressoParaAtualizar(tx, ir.TipoIngressoID)
			if err != nil || tipo.EventoID != eventoID || tipo.Interno() {
				return fmt.Errorf("%w: tipo de ingresso não encontrado", ErrPortaInvalida)
			}
			if !tipo.Ativo {
				return fmt.Errorf("%w: %s", ErrTipoIndisponivel, tipo.Nome)
			}
			ativos, err := s.itensPedido.ContarAtivosPorTipo(tx, tipo.ID)
			if err != nil {
				return err
			}
			guardadas, err := VagasGuardadas(tx, tipo.ID, 0)
			if err != nil {
				return err
			}
			// Porta vende só o estoque do organizador (não mexe na revenda).
			if livre := int64(tipo.Quantidade) - ativos - guardadas; int64(ir.Quantidade) > livre {
				if livre < 0 {
					livre = 0
				}
				return fmt.Errorf("%w: %s (restam %d)", ErrEstoqueInsuficiente, tipo.Nome, livre)
			}
			v := taxas.Calcular(tipo.PrecoCentavos, false)
			for i := 0; i < ir.Quantidade; i++ {
				codigo, err := gerarCodigoItem()
				if err != nil {
					return err
				}
				item := domain.ItemPedido{
					PedidoID: pedido.ID, TipoIngressoID: tipo.ID, TitularNome: nome, TitularEmail: email,
					PrecoCentavos: tipo.PrecoCentavos, TaxaPlataformaCentavos: v.TaxaCentavos, TaxaAbsorvida: v.TaxaAbsorvida,
					TotalCentavos: v.TotalCentavos, Status: domain.StatusItemReservado, Codigo: codigo, QRToken: s.gerarQRToken(codigo), CriadoEm: agora,
				}
				if metodo == MetodoPortaDinheiro {
					item.Status, item.UtilizadoEm = domain.StatusItemUtilizado, &agora
				}
				if err := s.itensPedido.Criar(tx, &item); err != nil {
					return err
				}
				if metodo == MetodoPortaDinheiro && item.TaxaPlataformaCentavos > 0 {
					// O dinheiro ficou com o organizador: só a taxa entra no ledger
					// (receita da Evve, cobrada do saldo dele).
					lancamentos = append(lancamentos,
						domain.Lancamento{Tipo: domain.LancamentoTaxaPlataforma, ValorCentavos: item.TaxaPlataformaCentavos, Sinal: "+",
							EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora},
						domain.Lancamento{Tipo: domain.LancamentoTaxaAbsorvida, ValorCentavos: item.TaxaPlataformaCentavos, Sinal: "-",
							EventoID: &evento.ID, OrganizadorID: &evento.OrganizadorID, PedidoID: &pedido.ID, ItemID: &item.ID, CriadoEm: agora})
				}
				itens = append(itens, item)
				pedido.TotalCentavos += item.TotalCentavos
			}
		}
		if len(lancamentos) > 0 {
			if err := tx.Create(&lancamentos).Error; err != nil {
				return err
			}
		}
		return tx.Save(pedido).Error
	})
	if err != nil {
		return nil, nil, err
	}
	if metodo == MetodoPortaDinheiro {
		s.enviarIngressosPorta(pedido, itens, evento)
	}
	return pedido, itens, nil
}

// pedidoDaPorta confere que o pedido é da porta deste evento e que quem
// pede tem acesso.
func (s *CheckoutService) pedidoDaPorta(vendedorID, eventoID, pedidoID int64) (*domain.Pedido, error) {
	if !s.podeVenderNaPorta(vendedorID, eventoID) {
		return nil, ErrPortaSemAcesso
	}
	p, err := s.pedidos.BuscarPorID(pedidoID)
	if err != nil || p.EventoID != eventoID || p.Canal != domain.CanalPorta {
		return nil, ErrPedidoNaoEncontrado
	}
	return p, nil
}

// GerarPixPorta mostra o QR no celular da equipe.
func (s *CheckoutService) GerarPixPorta(vendedorID, eventoID, pedidoID int64) (*PixNaTela, error) {
	p, err := s.pedidoDaPorta(vendedorID, eventoID, pedidoID)
	if err != nil {
		return nil, err
	}
	return s.GerarPix(p.UsuarioID, p.ID)
}

// SituacaoPorta consulta o Pix (sem esperar o webhook) e devolve o pedido.
func (s *CheckoutService) SituacaoPorta(vendedorID, eventoID, pedidoID int64) (*domain.Pedido, []domain.ItemPedido, error) {
	p, err := s.pedidoDaPorta(vendedorID, eventoID, pedidoID)
	if err != nil {
		return nil, nil, err
	}
	if p.Status == domain.StatusPedidoAguardandoPagamento && p.PixMPPaymentID != nil {
		if atual, err := s.VerificarPagamento(p.UsuarioID, p.ID, ""); err == nil {
			p = atual
		}
	}
	itens, err := s.itensPedido.ListarPorPedido(p.ID)
	return p, itens, err
}

// marcarEntradaPorta: Pix da porta pago = ingresso já usado (a pessoa entrou).
func marcarEntradaPorta(tx *gorm.DB, pedido *domain.Pedido, agora time.Time) error {
	if pedido.Canal != domain.CanalPorta {
		return nil
	}
	return tx.Model(&domain.ItemPedido{}).Where("pedido_id = ? AND status = ?", pedido.ID, domain.StatusItemPago).
		Updates(map[string]any{"status": domain.StatusItemUtilizado, "utilizado_em": agora}).Error
}

// enviarIngressosPorta manda o ingresso se o comprador deu e-mail.
func (s *CheckoutService) enviarIngressosPorta(pedido *domain.Pedido, itens []domain.ItemPedido, evento *domain.Evento) {
	if len(itens) == 0 || itens[0].TitularEmail == "" {
		return
	}
	intro := fmt.Sprintf("Compra na porta de %s confirmada (pedido #%d). Seu ingresso já foi usado na entrada; guarde este e-mail como comprovante.", evento.Titulo, pedido.ID)
	corpo, imagens := corpoEmailIngressos("Olá!", intro, evento, itens, s.qrSecret, s.frontendURL)
	_ = s.mailCliente.EnviarComImagens(itens[0].TitularEmail, "Ingresso — "+evento.Titulo+" — "+s.nomePlataforma, corpo, imagens)
}

// ResumoPorta: vendas da porta por pessoa da equipe (o organizador vê todas;
// a equipe, só as próprias).
type LinhaPorta struct {
	VendedorID           int64
	Vendedor             string
	Ingressos            int64
	PixCentavos          int64
	DinheiroCentavos     int64
	TaxaDinheiroCentavos int64
}

func (s *CheckoutService) ResumoPorta(usuarioID, eventoID int64, todos bool) ([]LinhaPorta, error) {
	if !s.podeVenderNaPorta(usuarioID, eventoID) {
		return nil, ErrPortaSemAcesso
	}
	q := `SELECT p.vendido_por AS vendedor_id, u.nome AS vendedor, COUNT(i.id) AS ingressos,
		COALESCE(SUM(i.total_centavos) FILTER (WHERE p.metodo_porta = 'pix'), 0) AS pix_centavos,
		COALESCE(SUM(i.preco_centavos) FILTER (WHERE p.metodo_porta = 'dinheiro'), 0) AS dinheiro_centavos,
		COALESCE(SUM(i.taxa_plataforma_centavos) FILTER (WHERE p.metodo_porta = 'dinheiro'), 0) AS taxa_dinheiro_centavos
		FROM pedidos p JOIN itens_pedido i ON i.pedido_id = p.id JOIN usuarios u ON u.id = p.vendido_por
		WHERE p.evento_id = ? AND p.canal = 'porta' AND i.status IN ('pago', 'utilizado')`
	args := []any{eventoID}
	if !todos {
		q += ` AND p.vendido_por = ?`
		args = append(args, usuarioID)
	}
	var out []LinhaPorta
	err := s.db.Raw(q+` GROUP BY p.vendido_por, u.nome ORDER BY u.nome`, args...).Scan(&out).Error
	return out, err
}

// EhOrganizadorDoEvento diz se o resumo mostra a equipe toda.
func (s *CheckoutService) EhOrganizadorDoEvento(usuarioID, eventoID int64) bool {
	var n int64
	s.db.Table("eventos e").Joins("JOIN organizadores o ON o.id = e.organizador_id").
		Where("e.id = ? AND o.usuario_id = ?", eventoID, usuarioID).Count(&n)
	return n > 0
}

// TipoPorta é o que a tela da porta mostra para cada ingresso.
type TipoPorta struct {
	ID            int64
	Nome          string
	PrecoCentavos int64
	TotalPix      int64 // preço + taxa (ou só o preço, se o evento absorve)
	Disponivel    int64
}

func (s *CheckoutService) TiposPorta(usuarioID, eventoID int64) ([]TipoPorta, error) {
	if !s.podeVenderNaPorta(usuarioID, eventoID) {
		return nil, ErrPortaSemAcesso
	}
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, err
	}
	taxas, err := s.taxas.ParaEvento(evento)
	if err != nil {
		return nil, err
	}
	var tipos []domain.TipoIngresso
	if err := s.db.Where("evento_id = ? AND ativo AND NOT inscricao_talento AND NOT espaco_expositor", eventoID).Order("ordem, id").Find(&tipos).Error; err != nil {
		return nil, err
	}
	out := make([]TipoPorta, 0, len(tipos))
	for _, t := range tipos {
		ativos, _ := s.itensPedido.ContarAtivosPorTipo(s.db, t.ID)
		guardadas, _ := VagasGuardadas(s.db, t.ID, 0)
		livre := int64(t.Quantidade) - ativos - guardadas
		if livre < 0 {
			livre = 0
		}
		out = append(out, TipoPorta{ID: t.ID, Nome: t.Nome, PrecoCentavos: t.PrecoCentavos, TotalPix: taxas.Calcular(t.PrecoCentavos, false).TotalCentavos, Disponivel: livre})
	}
	return out, nil
}
