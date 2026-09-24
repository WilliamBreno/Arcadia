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
	repasseRepo := repository.NovoRepasseRepository(db)
	cupomRepo := repository.NovoCupomRepository(db)
	solicitacaoRepo := repository.NovoSolicitacaoPlateiaRepository(db)
	avaliacaoRepo := repository.NovoAvaliacaoRepository(db)
	cronogramaRepo := repository.NovoCronogramaRepository(db)
	membroRepo := repository.NovoOrganizadorMembroRepository(db)
	afiliadoRepo := repository.NovoAfiliadoRepository(db)
	apiKeyRepo := repository.NovoAPIKeyRepository(db)

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
		db, eventoRepo, itemPedidoRepo, pedidoRepo, pagamentoRepo, lancamentoRepo, configPlataformaRepo, cupomRepo, tipoIngressoRepo,
		mpCliente, mailCliente, cfg.JWTSecret, cfg.FrontendURL, cfg.BackendURL, cfg.NomePlataforma,
	)
	cancelamentoService := service.NovoCancelamentoService(
		itemPedidoRepo, pedidoRepo, pagamentoRepo, reembolsoRepo, lancamentoRepo, eventoRepo, configPlataformaRepo,
		mpCliente, mailCliente, cfg.NomePlataforma,
	)
	contaService := service.NovoContaService(organizadorRepo, eventoRepo, papelEventoRepo, itemPedidoRepo, pedidoRepo)
	checkinService := service.NovoCheckinService(itemPedidoRepo, tipoIngressoRepo, eventoRepo, organizadorRepo, papelEventoRepo, cfg.JWTSecret)
	staffService := service.NovoStaffService(eventoRepo, usuarioRepo, papelEventoRepo)
	vendasService := service.NovoVendasService(eventoRepo, itemPedidoRepo)
	repasseService := service.NovoRepasseService(db, eventoRepo, itemPedidoRepo, repasseRepo, configPlataformaRepo)

	armazenamento, err := storage.NovoDiscoLocal(cfg.UploadsDir, cfg.UploadsBaseURL)
	if err != nil {
		slog.Error("erro ao preparar armazenamento de uploads", "erro", err)
		os.Exit(1)
	}

	authHandler := handler.NovoAuthHandler(usuarioRepo, authService, googleAuthService, mailCliente, cfg)
	organizadorHandler := handler.NovoOrganizadorHandler(organizadorRepo, organizadorService, membroRepo)
	localHandler := handler.NovoLocalHandler(organizadorHandler, localService)
	eventoHandler := handler.NovoEventoHandler(organizadorHandler, eventoService, cancelamentoService)
	tipoIngressoHandler := handler.NovoTipoIngressoHandler(organizadorHandler, tipoIngressoService)
	publicoHandler := handler.NovoPublicoHandler(eventoRepo, tipoIngressoRepo, localRepo, organizadorRepo, configPlataformaRepo, itemPedidoRepo, cronogramaRepo)
	cupomService := service.NovoCupomService(eventoService, cupomRepo)
	plateiaService := service.NovoPlateiaService(eventoRepo, eventoService, solicitacaoRepo, usuarioRepo, mailCliente, cfg.FrontendURL, cfg.NomePlataforma)
	apiPublicaHandler := handler.NovoAPIPublicaHandler(organizadorHandler, apiKeyRepo, eventoRepo, tipoIngressoRepo, itemPedidoRepo)
	checkoutService.DefinirPlateia(plateiaService)
	checkoutService.DefinirAfiliados(afiliadoRepo)
	afiliadoHandler := handler.NovoAfiliadoHandler(organizadorHandler, service.NovoAfiliadoService(eventoService, afiliadoRepo))
	cancelamentoService.DefinirPromotor(plateiaService)
	equipeHandler := handler.NovoEquipeHandler(organizadorHandler, membroRepo, usuarioRepo, itemPedidoRepo, mailCliente, cfg.NomePlataforma, cfg.FrontendURL)
	cronogramaService := service.NovoCronogramaService(eventoService, cronogramaRepo)
	cronogramaHandler := handler.NovoCronogramaHandler(organizadorHandler, cronogramaService, fichaService)
	avaliacaoService := service.NovoAvaliacaoService(eventoService, eventoRepo, fichaRepo, papelEventoRepo, avaliacaoRepo)
	avaliacaoHandler := handler.NovoAvaliacaoHandler(organizadorHandler, eventoRepo, avaliacaoService)
	plateiaHandler := handler.NovoPlateiaHandler(organizadorHandler, eventoRepo, plateiaService)
	transferenciaService := service.NovoTransferenciaService(db, itemPedidoRepo, pedidoRepo, eventoRepo, mailCliente, cfg.JWTSecret, cfg.FrontendURL, cfg.NomePlataforma)
	transferenciaHandler := handler.NovoTransferenciaHandler(transferenciaService)
	qrHandler := handler.NovoQRHandler(service.NovoQRService(itemPedidoRepo, pedidoRepo, tipoIngressoRepo, eventoRepo, cfg.JWTSecret))
	cortesiaService := service.NovoCortesiaService(db, eventoService, itemPedidoRepo, pedidoRepo, mailCliente, cfg.JWTSecret, cfg.FrontendURL, cfg.NomePlataforma)
	cortesiaService.DefinirPromotor(plateiaService)
	cortesiaHandler := handler.NovoCortesiaHandler(organizadorHandler, cortesiaService, fichaService, itemPedidoRepo, tipoIngressoRepo, eventoRepo)
	cupomHandler := handler.NovoCupomHandler(organizadorHandler, eventoRepo, cupomService)
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
	repasseHandler := handler.NovoRepasseHandler(organizadorHandler, repasseService)
	adminHandler := handler.NovoAdminHandler(reembolsoRepo, itemPedidoRepo, cancelamentoService)

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
		v1.GET("/eventos/:slug/resultado", avaliacaoHandler.ResultadoPublico)
		v1.GET("/ingressos/:codigo/:token", cortesiaHandler.IngressoPublico)
		v1.GET("/ingressos/:codigo/:token/qr", qrHandler.PorLink)
		v1.GET("/convites/:token", conviteHandler.Consultar)
		v1.POST("/webhooks/mercadopago", webhookHandler.MercadoPago)

		api := router.Group("/api/public/v1", middleware.LimitarTaxaPorIP(5, 20), middleware.ExigirAPIKey(apiKeyRepo))
		api.GET("/eventos", apiPublicaHandler.Eventos)
		api.GET("/eventos/:id", apiPublicaHandler.Evento)
		api.GET("/eventos/:id/ingressos", apiPublicaHandler.Ingressos)
		api.GET("/eventos/:id/participantes", apiPublicaHandler.Participantes)
		api.GET("/eventos/:id/resumo", apiPublicaHandler.Resumo)

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
		autenticado.GET("/me/ingressos/:id/qr", qrHandler.DoComprador)
		autenticado.POST("/convites/:token/aceitar", conviteHandler.Aceitar)
		autenticado.GET("/eventos/:slug/minha-ficha", fichaHandler.ObterMinha)
		autenticado.PUT("/eventos/:slug/minha-ficha", fichaHandler.AtualizarMinha)
		autenticado.POST("/eventos/:slug/inscricao", fichaHandler.Inscrever)
		autenticado.GET("/eventos/:slug/participantes", fichaHandler.ListarParaJurado)
		autenticado.GET("/eventos/:slug/participantes/:fichaId", fichaHandler.ObterParaJurado)
		autenticado.POST("/uploads", uploadHandler.Criar)
		autenticado.POST("/eventos/:slug/pedidos", pedidoHandler.Criar)
		autenticado.POST("/eventos/:slug/cupom", cupomHandler.Validar)
		autenticado.GET("/eventos/:slug/participantes/:fichaId/avaliacao", avaliacaoHandler.FormularioJurado)
		autenticado.PUT("/eventos/:slug/participantes/:fichaId/avaliacao", avaliacaoHandler.Avaliar)
		autenticado.GET("/eventos/:slug/solicitacao", plateiaHandler.Minha)
		autenticado.POST("/eventos/:slug/solicitacao", plateiaHandler.Solicitar)
		autenticado.GET("/pedidos/:id", pedidoHandler.Obter)
		autenticado.POST("/pedidos/:id/pagar", pedidoHandler.Pagar)
		autenticado.GET("/itens/:id/cancelamento", cancelamentoHandler.Simular)
		autenticado.POST("/itens/:id/cancelar", cancelamentoHandler.Cancelar)
		autenticado.POST("/itens/:id/transferir", transferenciaHandler.Transferir)
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

		org.GET("/eventos/:id/cortesias", cortesiaHandler.Listar)
		org.POST("/eventos/:id/cortesias", cortesiaHandler.Emitir)
		org.DELETE("/eventos/:id/cortesias/:itemId", cortesiaHandler.Revogar)
		org.GET("/eventos/:id/exportar.csv", cortesiaHandler.ExportarCSV)
		org.GET("/eventos/:id/afiliados", afiliadoHandler.Listar)
		org.POST("/eventos/:id/afiliados", afiliadoHandler.Criar)
		org.DELETE("/eventos/:id/afiliados/:afiliadoId", afiliadoHandler.Desativar)
		org.GET("/api-keys", apiPublicaHandler.ListarChaves)
		org.POST("/api-keys", apiPublicaHandler.CriarChave)
		org.DELETE("/api-keys/:id", apiPublicaHandler.RevogarChave)
		org.GET("/equipe", equipeHandler.Listar)
		org.POST("/equipe", equipeHandler.Adicionar)
		org.DELETE("/equipe/:usuarioId", equipeHandler.Remover)
		org.GET("/relatorios", equipeHandler.Relatorios)
		org.GET("/eventos/:id/cronograma", cronogramaHandler.Listar)
		org.POST("/eventos/:id/cronograma", cronogramaHandler.Criar)
		org.PUT("/eventos/:id/cronograma/:itemId", cronogramaHandler.Atualizar)
		org.DELETE("/eventos/:id/cronograma/:itemId", cronogramaHandler.Excluir)
		org.PUT("/eventos/:id/ordem-apresentacao", cronogramaHandler.DefinirOrdem)
		org.GET("/eventos/:id/criterios", avaliacaoHandler.ListarCriterios)
		org.POST("/eventos/:id/criterios", avaliacaoHandler.CriarCriterio)
		org.PUT("/eventos/:id/criterios/:criterioId", avaliacaoHandler.AtualizarCriterio)
		org.DELETE("/eventos/:id/criterios/:criterioId", avaliacaoHandler.ExcluirCriterio)
		org.GET("/eventos/:id/ranking", avaliacaoHandler.Ranking)
		org.POST("/eventos/:id/resultado", avaliacaoHandler.Liberar)
		org.GET("/eventos/:id/solicitacoes", plateiaHandler.Listar)
		org.POST("/eventos/:id/solicitacoes/:solicitacaoId", plateiaHandler.Decidir)
		org.GET("/eventos/:id/cupons", cupomHandler.Listar)
		org.POST("/eventos/:id/cupons", cupomHandler.Criar)
		org.DELETE("/eventos/:id/cupons/:cupomId", cupomHandler.Desativar)

		org.GET("/eventos/:id/convites", conviteHandler.Listar)
		org.POST("/eventos/:id/convites", conviteHandler.Gerar)
		org.DELETE("/eventos/:id/convites/:conviteId", conviteHandler.Revogar)

		org.GET("/eventos/:id/participantes", fichaHandler.ListarDoOrganizador)
		org.POST("/eventos/:id/participantes/:fichaId/aprovar", fichaHandler.Aprovar)
		org.POST("/eventos/:id/participantes/:fichaId/rejeitar", fichaHandler.Rejeitar)

		org.GET("/eventos/:id/vendas", vendasHandler.Listar)
		org.GET("/eventos/:id/financeiro", repasseHandler.FinanceiroEvento)
		org.GET("/repasses", repasseHandler.Extrato)

		org.GET("/eventos/:id/staff", staffHandler.Listar)
		org.POST("/eventos/:id/staff", staffHandler.Adicionar)
		org.DELETE("/eventos/:id/staff/:usuarioId", staffHandler.Remover)

		jobs := v1.Group("/jobs", middleware.ExigirCronSecret(cfg.CronSecret))
		jobs.POST("/expirar-reservas", jobHandler.ExpirarReservas)
		jobs.POST("/gerar-repasses", repasseHandler.GerarRepasses)
		jobs.POST("/reprocessar-reembolsos", adminHandler.ReprocessarFalhas)

		admin := v1.Group("/admin", middleware.ExigirAutenticacao(jwtService), middleware.ExigirAdminPlataforma())
		admin.GET("/repasses", repasseHandler.ListarAdmin)
		admin.POST("/repasses/:id/pagar", repasseHandler.Pagar)
		admin.GET("/relatorios", adminHandler.Relatorios)
		admin.GET("/reembolsos", adminHandler.ListarReembolsos)
		admin.POST("/reembolsos/:id/reprocessar", adminHandler.ReprocessarReembolso)
	}

	endereco := ":" + cfg.Porta
	slog.Info("iniciando servidor", "plataforma", cfg.NomePlataforma, "endereco", endereco, "ambiente", cfg.AmbienteApp)
	if err := router.Run(endereco); err != nil {
		slog.Error("erro ao iniciar servidor", "erro", err)
		os.Exit(1)
	}
}
