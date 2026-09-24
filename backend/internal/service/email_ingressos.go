package service

import (
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// corpoEmailIngressos monta o e-mail de ingressos: confirmação + QR embutido
// (PNG por CID) + link do ingresso online. Evento com QR rotativo não leva
// imagem (um QR fixo seria recusado na portaria): só o link, que abre o QR
// dinâmico. Todo texto vindo de usuário/organizador é escapado.
func corpoEmailIngressos(saudacao, intro string, evento *domain.Evento, itens []domain.ItemPedido, qrSecret, frontendURL string) (string, []mail.ImagemInline) {
	var b strings.Builder
	var imagens []mail.ImagemInline

	fmt.Fprintf(&b, `<div style="font-family:Arial,sans-serif;max-width:560px;margin:auto;color:#111">`)
	fmt.Fprintf(&b, `<p>%s</p><p>%s</p>`, html.EscapeString(saudacao), intro)
	fmt.Fprintf(&b, `<h2 style="margin:16px 0 4px">%s</h2>`, html.EscapeString(evento.Titulo))
	if evento.InicioEm != nil {
		fmt.Fprintf(&b, `<p style="margin:0 0 16px;color:#555">%s</p>`, evento.InicioEm.In(time.Local).Format("02/01/2006 às 15:04"))
	}

	for i, it := range itens {
		fmt.Fprintf(&b, `<div style="border:1px solid #ddd;border-radius:8px;padding:12px;margin:12px 0;text-align:center">`)
		fmt.Fprintf(&b, `<p style="margin:0 0 4px"><strong>%s</strong></p>`, html.EscapeString(it.TitularNome))
		fmt.Fprintf(&b, `<p style="margin:0 0 8px;font-family:monospace;font-size:16px">%s</p>`, html.EscapeString(it.Codigo))

		if !evento.QRRotativo {
			if png, err := qrcode.Encode(it.Codigo+":"+it.QRToken, qrcode.Medium, 300); err == nil {
				cid := fmt.Sprintf("qr%d-%s", i, it.Codigo)
				imagens = append(imagens, mail.ImagemInline{ContentID: cid, Nome: "ingresso-" + it.Codigo + ".png", PNG: png})
				fmt.Fprintf(&b, `<img src="cid:%s" alt="QR do ingresso %s" width="220" height="220" style="display:block;margin:0 auto 8px">`, cid, html.EscapeString(it.Codigo))
			} else {
				slog.Error("erro ao gerar QR do e-mail", "codigo", it.Codigo, "erro", err)
			}
		}
		link := fmt.Sprintf("%s/ingresso/%s/%s", frontendURL, it.Codigo, it.QRToken)
		fmt.Fprintf(&b, `<p style="margin:0"><a href="%s">Abrir ingresso online</a></p></div>`, link)
	}

	if evento.QRRotativo {
		b.WriteString(`<p style="color:#555;font-size:13px">Neste evento o QR muda a cada 30 segundos: na entrada, abra o ingresso pelo link acima (precisa de internet). Print ou foto do QR não funciona.</p>`)
	} else {
		b.WriteString(`<p style="color:#555;font-size:13px">Apresente o QR na entrada (na tela do celular ou impresso). Guarde este e-mail: quem tiver o QR consegue entrar, então não o compartilhe.</p>`)
	}
	b.WriteString(`</div>`)
	return b.String(), imagens
}

func formatarReais(centavos int64) string {
	return fmt.Sprintf("R$ %d,%02d", centavos/100, centavos%100)
}
