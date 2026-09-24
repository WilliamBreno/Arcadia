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
	"github.com/WilliamBreno/Arcadia/backend/internal/mercadopago"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
	"github.com/WilliamBreno/Arcadia/backend/internal/storage"
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
	conviteRepo := repository.NovoConviteRepository(db)
	papelEventoRepo := repository.NovoPapelEventoRepository(db)
	fichaRepo := repository.NovoFichaParticipacaoRepository(db)
	configPlataformaRepo := repository.NovoConfigPlataformaRepository(db)
	pedidoRepo := repository.NovoPedidoRepository(db)
	itemPedidoRepo := repository.NovoItemPedidoRepository(db)
	pagamentoRepo := repository.NovoPagamentoRepository(db)
	lancamentoRepo := repository.NovoLancamentoRepository(db)
	reembolsoRepo := repository.NovoReembolsoRepository(db)
	regraRegionalRepo := repository.NovoRegraRegionalRepository(db)

	jwtService := service.NovoJWTService(cfg.JWTSecret, cfg.AccessTokenTTLMin)
	authService := service.NovoAuthService(usuarioRepo, refreshTokenRepo, jwtService, cfg.RefreshTokenTTLDias)
	googleAuthService := service.NovoGoogleAuthService(cfg.GoogleClientID)
	mailCliente := mail.NovoCliente(cfg.ResendAPIKey, cfg.EmailRemetente)
	organizadorService := service.NovoOrganizadorService(organizadorRepo)
	localService := service.NovoLocalService(localRepo)
	eventoService := service.NovoEventoService(eventoRepo, tipoIngressoRepo, organizadorRepo, localRepo, regraRegionalRepo, configPlataformaRepo)
	tipoIngressoService := service.NovoTipoIngressoService(eventoService, tipoIngressoRepo)
	conviteService := service.NovoConviteService(conviteRepo, eventoService, papelEventoRepo, fichaRepo)
	fichaService := service.NovoFichaService(fichaRepo, papelEventoRepo, eventoRepo, usuarioRepo, mailCliente)
	mpCliente := mercadopago.NovoCliente(cfg.MercadoPagoAccessToken)
	checkoutService := service.NovoCheckoutService(
		db, eventoRepo, itemPedidoRepo, pedidoRepo, pagamentoRepo, lancamentoRepo, configPlataformaRepo,
		mpCliente, mailCliente, cfg.JWTSecret, cfg.FrontendURL, cfg.BackendURL, cfg.NomePlataforma,
	)
	cancelamentoService := service.NovoCancelamentoService(
		itemPedidoRepo, pedidoRepo, pagamentoRepo, reembolsoRepo, lancamentoRepo, eventoRepo, configPlataformaRepo,
		mpCliente, mailCliente, cfg.NomePlataforma,
	)
	contaService := service.NovoContaService(organizadorRepo, eventoRepo, papelEventoRepo, itemPedidoRepo, pedidoRepo)
	checkinService := service.NovoCheckinService(itemPedidoRepo, tipoIngressoRepo, eventoRepo, organizadorRepo, papelEventoRepo)
	staffService := service.NovoStaffService(eventoRepo, usuarioRepo, papelEventoRepo)
	vendasService := service.NovoVendasService(eventoRepo, itemPedidoRepo)

	armazenamento, err := storage.NovoDiscoLocal(cfg.UploadsDir, cfg.UploadsBaseURL)
	if err != nil {
		slog.Error("erro ao preparar armazenamento de uploads", "erro", err)
		os.Exit(1)
	}

	authHandler := handler.NovoAuthHandler(usuarioRepo, authService, googleAuthService, mailCliente, cfg)
	organizadorHandler := handler.NovoOrganizadorHandler(organizadorRepo, organizadorService)
	localHandler := handler.NovoLocalHandler(organizadorHandler, localService)
	eventoHandler := handler.NovoEventoHandler(organizadorHandler, eventoService, cancelamentoService)
	tipoIngressoHandler := handler.NovoTipoIngressoHandler(organizadorHandler, tipoIngressoService)
	publicoHandler := handler.NovoPublicoHandler(eventoRepo, tipoIngressoRepo, localRepo, organizadorRepo, configPlataformaRepo)
	conviteHandler := handler.NovoConviteHandler(organizadorHandler, eventoRepo, conviteService)
	fichaHandler := handler.NovoFichaHandler(eventoRepo, organizadorHandler, fichaService)
	uploadHandler := handler.NovoUploadHandler(armazenamento)
	pedidoHandler := handler.NovoPedidoHandler(eventoRepo, usuarioRepo, checkoutService)
	webhookHandler := handler.NovoWebhookHandler(checkoutService, cfg.MercadoPagoWebhookSecret)
	jobHandler := handler.NovoJobHandler(checkoutService)
	cancelamentoHandler := handler.NovoCancelamentoHandler(cancelamentoService)
	contaHandler := handler.NovoContaHandler(contaService)
	checkinHandler := handler.NovoCheckinHandler(checkinService)
	staffHandler := handler.NovoStaffHandler(organizadorHandler, staffService)
	vendasHandler := handler.NovoVendasHandler(organizadorHandler, vendasService)

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
	router.Static("/uploads", cfg.UploadsDir)

	limiteAuth := middleware.LimitarTaxaPorIP(1, 10)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/eventos", publicoHandler.ListarEventos)
		v1.GET("/eventos/:slug", publicoHandler.ObterEvento)
		v1.GET("/organizadores/:slug", publicoHandler.ObterOrganizador)
		v1.GET("/categorias", publicoHandler.Categorias)
		v1.GET("/convites/:token", conviteHandler.Consultar)
		v1.POST("/webhooks/mercadopago", webhookHandler.MercadoPago)

		auth := v1.Group("/auth")
		auth.POST("/cadastro", limiteAuth, authHandler.Cadastro)
		auth.POST("/login", limiteAuth, authHandler.Login)
		auth.POST("/google", limiteAuth, authHandler.Google)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/verificar-email", authHandler.VerificarEmail)
		auth.POST("/esqueci-senha", limiteAuth, authHandler.EsqueciSenha)
		auth.POST("/redefinir-senha", authHandler.RedefinirSenha)

		autenticado := v1.Group("", middleware.ExigirAutenticacao(jwtService))
		autenticado.GET("/me", authHandler.Me)
		autenticado.GET("/me/eventos", contaHandler.MeusEventos)
		autenticado.GET("/me/ingressos", contaHandler.MeusIngressos)
		autenticado.GET("/me/ingressos/:id", contaHandler.MeuIngresso)
		autenticado.POST("/convites/:token/aceitar", conviteHandler.Aceitar)
		autenticado.GET("/eventos/:slug/minha-ficha", fichaHandler.ObterMinha)
		autenticado.PUT("/eventos/:slug/minha-ficha", fichaHandler.AtualizarMinha)
		autenticado.POST("/eventos/:slug/inscricao", fichaHandler.Inscrever)
		autenticado.GET("/eventos/:slug/participantes", fichaHandler.ListarParaJurado)
		autenticado.GET("/eventos/:slug/participantes/:fichaId", fichaHandler.ObterParaJurado)
		autenticado.POST("/uploads", uploadHandler.Criar)
		autenticado.POST("/eventos/:slug/pedidos", pedidoHandler.Criar)
		autenticado.GET("/pedidos/:id", pedidoHandler.Obter)
		autenticado.POST("/pedidos/:id/pagar", pedidoHandler.Pagar)
		autenticado.GET("/itens/:id/cancelamento", cancelamentoHandler.Simular)
		autenticado.POST("/itens/:id/cancelar", cancelamentoHandler.Cancelar)
		autenticado.POST("/checkin/validar", checkinHandler.Validar)
		autenticado.GET("/checkin/eventos/:id/resumo", checkinHandler.Resumo)
		autenticado.GET("/checkin/eventos/:id/busca", checkinHandler.Buscar)

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
		org.POST("/eventos/:id/cancelar", eventoHandler.Cancelar)
		org.GET("/eventos/:id/ingressos", tipoIngressoHandler.Listar)
		org.POST("/eventos/:id/ingressos", tipoIngressoHandler.Criar)
		org.PUT("/eventos/:id/ingressos/:ingressoId", tipoIngressoHandler.Atualizar)
		org.DELETE("/eventos/:id/ingressos/:ingressoId", tipoIngressoHandler.Excluir)

		org.GET("/eventos/:id/convites", conviteHandler.Listar)
		org.POST("/eventos/:id/convites", conviteHandler.Gerar)
		org.DELETE("/eventos/:id/convites/:conviteId", conviteHandler.Revogar)

		org.GET("/eventos/:id/participantes", fichaHandler.ListarDoOrganizador)
		org.POST("/eventos/:id/participantes/:fichaId/aprovar", fichaHandler.Aprovar)
		org.POST("/eventos/:id/participantes/:fichaId/rejeitar", fichaHandler.Rejeitar)

		org.GET("/eventos/:id/vendas", vendasHandler.Listar)

		org.GET("/eventos/:id/staff", staffHandler.Listar)
		org.POST("/eventos/:id/staff", staffHandler.Adicionar)
		org.DELETE("/eventos/:id/staff/:usuarioId", staffHandler.Remover)

		jobs := v1.Group("/jobs", middleware.ExigirCronSecret(cfg.CronSecret))
		jobs.POST("/expirar-reservas", jobHandler.ExpirarReservas)
	}

	endereco := ":" + cfg.Porta
	slog.Info("iniciando servidor", "plataforma", cfg.NomePlataforma, "endereco", endereco, "ambiente", cfg.AmbienteApp)
	if err := router.Run(endereco); err != nil {
		slog.Error("erro ao iniciar servidor", "erro", err)
		os.Exit(1)
	}
}
