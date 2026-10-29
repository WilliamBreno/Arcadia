package config

import (
	"strings"
	"testing"
)

func configProducaoValida() Config {
	return Config{
		AmbienteApp: "production", JWTSecret: "0123456789abcdef0123456789abcdef", CronSecret: "cron-secret-bem-grande",
		FrontendURL: "https://evve.app", BackendURL: "https://api.evve.app", DatabaseURL: "postgres://u:p@db.exemplo.com/evve",
		StorageDriver: "s3", MercadoPagoAccessToken: "APP_USR-x", MercadoPagoWebhookSecret: "w", MercadoPagoPublicKey: "APP_USR-pub", ResendAPIKey: "re_x",
		EmailRemetente: "Evve <no-reply@evve.app>", AlertaWebhookURL: "https://discord.example/webhook",
	}
}

func TestValidarProducaoAceitaConfigCompleta(t *testing.T) {
	erros, avisos := configProducaoValida().ValidarProducao()
	if len(erros) != 0 || len(avisos) != 0 {
		t.Errorf("config completa não deveria ter erros/avisos: %v %v", erros, avisos)
	}
}

func TestValidarProducaoRecusaInseguro(t *testing.T) {
	c := configProducaoValida()
	c.JWTSecret = jwtSecretPadrao
	c.CronSecret = ""
	c.FrontendURL = "http://evve.app"
	c.BackendURL = "http://api.evve.app"
	c.DatabaseURL = "postgres://u:p@localhost/x"
	c.StorageDriver = "disco"
	erros, _ := c.ValidarProducao()
	if len(erros) != 6 {
		t.Errorf("esperava 6 erros, veio %d: %v", len(erros), erros)
	}
}

func TestValidarProducaoIgnoraDesenvolvimento(t *testing.T) {
	erros, avisos := Config{AmbienteApp: "development", JWTSecret: jwtSecretPadrao}.ValidarProducao()
	if erros != nil || avisos != nil {
		t.Error("desenvolvimento não valida")
	}
}

func TestAvisosDeTokenTesteEEmailDev(t *testing.T) {
	c := configProducaoValida()
	c.MercadoPagoAccessToken = "TEST-abc"
	c.EmailRemetente = "Evve <no-reply@evve.local>"
	c.ResendAPIKey = ""
	_, avisos := c.ValidarProducao()
	if len(avisos) != 3 {
		t.Errorf("esperava 3 avisos, veio %d: %v", len(avisos), avisos)
	}
}

func TestValidarProducaoRecusaMercadoPagoFalso(t *testing.T) {
	c := configProducaoValida()
	c.MercadoPagoBaseURL = "http://localhost:9099"
	erros, _ := c.ValidarProducao()
	if len(erros) == 0 {
		t.Error("MERCADOPAGO_BASE_URL definida em produção deveria ser erro")
	}
}

func TestAvisoChavePublicaDoCartao(t *testing.T) {
	c := configProducaoValida()
	c.MercadoPagoPublicKey = ""
	if _, avisos := c.ValidarProducao(); len(avisos) != 1 || !strings.Contains(avisos[0], "MERCADOPAGO_PUBLIC_KEY") {
		t.Errorf("sem chave pública: aviso (o cartão cai no plano B), veio %v", avisos)
	}
	c.MercadoPagoPublicKey = "TEST-pub"
	if _, avisos := c.ValidarProducao(); len(avisos) != 1 {
		t.Errorf("chave pública de teste em produção: aviso, veio %v", avisos)
	}
}
