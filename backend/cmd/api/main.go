package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/config"
	"github.com/WilliamBreno/Arcadia/backend/internal/handler"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

func main() {
	cfg := config.Carregar()
	config.ConfigurarLogger(cfg)

	if cfg.AmbienteApp == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := repository.Conectar(cfg.DatabaseURL)
	if err != nil {
		slog.Error("erro ao conectar no banco", "erro", err)
		os.Exit(1)
	}

	usuarioRepo := repository.NovoUsuarioRepository(db)
	refreshTokenRepo := repository.NovoRefreshTokenRepository(db)

	jwtService := service.NovoJWTService(cfg.JWTSecret, cfg.AccessTokenTTLMin)
	authService := service.NovoAuthService(usuarioRepo, refreshTokenRepo, jwtService, cfg.RefreshTokenTTLDias)
	googleAuthService := service.NovoGoogleAuthService(cfg.GoogleClientID)
	mailCliente := mail.NovoCliente(cfg.ResendAPIKey, cfg.EmailRemetente)

	authHandler := handler.NovoAuthHandler(usuarioRepo, authService, googleAuthService, mailCliente, cfg)

	router := gin.New()
	router.Use(middleware.LogRequisicoes(), middleware.TratadorDeErros())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/healthz", handler.Healthz)

	limiteAuth := middleware.LimitarTaxaPorIP(1, 10)

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/cadastro", limiteAuth, authHandler.Cadastro)
		auth.POST("/login", limiteAuth, authHandler.Login)
		auth.POST("/google", limiteAuth, authHandler.Google)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/verificar-email", authHandler.VerificarEmail)
		auth.POST("/esqueci-senha", limiteAuth, authHandler.EsqueciSenha)
		auth.POST("/redefinir-senha", authHandler.RedefinirSenha)

		v1.GET("/me", middleware.ExigirAutenticacao(jwtService), authHandler.Me)
	}

	endereco := ":" + cfg.Porta
	slog.Info("iniciando servidor", "plataforma", cfg.NomePlataforma, "endereco", endereco, "ambiente", cfg.AmbienteApp)
	if err := router.Run(endereco); err != nil {
		slog.Error("erro ao iniciar servidor", "erro", err)
		os.Exit(1)
	}
}
