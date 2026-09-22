package mail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Cliente envia e-mails transacionais via Resend (seção 4 do plano).
// Sem RESEND_API_KEY configurada (dev sem conta Resend), apenas loga o
// conteúdo em vez de falhar — assim o fluxo de auth continua testável.
type Cliente struct {
	apiKey    string
	remetente string
}

func NovoCliente(apiKey, remetente string) *Cliente {
	return &Cliente{apiKey: apiKey, remetente: remetente}
}

type mensagemResend struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
}

func (c *Cliente) Enviar(destinatario, assunto, htmlCorpo string) error {
	if c.apiKey == "" {
		slog.Info("e-mail não enviado (RESEND_API_KEY ausente, modo dev)", "para", destinatario, "assunto", assunto, "corpo", htmlCorpo)
		return nil
	}

	corpo, err := json.Marshal(mensagemResend{
		From:    c.remetente,
		To:      []string{destinatario},
		Subject: assunto,
		Html:    htmlCorpo,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend: status %d", resp.StatusCode)
	}
	return nil
}
