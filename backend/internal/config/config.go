package config

import (
	"os"
	"strconv"
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

	UploadsDir     string
	UploadsBaseURL string

	BackendURL string

	MercadoPagoAccessToken   string
	MercadoPagoWebhookSecret string

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

// Carregar lê as variáveis de ambiente e retorna a configuração da aplicação.
func Carregar() Config {
	return Config{
		Porta:          getEnv("PORTA", "8080"),
		NomePlataforma: getEnv("NOME_PLATAFORMA", "Arcadia"),
		AmbienteApp:    getEnv("AMBIENTE_APP", "development"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://arcadia:arcadia@localhost:5442/arcadia?sslmode=disable"),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		JWTSecret:           getEnv("JWT_SECRET", "dev-secret-troque-em-producao"),
		AccessTokenTTLMin:   getEnvInt("ACCESS_TOKEN_TTL_MIN", 15),
		RefreshTokenTTLDias: getEnvInt("REFRESH_TOKEN_TTL_DIAS", 30),

		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),

		ResendAPIKey:   getEnv("RESEND_API_KEY", ""),
		EmailRemetente: getEnv("EMAIL_REMETENTE", "Arcadia <no-reply@arcadia.local>"),

		UploadsDir:     getEnv("UPLOADS_DIR", "./uploads"),
		UploadsBaseURL: getEnv("UPLOADS_BASE_URL", "http://localhost:8080/uploads"),

		BackendURL: getEnv("BACKEND_URL", "http://localhost:8080"),

		MercadoPagoAccessToken:   getEnv("MERCADOPAGO_ACCESS_TOKEN", ""),
		MercadoPagoWebhookSecret: getEnv("MERCADOPAGO_WEBHOOK_SECRET", ""),

		CronSecret: getEnv("CRON_SECRET", ""),
	}
}
