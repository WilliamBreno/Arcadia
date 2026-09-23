package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type WebhookHandler struct {
	checkout      *service.CheckoutService
	webhookSecret string
}

func NovoWebhookHandler(checkout *service.CheckoutService, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{checkout: checkout, webhookSecret: webhookSecret}
}

type mercadoPagoWebhookBody struct {
	Type string `json:"type"`
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

// validarAssinatura confere o header x-signature contra o segredo do
// webhook (documentação do Mercado Pago). Se MERCADOPAGO_WEBHOOK_SECRET
// não estiver configurado, pula essa checagem — a segurança real do
// endpoint vem de sempre reconsultar o pagamento na API (nunca confiar
// no corpo da notificação), o que acontece de qualquer forma.
func (h *WebhookHandler) validarAssinatura(c *gin.Context, dataID string) bool {
	if h.webhookSecret == "" {
		return true
	}

	assinatura := c.GetHeader("x-signature")
	requestID := c.GetHeader("x-request-id")
	if assinatura == "" {
		return false
	}

	var ts, v1 string
	for _, parte := range strings.Split(assinatura, ",") {
		chaveValor := strings.SplitN(strings.TrimSpace(parte), "=", 2)
		if len(chaveValor) != 2 {
			continue
		}
		switch chaveValor[0] {
		case "ts":
			ts = chaveValor[1]
		case "v1":
			v1 = chaveValor[1]
		}
	}
	if ts == "" || v1 == "" {
		return false
	}

	manifest := "id:" + strings.ToLower(dataID) + ";request-id:" + requestID + ";ts:" + ts + ";"
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write([]byte(manifest))
	esperado := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(esperado), []byte(v1))
}

// MercadoPago recebe a notificação (POST /webhooks/mercadopago). O MP
// aceita tanto query params (?type=payment&data.id=123) quanto corpo
// JSON — tentamos os dois. Sempre responde 200 rápido (mesmo em erro
// interno já registrado) para o MP não ficar reenviando em loop; só
// responde diferente de 200 quando a notificação em si é inválida.
func (h *WebhookHandler) MercadoPago(c *gin.Context) {
	tipo := c.Query("type")
	dataID := c.Query("data.id")

	if tipo == "" || dataID == "" {
		var body mercadoPagoWebhookBody
		if err := c.ShouldBindJSON(&body); err == nil {
			tipo = body.Type
			dataID = body.Data.ID
		}
	}

	if tipo != "payment" || dataID == "" {
		c.JSON(http.StatusOK, gin.H{"ok": true, "ignorado": true})
		return
	}

	if !h.validarAssinatura(c, dataID) {
		slog.Warn("webhook mercadopago com assinatura inválida", "data_id", dataID)
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "assinatura inválida"})
		return
	}

	if err := h.checkout.ProcessarWebhook(dataID); err != nil {
		slog.Error("erro ao processar webhook mercadopago", "erro", err, "data_id", dataID)
		// Responde 200 mesmo assim: o erro já está logado, e devolver erro
		// faz o MP reenviar — o que só ajuda se o problema for transitório
		// (nosso banco fora do ar, por exemplo). Reenvio automático do MP
		// cobre esse caso sem precisar sinalizar falha aqui.
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
