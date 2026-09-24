package mail

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Cliente envia e-mails transacionais via Resend (seção 4 do plano).
// Sem RESEND_API_KEY configurada (dev sem conta Resend), apenas loga o
// conteúdo em vez de falhar — assim o fluxo de auth continua testável.
type Cliente struct {
	apiKey    string
	remetente string
	http      *http.Client
}

func NovoCliente(apiKey, remetente string) *Cliente {
	return &Cliente{apiKey: apiKey, remetente: remetente, http: &http.Client{Timeout: 15 * time.Second}}
}

// ImagemInline é uma imagem embutida no corpo (<img src="cid:ContentID">).
// Gmail/Outlook bloqueiam data: URIs; CID é o jeito que funciona.
type ImagemInline struct {
	ContentID string
	Nome      string
	PNG       []byte
}

type anexoResend struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id"`
}

type mensagemResend struct {
	From        string        `json:"from"`
	To          []string      `json:"to"`
	Subject     string        `json:"subject"`
	Html        string        `json:"html"`
	Attachments []anexoResend `json:"attachments,omitempty"`
}

func (c *Cliente) Enviar(destinatario, assunto, htmlCorpo string) error {
	return c.EnviarComImagens(destinatario, assunto, htmlCorpo, nil)
}

func (c *Cliente) EnviarComImagens(destinatario, assunto, htmlCorpo string, imagens []ImagemInline) error {
	if c.apiKey == "" {
		slog.Info("e-mail não enviado (RESEND_API_KEY ausente, modo dev)", "para", destinatario, "assunto", assunto, "imagens", len(imagens), "corpo", htmlCorpo)
		return nil
	}

	msg := mensagemResend{From: c.remetente, To: []string{destinatario}, Subject: assunto, Html: htmlCorpo}
	for _, img := range imagens {
		msg.Attachments = append(msg.Attachments, anexoResend{
			Filename: img.Nome, Content: base64.StdEncoding.EncodeToString(img.PNG), ContentType: "image/png", ContentID: img.ContentID,
		})
	}
	corpo, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Error("falha ao enviar e-mail", "para", destinatario, "assunto", assunto, "erro", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slog.Error("resend recusou o e-mail", "para", destinatario, "assunto", assunto, "status", resp.StatusCode)
		return fmt.Errorf("resend: status %d", resp.StatusCode)
	}
	return nil
}
