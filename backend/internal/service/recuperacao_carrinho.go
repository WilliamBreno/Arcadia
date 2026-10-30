package service

import (
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Recuperação de carrinho (item 7.2): quem reservou e não pagou recebe um
// e-mail, uma vez por evento, cerca de 1 hora depois de a reserva expirar,
// e só se ainda houver ingresso à venda e a pessoa não tiver comprado
// depois. Respeita "receber novidades" e traz o link para não receber mais
// (decisões do dono, 29/10/2026).

const (
	EsperaRecuperacao = time.Hour
	JanelaRecuperacao = 24 * time.Hour
	// Alvo do descadastro assinado (o mesmo mecanismo do item 6.4).
	TipoDescadastroCarrinho = "carrinho"
	AlvoDescadastroCarrinho = "lembretes"
)

type RecuperacaoCarrinhoService struct {
	db          *gorm.DB
	mail        *mail.Cliente
	seguir      *SeguirService
	frontendURL string
	nome        string
}

func NovoRecuperacaoCarrinhoService(db *gorm.DB, m *mail.Cliente, seguir *SeguirService, frontendURL, nome string) *RecuperacaoCarrinhoService {
	return &RecuperacaoCarrinhoService{db: db, mail: m, seguir: seguir, frontendURL: strings.TrimRight(frontendURL, "/"), nome: nome}
}

// ItemRetomar é um tipo de ingresso do carrinho, com quantos a pessoa queria.
type ItemRetomar struct {
	TipoID     int64
	Nome       string
	Quantidade int64
	Preco      int64
}

// ParametroRetomar monta ?retomar=12:2,13:1 (a página do evento já abre com
// esses ingressos escolhidos). Testada.
func ParametroRetomar(itens []ItemRetomar) string {
	partes := make([]string, 0, len(itens))
	for _, it := range itens {
		partes = append(partes, strconv.FormatInt(it.TipoID, 10)+":"+strconv.FormatInt(it.Quantidade, 10))
	}
	return strings.Join(partes, ",")
}

// Enviar roda no job de expirar reservas (a cada 5 minutos).
func (s *RecuperacaoCarrinhoService) Enviar(agora time.Time) (int, error) {
	var pedidos []domain.Pedido
	if err := s.db.Where("status = ? AND total_centavos > 0 AND recuperacao_em IS NULL AND canal = 'online' AND expira_em BETWEEN ? AND ?",
		domain.StatusPedidoExpirado, agora.Add(-JanelaRecuperacao), agora.Add(-EsperaRecuperacao)).
		Order("id").Limit(200).Find(&pedidos).Error; err != nil {
		return 0, err
	}
	enviados := 0
	for i := range pedidos {
		ok, err := s.processar(&pedidos[i], agora)
		if err != nil {
			slog.Warn("recuperação de carrinho", "pedido_id", pedidos[i].ID, "erro", err)
			continue
		}
		if ok {
			enviados++
		}
	}
	return enviados, nil
}

func (s *RecuperacaoCarrinhoService) processar(p *domain.Pedido, agora time.Time) (bool, error) {
	// Marca antes: dois jobs ao mesmo tempo não mandam dois e-mails.
	res := s.db.Model(&domain.Pedido{}).Where("id = ? AND recuperacao_em IS NULL", p.ID).Update("recuperacao_em", agora)
	if res.Error != nil || res.RowsAffected == 0 {
		return false, res.Error
	}

	var u struct {
		Nome              string
		Email             string
		ReceberNovidades  bool
		EmailVerificadoEm *time.Time
	}
	if err := s.db.Table("usuarios").Select("nome, email, receber_novidades, email_verificado_em").Where("id = ?", p.UsuarioID).Scan(&u).Error; err != nil {
		return false, err
	}
	// E-mail não verificado pode ser de outra pessoa (erro de digitação).
	if u.Email == "" || !u.ReceberNovidades || u.EmailVerificadoEm == nil {
		return false, nil
	}

	var evento domain.Evento
	if err := s.db.First(&evento, p.EventoID).Error; err != nil {
		return false, err
	}
	fim := evento.InicioEm
	if evento.FimEm != nil {
		fim = evento.FimEm
	}
	if evento.Status != domain.StatusEventoPublicado || fim == nil || !fim.After(agora) {
		return false, nil
	}

	// Comprou (ou está pagando) depois: não precisa de lembrete.
	var depois int64
	if err := s.db.Model(&domain.Pedido{}).
		Where("usuario_id = ? AND evento_id = ? AND id > ? AND status IN ?", p.UsuarioID, p.EventoID, p.ID,
			[]domain.StatusPedido{domain.StatusPedidoPago, domain.StatusPedidoAguardandoPagamento}).
		Count(&depois).Error; err != nil || depois > 0 {
		return false, err
	}

	itens, err := s.itensAindaAVenda(p.ID, agora)
	if err != nil || len(itens) == 0 {
		return false, err
	}

	res = s.db.Exec("INSERT INTO lembretes_enviados (evento_id, email, tipo) VALUES (?, ?, 'carrinho') ON CONFLICT DO NOTHING", evento.ID, strings.ToLower(u.Email))
	if res.Error != nil || res.RowsAffected == 0 {
		return false, res.Error
	}

	link := s.frontendURL + "/e/" + evento.Slug + "?" + url.Values{"retomar": {ParametroRetomar(itens)}}.Encode()
	descadastro := s.seguir.LinkDescadastro(p.UsuarioID, TipoDescadastroCarrinho, AlvoDescadastroCarrinho)
	corpo := CorpoRecuperacao(primeiroNome(u.Nome), &evento, itens, link, descadastro, s.nome)
	if err := s.mail.Enviar(u.Email, "Seus ingressos para "+evento.Titulo+" ainda estão disponíveis", corpo); err != nil {
		return false, err
	}
	return true, nil
}

// itensAindaAVenda: os tipos do pedido que ainda dá para comprar agora, na
// quantidade que a pessoa queria (sem estoque suficiente, fica de fora).
func (s *RecuperacaoCarrinhoService) itensAindaAVenda(pedidoID int64, agora time.Time) ([]ItemRetomar, error) {
	var linhas []struct {
		TipoID       int64
		Nome         string
		Quantidade   int64
		Preco        int64
		Ativo        bool
		Estoque      int64
		VendasInicio *time.Time
		VendasFim    *time.Time
		Ocupados     int64
	}
	err := s.db.Raw(`
		SELECT t.id AS tipo_id, t.nome, COUNT(i.id) AS quantidade, t.preco_centavos AS preco, t.ativo,
		       t.quantidade AS estoque, t.vendas_inicio, t.vendas_fim,
		       (SELECT COUNT(*) FROM itens_pedido o WHERE o.tipo_ingresso_id = t.id AND o.status IN ('reservado','pago','utilizado'))
		       + (SELECT COALESCE(SUM(f.quantidade),0) FROM fila_espera f WHERE f.tipo_ingresso_id = t.id AND f.status = 'convidado' AND f.prazo_em > now()) AS ocupados
		FROM itens_pedido i JOIN tipos_ingresso t ON t.id = i.tipo_ingresso_id
		WHERE i.pedido_id = ?
		GROUP BY t.id ORDER BY t.id`, pedidoID).Scan(&linhas).Error
	if err != nil {
		return nil, err
	}
	var out []ItemRetomar
	for _, l := range linhas {
		if !l.Ativo || (l.VendasInicio != nil && agora.Before(*l.VendasInicio)) || (l.VendasFim != nil && agora.After(*l.VendasFim)) {
			continue
		}
		if l.Estoque-l.Ocupados < l.Quantidade {
			continue
		}
		out = append(out, ItemRetomar{TipoID: l.TipoID, Nome: l.Nome, Quantidade: l.Quantidade, Preco: l.Preco})
	}
	return out, nil
}

// CorpoRecuperacao é o e-mail. Testada (escapa o que vem do organizador).
func CorpoRecuperacao(nome string, e *domain.Evento, itens []ItemRetomar, link, descadastro, plataforma string) string {
	var lista strings.Builder
	for _, it := range itens {
		fmt.Fprintf(&lista, "<li>%d × %s (%s cada, mais a taxa)</li>", it.Quantidade, html.EscapeString(it.Nome), formatarReais(it.Preco))
	}
	quando := DataDoEvento(e)
	return fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`+
		`<p>Olá, %s!</p>`+
		`<p>Você começou a comprar ingressos para <strong>%s</strong>%s, mas a reserva acabou antes do pagamento. Eles ainda estão à venda:</p>`+
		`<ul>%s</ul>`+
		`<p><a href="%s" style="background:#111;color:#fff;padding:10px 16px;border-radius:8px;text-decoration:none">Continuar a compra</a></p>`+
		`<p style="color:#555;font-size:13px">O preço e a disponibilidade valem na hora da compra.</p>`+
		`<p style="color:#777;font-size:12px">Você recebeu este e-mail porque começou uma compra no %s. `+
		`<a href="%s" style="color:#777">Não quero receber estes lembretes</a>.</p></div>`,
		html.EscapeString(nome), html.EscapeString(e.Titulo), quandoTexto(quando), lista.String(), link,
		html.EscapeString(plataforma), html.EscapeString(descadastro))
}

func quandoTexto(q string) string {
	if q == "" {
		return ""
	}
	return " (" + html.EscapeString(q) + ")"
}
