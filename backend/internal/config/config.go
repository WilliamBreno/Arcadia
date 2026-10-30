package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config concentra a configuração da aplicação, lida a partir de variáveis de ambiente.
type Config struct {
	Porta          string
	NomePlataforma string
	AmbienteApp    string
	DatabaseURL    string

	FrontendURL string

	JWTSecret           string
	AccessTokenTTLMin   int
	RefreshTokenTTLDias int

	GoogleClientID string

	ResendAPIKey   string
	EmailRemetente string

	// Uploads: "disco" (dev) ou "s3" (R2/S3/B2/Supabase — obrigatório em produção).
	StorageDriver  string
	UploadsDir     string
	UploadsBaseURL string
	S3Endpoint     string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3Regiao       string
	S3UsarSSL      bool

	BackendURL string

	MercadoPagoAccessToken   string
	MercadoPagoWebhookSecret string
	// MercadoPagoPublicKey é a chave pública do formulário de cartão na
	// página (item 7.1). Não é segredo: vai para o navegador.
	MercadoPagoPublicKey string
	// MercadoPagoBaseURL troca a API do MP por um servidor falso local
	// (só testes de desenvolvimento; proibido em produção).
	MercadoPagoBaseURL string
	// MercadoPagoPayoutChavePrivada é a chave Ed25519 (PEM) que assina o
	// envio de Pix ao organizador (item 9.1). Vazia em produção = repasse
	// automático desligado (segue manual).
	MercadoPagoPayoutChavePrivada string
	// AlertaWebhookURL recebe avisos de erro 500 (webhook do Discord ou
	// Slack, item 5.5). Opcional.
	AlertaWebhookURL string
	// DBMaxConexoes limita o pool de conexões com o Postgres (item 5.5).
	// Ajuste ao limite do plano do banco (o Neon gratuito aceita poucas).
	DBMaxConexoes int

	CronSecret string
}

func getEnv(chave, padrao string) string {
	if valor := os.Getenv(chave); valor != "" {
		return valor
	}
	return padrao
}

func getEnvInt(chave string, padrao int) int {
	if valor := os.Getenv(chave); valor != "" {
		if n, err := strconv.Atoi(valor); err == nil {
			return n
		}
	}
	return padrao
}

func getEnvBool(chave string, padrao bool) bool {
	if valor := os.Getenv(chave); valor != "" {
		if b, err := strconv.ParseBool(valor); err == nil {
			return b
		}
	}
	return padrao
}

const jwtSecretPadrao = "dev-secret-troque-em-producao"

// Carregar lê as variáveis de ambiente e retorna a configuração da aplicação.
func Carregar() Config {
	return Config{
		Porta:          getEnv("PORTA", getEnv("PORT", "8080")), // Render/Heroku injetam PORT
		NomePlataforma: getEnv("NOME_PLATAFORMA", "Evve"),
		AmbienteApp:    getEnv("AMBIENTE_APP", "development"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://arcadia:arcadia@localhost:5442/arcadia?sslmode=disable"),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		JWTSecret:           getEnv("JWT_SECRET", jwtSecretPadrao),
		AccessTokenTTLMin:   getEnvInt("ACCESS_TOKEN_TTL_MIN", 15),
		RefreshTokenTTLDias: getEnvInt("REFRESH_TOKEN_TTL_DIAS", 30),

		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),

		ResendAPIKey:   getEnv("RESEND_API_KEY", ""),
		EmailRemetente: getEnv("EMAIL_REMETENTE", "Evve <no-reply@evve.local>"),

		StorageDriver:  getEnv("STORAGE_DRIVER", "disco"),
		UploadsDir:     getEnv("UPLOADS_DIR", "./uploads"),
		UploadsBaseURL: getEnv("UPLOADS_BASE_URL", "http://localhost:8080/uploads"),
		S3Endpoint:     getEnv("S3_ENDPOINT", ""),
		S3Bucket:       getEnv("S3_BUCKET", ""),
		S3AccessKey:    getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:    getEnv("S3_SECRET_KEY", ""),
		S3Regiao:       getEnv("S3_REGION", "auto"),
		S3UsarSSL:      getEnvBool("S3_USE_SSL", true),

		BackendURL: getEnv("BACKEND_URL", "http://localhost:8080"),

		MercadoPagoAccessToken:        getEnv("MERCADOPAGO_ACCESS_TOKEN", ""),
		MercadoPagoWebhookSecret:      getEnv("MERCADOPAGO_WEBHOOK_SECRET", ""),
		MercadoPagoPublicKey:          getEnv("MERCADOPAGO_PUBLIC_KEY", ""),
		MercadoPagoBaseURL:            getEnv("MERCADOPAGO_BASE_URL", ""),
		MercadoPagoPayoutChavePrivada: getEnv("MERCADOPAGO_PAYOUT_CHAVE_PRIVADA", ""),
		AlertaWebhookURL:              getEnv("ALERTA_WEBHOOK_URL", ""),
		DBMaxConexoes:                 getEnvInt("DB_MAX_CONEXOES", 20),

		CronSecret: getEnv("CRON_SECRET", ""),
	}
}

// ValidarProducao recusa configuração insegura quando AMBIENTE_APP=production
// (erros → a API não sobe) e devolve avisos do que está faltando mas não
// impede o boot. Em outros ambientes não valida nada.
func (c Config) ValidarProducao() (erros []error, avisos []string) {
	if c.AmbienteApp != "production" {
		return nil, nil
	}
	if c.JWTSecret == jwtSecretPadrao || len(c.JWTSecret) < 32 {
		erros = append(erros, fmt.Errorf("JWT_SECRET precisa ter 32+ caracteres e não pode ser o valor de desenvolvimento"))
	}
	if len(c.CronSecret) < 16 {
		erros = append(erros, fmt.Errorf("CRON_SECRET precisa ter 16+ caracteres (protege /jobs/*)"))
	}
	if !strings.HasPrefix(c.FrontendURL, "https://") {
		erros = append(erros, fmt.Errorf("FRONTEND_URL precisa ser https:// (cookie de sessão Secure e CORS)"))
	}
	if !strings.HasPrefix(c.BackendURL, "https://") {
		erros = append(erros, fmt.Errorf("BACKEND_URL precisa ser https:// (o Mercado Pago só chama webhook https)"))
	}
	if strings.Contains(c.DatabaseURL, "localhost") {
		erros = append(erros, fmt.Errorf("DATABASE_URL aponta para localhost"))
	}
	if c.StorageDriver != "s3" {
		erros = append(erros, fmt.Errorf("STORAGE_DRIVER precisa ser s3 em produção (disco local some a cada deploy)"))
	}
	if c.MercadoPagoBaseURL != "" {
		erros = append(erros, fmt.Errorf("MERCADOPAGO_BASE_URL é só para testes locais e não pode estar definida em produção"))
	}
	if strings.HasPrefix(c.MercadoPagoAccessToken, "TEST-") {
		avisos = append(avisos, "MERCADOPAGO_ACCESS_TOKEN é de TESTE em produção: pagamentos não serão reais")
	}
	if c.MercadoPagoAccessToken == "" {
		avisos = append(avisos, "MERCADOPAGO_ACCESS_TOKEN vazio: checkout indisponível")
	}
	if c.MercadoPagoPublicKey == "" {
		avisos = append(avisos, "MERCADOPAGO_PUBLIC_KEY vazio: o cartão abre no site do Mercado Pago (o Pix continua na página)")
	} else if strings.HasPrefix(c.MercadoPagoPublicKey, "TEST-") {
		avisos = append(avisos, "MERCADOPAGO_PUBLIC_KEY é de TESTE em produção")
	}
	if c.MercadoPagoPayoutChavePrivada == "" {
		avisos = append(avisos, "MERCADOPAGO_PAYOUT_CHAVE_PRIVADA vazia: repasse automático por Pix desligado (segue manual)")
	}
	if c.MercadoPagoWebhookSecret == "" {
		avisos = append(avisos, "MERCADOPAGO_WEBHOOK_SECRET vazio: webhook sem validação de assinatura (segue seguro por reconsultar o MP, mas configure)")
	}
	if c.AlertaWebhookURL == "" {
		avisos = append(avisos, "ALERTA_WEBHOOK_URL vazio: erros 500 só aparecem no log (configure um webhook do Discord/Slack)")
	}
	if c.ResendAPIKey == "" {
		avisos = append(avisos, "RESEND_API_KEY vazio: e-mails (verificação, ingressos) só serão logados")
	}
	if strings.HasSuffix(c.EmailRemetente, ".local>") || strings.Contains(c.EmailRemetente, "evve.local") {
		avisos = append(avisos, "EMAIL_REMETENTE ainda é o de desenvolvimento: use um domínio verificado no Resend")
	}
	return erros, avisos
}
