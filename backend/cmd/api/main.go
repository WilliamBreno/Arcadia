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
	organizadorRepo := repository.NovoOrganizadorRepository(db)
	localRepo := repository.NovoLocalRepository(db)
	eventoRepo := repository.NovoEventoRepository(db)
	tipoIngressoRepo := repository.NovoTipoIngressoRepository(db)

	jwtService := service.NovoJWTService(cfg.JWTSecret, cfg.AccessTokenTTLMin)
	authService := service.NovoAuthService(usuarioRepo, refreshTokenRepo, jwtService, cfg.RefreshTokenTTLDias)
	googleAuthService := service.NovoGoogleAuthService(cfg.GoogleClientID)
	mailCliente := mail.NovoCliente(cfg.ResendAPIKey, cfg.EmailRemetente)
	organizadorService := service.NovoOrganizadorService(organizadorRepo)
	localService := service.NovoLocalService(localRepo)
	eventoService := service.NovoEventoService(eventoRepo, tipoIngressoRepo, organizadorRepo)
	tipoIngressoService := service.NovoTipoIngressoService(eventoService, tipoIngressoRepo)

	authHandler := handler.NovoAuthHandler(usuarioRepo, authService, googleAuthService, mailCliente, cfg)
	organizadorHandler := handler.NovoOrganizadorHandler(organizadorRepo, organizadorService)
	localHandler := handler.NovoLocalHandler(organizadorHandler, localService)
	eventoHandler := handler.NovoEventoHandler(organizadorHandler, eventoService)
	tipoIngressoHandler := handler.NovoTipoIngressoHandler(organizadorHandler, tipoIngressoService)
	publicoHandler := handler.NovoPublicoHandler(eventoRepo, tipoIngressoRepo, localRepo, organizadorRepo)

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
		v1.GET("/eventos", publicoHandler.ListarEventos)
		v1.GET("/eventos/:slug", publicoHandler.ObterEvento)
		v1.GET("/organizadores/:slug", publicoHandler.ObterOrganizador)
		v1.GET("/categorias", publicoHandler.Categorias)

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

		org := v1.Group("/org", middleware.ExigirAutenticacao(jwtService))
		org.POST("/perfil", organizadorHandler.CriarPerfil)
		org.GET("/perfil", organizadorHandler.MeuPerfil)
		org.PUT("/perfil", organizadorHandler.AtualizarPerfil)
		org.GET("/locais", localHandler.Listar)
		org.POST("/locais", localHandler.Criar)
		org.PUT("/locais/:id", localHandler.Atualizar)
		org.DELETE("/locais/:id", localHandler.Excluir)

		org.GET("/eventos", eventoHandler.Listar)
		org.POST("/eventos", eventoHandler.Criar)
		org.GET("/eventos/:id", eventoHandler.Obter)
		org.PUT("/eventos/:id", eventoHandler.Atualizar)
		org.POST("/eventos/:id/publicar", eventoHandler.Publicar)
		org.GET("/eventos/:id/ingressos", tipoIngressoHandler.Listar)
		org.POST("/eventos/:id/ingressos", tipoIngressoHandler.Criar)
		org.PUT("/eventos/:id/ingressos/:ingressoId", tipoIngressoHandler.Atualizar)
		org.DELETE("/eventos/:id/ingressos/:ingressoId", tipoIngressoHandler.Excluir)
	}

	endereco := ":" + cfg.Porta
	slog.Info("iniciando servidor", "plataforma", cfg.NomePlataforma, "endereco", endereco, "ambiente", cfg.AmbienteApp)
	if err := router.Run(endereco); err != nil {
		slog.Error("erro ao iniciar servidor", "erro", err)
		os.Exit(1)
	}
}
