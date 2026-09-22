package config

import "os"

// Config concentra a configuração da aplicação, lida a partir de variáveis de ambiente.
type Config struct {
	Porta          string
	NomePlataforma string
	AmbienteApp    string
}

func getEnv(chave, padrao string) string {
	if valor := os.Getenv(chave); valor != "" {
		return valor
	}
	return padrao
}

// Carregar lê as variáveis de ambiente e retorna a configuração da aplicação.
func Carregar() Config {
	return Config{
		Porta:          getEnv("PORTA", "8080"),
		NomePlataforma: getEnv("NOME_PLATAFORMA", "Arcadia"),
		AmbienteApp:    getEnv("AMBIENTE_APP", "development"),
	}
}
