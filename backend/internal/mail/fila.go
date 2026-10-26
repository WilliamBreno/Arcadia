package mail

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Fila de e-mails (item 5.5). Com a fila ligada, Enviar/EnviarComImagens
// só gravam a mensagem no banco (rápido, não depende do provedor) e um
// processo em segundo plano envia, tentando de novo com intervalo
// crescente quando o Resend falha. Antes, uma falha do provedor fazia o
// e-mail (ex.: ingresso) se perder em silêncio.

const maxTentativas = 8

type emailFila struct {
	ID                 int64 `gorm:"primaryKey"`
	Destinatario       string
	Assunto            string
	CorpoHTML          string `gorm:"column:corpo_html"`
	Imagens            datatypes.JSON
	Status             string
	Tentativas         int
	UltimoErro         string
	ProximaTentativaEm time.Time
	CriadoEm           time.Time
	EnviadoEm          *time.Time
}

func (emailFila) TableName() string { return "emails_fila" }

// intervaloNovaTentativa: 1, 2, 4, 8... minutos, no máximo 2 horas.
// Função pura, testada.
func intervaloNovaTentativa(tentativas int) time.Duration {
	d := time.Minute << max(tentativas-1, 0)
	if d > 2*time.Hour || d <= 0 {
		return 2 * time.Hour
	}
	return d
}

// UsarFila liga a fila no banco. Sem chamar isto, o envio é direto (testes
// e ferramentas de linha de comando).
func (c *Cliente) UsarFila(db *gorm.DB) *Cliente {
	c.db = db
	return c
}

func (c *Cliente) enfileirar(destinatario, assunto, html string, imagens []ImagemInline) error {
	if imagens == nil {
		imagens = []ImagemInline{}
	}
	js, err := json.Marshal(imagens) // []byte vira base64 no JSON
	if err != nil {
		return err
	}
	agora := time.Now()
	return c.db.Create(&emailFila{
		Destinatario: destinatario, Assunto: assunto, CorpoHTML: html, Imagens: datatypes.JSON(js),
		Status: "pendente", ProximaTentativaEm: agora, CriadoEm: agora,
	}).Error
}

// ProcessarFila envia até `lote` e-mails pendentes. Primeiro "reserva" as
// mensagens numa transação curta (SKIP LOCKED + empurra a próxima tentativa
// 10 min para frente), depois envia fora da transação — assim várias
// instâncias trabalham juntas sem mandar o mesmo e-mail duas vezes, e uma
// queda no meio só faz a mensagem voltar para a fila depois de 10 min.
func (c *Cliente) ProcessarFila(lote int) (enviados, falhas int, err error) {
	if c.db == nil {
		return 0, 0, nil
	}
	var pendentes []emailFila
	err = c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND proxima_tentativa_em <= ?", "pendente", time.Now()).
			Order("proxima_tentativa_em").Limit(lote).Find(&pendentes).Error; err != nil {
			return err
		}
		if len(pendentes) == 0 {
			return nil
		}
		ids := make([]int64, len(pendentes))
		for i := range pendentes {
			ids[i] = pendentes[i].ID
		}
		return tx.Model(&emailFila{}).Where("id IN ?", ids).
			Update("proxima_tentativa_em", time.Now().Add(10*time.Minute)).Error
	})
	if err != nil {
		return 0, 0, err
	}

	for i := range pendentes {
		e := &pendentes[i]
		var imagens []ImagemInline
		_ = json.Unmarshal(e.Imagens, &imagens)
		erroEnvio := c.enviarAgora(e.Destinatario, e.Assunto, e.CorpoHTML, imagens)
		campos := map[string]any{"tentativas": e.Tentativas + 1}
		if erroEnvio == nil {
			campos["status"], campos["enviado_em"], campos["ultimo_erro"] = "enviado", time.Now(), ""
			enviados++
		} else {
			falhas++
			campos["ultimo_erro"] = erroEnvio.Error()
			if e.Tentativas+1 >= maxTentativas {
				campos["status"] = "falhou"
				slog.Error("e-mail desistido após várias tentativas", "id", e.ID, "para", e.Destinatario, "erro", erroEnvio)
			} else {
				campos["proxima_tentativa_em"] = time.Now().Add(intervaloNovaTentativa(e.Tentativas + 1))
			}
		}
		if err := c.db.Model(&emailFila{}).Where("id = ?", e.ID).Updates(campos).Error; err != nil {
			slog.Error("erro ao atualizar e-mail da fila", "id", e.ID, "erro", err)
		}
	}
	return enviados, falhas, nil
}

// IniciarProcessador roda a fila a cada `intervalo` até o contexto acabar.
func (c *Cliente) IniciarProcessador(ctx context.Context, intervalo time.Duration) {
	if c.db == nil {
		return
	}
	go func() {
		t := time.NewTicker(intervalo)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, _, err := c.ProcessarFila(20); err != nil {
					slog.Error("erro ao processar fila de e-mails", "erro", err)
				}
			}
		}
	}()
}

// ContarFalhos é usado no painel admin.
func (c *Cliente) ContarFalhos() int64 {
	if c.db == nil {
		return 0
	}
	var n int64
	c.db.Model(&emailFila{}).Where("status = ?", "falhou").Count(&n)
	return n
}
