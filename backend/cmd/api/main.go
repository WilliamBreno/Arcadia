package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
	_ "time/tzdata" // fusos horários embutidos (imagem Docker mínima não tem)

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
	erros, avisos := cfg.ValidarProducao()
	for _, aviso := range avisos {
		slog.Warn("configuração de produção", "aviso", aviso)
	}
	if len(erros) > 0 {
		for _, e := range erros {
			slog.Error("configuração de produção inválida", "erro", e)
		}
		os.Exit(1)
	}

	db, err := repository.Conectar(cfg.DatabaseURL, cfg.DBMaxConexoes)
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
	sessaoRepo := repository.NovoSessaoRepository(db)
	apiKeyRepo := repository.NovoAPIKeyRepository(db)

	jwtService := service.NovoJWTService(cfg.JWTSecret, cfg.AccessTokenTTLMin)
	authService := service.NovoAuthService(usuarioRepo, refreshTokenRepo, jwtService, cfg.RefreshTokenTTLDias)
	googleAuthService := service.NovoGoogleAuthService(cfg.GoogleClientID)
	// Item 5.5: e-mails vão para uma fila no banco e são enviados em segundo
	// plano, com novas tentativas.
	mailCliente := mail.NovoCliente(cfg.ResendAPIKey, cfg.EmailRemetente).UsarFila(db)
	mailCliente.IniciarProcessador(context.Background(), 5*time.Second)
	organizadorService := service.NovoOrganizadorService(organizadorRepo)
	localService := service.NovoLocalService(localRepo)
	eventoService := service.NovoEventoService(eventoRepo, tipoIngressoRepo, organizadorRepo, localRepo, regraRegionalRepo, configPlataformaRepo)
	tipoIngressoService := service.NovoTipoIngressoService(eventoService, tipoIngressoRepo, sessaoRepo)
	conviteService := service.NovoConviteService(conviteRepo, eventoService, papelEventoRepo, fichaRepo)
	fichaService := service.NovoFichaService(fichaRepo, papelEventoRepo, eventoRepo, usuarioRepo, mailCliente)
	// Item 5.2: os serviços falam com pagamento.Processador; o Mercado Pago
	// é a implementação atual.
	mpCliente := mercadopago.NovoProcessador(mercadopago.NovoCliente(cfg.MercadoPagoAccessToken).ComBaseURL(cfg.MercadoPagoBaseURL))
	checkoutService := service.NovoCheckoutService(
		db, eventoRepo, itemPedidoRepo, pedidoRepo, pagamentoRepo, lancamentoRepo, configPlataformaRepo, cupomRepo, tipoIngressoRepo,
		mpCliente, mailCliente, cfg.JWTSecret, cfg.FrontendURL, cfg.BackendURL, cfg.NomePlataforma,
	)
	cancelamentoService := service.NovoCancelamentoService(
		itemPedidoRepo, pedidoRepo, pagamentoRepo, reembolsoRepo, lancamentoRepo, eventoRepo, configPlataformaRepo,
		mpCliente, mailCliente, cfg.NomePlataforma,
	)
	contaService := service.NovoContaService(organizadorRepo, eventoRepo, papelEventoRepo, itemPedidoRepo, pedidoRepo)
	checkinService := service.NovoCheckinService(itemPedidoRepo, tipoIngressoRepo, eventoRepo, organizadorRepo, papelEventoRepo, cfg.JWTSecret, sessaoRepo)
	staffService := service.NovoStaffService(eventoRepo, usuarioRepo, papelEventoRepo)
	vendasService := service.NovoVendasService(eventoRepo, itemPedidoRepo)
	repasseService := service.NovoRepasseService(db, eventoRepo, itemPedidoRepo, repasseRepo, configPlataformaRepo)
	repasseService.DefinirPagamentos(pagamentoRepo)

	var armazenamento storage.Armazenamento
	if cfg.StorageDriver == "s3" {
		armazenamento, err = storage.NovoS3(storage.ConfigS3{
			Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey,
			Regiao: cfg.S3Regiao, BaseURL: cfg.UploadsBaseURL, UsarSSL: cfg.S3UsarSSL,
		})
	} else {
		armazenamento, err = storage.NovoDiscoLocal(cfg.UploadsDir, cfg.UploadsBaseURL)
	}
	if err != nil {
		slog.Error("erro ao preparar armazenamento de uploads", "erro", err)
		os.Exit(1)
	}

	authHandler := handler.NovoAuthHandler(usuarioRepo, authService, googleAuthService, mailCliente, cfg)
	organizadorHandler := handler.NovoOrganizadorHandler(organizadorRepo, organizadorService, membroRepo)
	localHandler := handler.NovoLocalHandler(organizadorHandler, localService)
	eventoHandler := handler.NovoEventoHandler(organizadorHandler, eventoService, cancelamentoService)
	tipoIngressoHandler := handler.NovoTipoIngressoHandler(organizadorHandler, tipoIngressoService)
	publicoHandler := handler.NovoPublicoHandler(eventoRepo, tipoIngressoRepo, localRepo, organizadorRepo, configPlataformaRepo, itemPedidoRepo, cronogramaRepo, sessaoRepo)
	cupomService := service.NovoCupomService(eventoService, cupomRepo)
	plateiaService := service.NovoPlateiaService(eventoRepo, eventoService, solicitacaoRepo, usuarioRepo, mailCliente, cfg.FrontendURL, cfg.NomePlataforma)
	apiPublicaHandler := handler.NovoAPIPublicaHandler(organizadorHandler, apiKeyRepo, eventoRepo, tipoIngressoRepo, itemPedidoRepo)
	sessaoHandler := handler.NovoSessaoHandler(organizadorHandler, service.NovoSessaoService(eventoService, eventoRepo, sessaoRepo, cancelamentoService, mailCliente, cfg.NomePlataforma))
	checkoutService.DefinirPlateia(plateiaService)
	// Item 7.3: lista de espera de ingresso esgotado.
	filaService := service.NovoFilaEsperaService(db, itemPedidoRepo, mailCliente, cfg.FrontendURL, cfg.NomePlataforma)
	checkoutService.DefinirFila(filaService)
	promotores := service.Promotores{plateiaService, filaService}
	// Item 5.1: taxa por faixa, garantia percentual e taxa absorvida.
	taxasService := service.NovoTaxasService(configPlataformaRepo, localRepo, regraRegionalRepo)
	checkoutService.DefinirTaxas(taxasService)
	publicoHandler.DefinirTaxas(taxasService)
	checkoutService.DefinirAfiliados(afiliadoRepo)
	afiliadoHandler := handler.NovoAfiliadoHandler(organizadorHandler, service.NovoAfiliadoService(eventoService, afiliadoRepo))
	cancelamentoService.DefinirPromotor(promotores)
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
	cortesiaService.DefinirPromotor(promotores)
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
	// Item 5.3: limite por conta, bloqueio de cartão e contestações.
	contestacaoRepo := repository.NovoContestacaoRepository(db)
	checkoutService.DefinirAntifraude(usuarioRepo, contestacaoRepo)
	adminHandler.DefinirContestacoes(contestacaoRepo)
	// Item 5.4: verificação do organizador e reserva de risco.
	adminHandler.DefinirOrganizadores(organizadorRepo)
	repasseService.DefinirOrganizadores(organizadorRepo)

	// Item 5.5: venda bloqueada depois do fim do evento ou do dia do ingresso.
	checkoutService.DefinirSessoes(sessaoRepo)
	adminHandler.DefinirMail(mailCliente)
	jobHandler.DefinirMail(mailCliente)
	// Item 5.5: limite de tentativas no Postgres (vale para todas as
	// instâncias) e alertas de erro 500.
	limiteRepo := repository.NovoLimiteTaxaRepository(db)
	authHandler.DefinirLimite(limiteRepo)
	authHandler.DefinirCodigo(service.NovoCodigoLoginService(db, mailCliente, cfg.JWTSecret, cfg.NomePlataforma))
	alertador := middleware.NovoAlertador(cfg.AlertaWebhookURL)

	// Item 6.4: seguir, aviso de evento novo, lembretes e calendário.
	seguirService := service.NovoSeguirService(db, organizadorRepo, localRepo, mailCliente, cfg.JWTSecret, cfg.FrontendURL, cfg.NomePlataforma)
	eventoService.DefinirAoPublicar(seguirService.NotificarPublicacao)
	publicoHandler.DefinirSeguir(seguirService)
	seguirHandler := handler.NovoSeguirHandler(seguirService, service.NovoLembretesService(db, localRepo, mailCliente, cfg.FrontendURL), eventoRepo, localRepo, cfg.FrontendURL)

	// Item 6.5: avaliação do evento por quem fez check-in.
	avaliacaoPublicoService := service.NovoAvaliacaoPublicoService(db, mailCliente, cfg.FrontendURL)
	avaliacaoPublicoHandler := handler.NovoAvaliacaoPublicoHandler(avaliacaoPublicoService, eventoRepo)
	publicoHandler.DefinirAvaliacoes(avaliacaoPublicoService)
	seguirHandler.DefinirConvites(avaliacaoPublicoService)
	// Item 7.2: e-mail de carrinho para quem reservou e não pagou.
	jobHandler.DefinirFila(filaService)
	// Item 7.4: revenda oficial ligada à lista de espera.
	revendaService := service.NovoRevendaService(db, itemPedidoRepo, cancelamentoService, filaService)
	checkoutService.DefinirRevenda(revendaService)
	cancelamentoService.DefinirRevenda(db)
	checkinService.DefinirRevenda(db)
	jobHandler.DefinirRevenda(revendaService)
	revendaHandler := handler.NovoRevendaHandler(revendaService)
	pixelsHandler := handler.NovoPixelsHandler(db, eventoRepo, organizadorHandler) // item 7.5
	perguntasService := service.NovoPerguntasService(db)                           // item 7.6
	// Item 8.1: perfil público do talento.
	talentoService := service.NovoPerfilTalentoService(db, avaliacaoService, mailCliente, cfg.FrontendURL, cfg.NomePlataforma)
	talentoHandler := handler.NovoTalentoHandler(talentoService)
	avaliacaoHandler.DefinirPerfis(talentoService, db)
	// Item 8.2: voto do público e devolutiva dos jurados.
	concursoService := service.NovoConcursoPublicoService(db)
	avaliacaoHandler.DefinirConcurso(concursoService)
	concursoHandler := handler.NovoConcursoPublicoHandler(concursoService, avaliacaoService, talentoService, eventoRepo, organizadorHandler)
	// Item 8.3: modo palco/telão e camarim.
	palcoService := service.NovoPalcoService(db, cfg.JWTSecret, avaliacaoService, concursoService, talentoService)
	palcoHandler := handler.NovoPalcoHandler(palcoService, checkinService, eventoRepo, organizadorHandler)
	fichaService.DefinirFrontendURL(cfg.FrontendURL)
	perguntasHandler := handler.NovoPerguntasHandler(perguntasService, eventoRepo, organizadorHandler)
	jobHandler.DefinirPerguntas(perguntasService)
	publicoHandler.DefinirPerguntas(db)
	filaHandler := handler.NovoFilaEsperaHandler(filaService, eventoRepo, organizadorHandler)
	jobHandler.DefinirRecuperacao(service.NovoRecuperacaoCarrinhoService(db, mailCliente, seguirService, cfg.FrontendURL, cfg.NomePlataforma))

	router := gin.New()
	router.Use(middleware.IDRequisicao(), middleware.LogRequisicoes(), middleware.AlertarErros(alertador), middleware.TratadorDeErros(), middleware.CabecalhosSeguranca())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Organizador-ID", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/healthz", handler.Healthz)
	if cfg.StorageDriver != "s3" {
		router.Static("/uploads", cfg.UploadsDir)
	}

	limiteAuth := middleware.LimitarPorJanela(limiteRepo, "auth", 60, time.Minute)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/eventos", publicoHandler.ListarEventos)
		v1.GET("/eventos/:slug", publicoHandler.ObterEvento)
		v1.GET("/organizadores/:slug", publicoHandler.ObterOrganizador)
		v1.GET("/categorias", publicoHandler.Categorias)
		// Item 6.2: descoberta (cidades, categorias, destaques e coleções).
		v1.GET("/descoberta", publicoHandler.Descoberta)
		v1.GET("/taxas", publicoHandler.Taxas) // item 6.3: página "Quanto custa"
		v1.GET("/eventos/:slug/calendario.ics", seguirHandler.Calendario)
		v1.POST("/descadastrar", seguirHandler.Descadastrar)
		v1.GET("/destaques", publicoHandler.Destaques)
		v1.GET("/colecoes", publicoHandler.Colecoes)
		v1.GET("/eventos/:slug/avaliacoes", avaliacaoPublicoHandler.DoEvento)
		v1.GET("/talentos/autorizacao", talentoHandler.VerAutorizacao)
		v1.POST("/talentos/autorizacao", talentoHandler.Decidir)
		v1.GET("/talentos/:slug", talentoHandler.Publico)
		v1.GET("/eventos/:slug/votacao", concursoHandler.Votacao)
		v1.GET("/telao/:token", palcoHandler.Telao)
		// Item 7.1: chave pública do formulário de cartão (vazia = plano B).
		v1.GET("/pagamento/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"public_key": cfg.MercadoPagoPublicKey})
		})
		v1.GET("/colecoes/:slug", publicoHandler.Colecao)
		// Item 6.1: sitemap e robots (a Vercel reescreve /sitemap.xml e /robots.txt para cá).
		seoHandler := handler.NovoSEOHandler(eventoRepo, cfg.FrontendURL)
		seoHandler.DefinirTalentos(talentoService.ParaSitemap)
		v1.GET("/seo/sitemap.xml", seoHandler.Sitemap)
		v1.GET("/seo/robots.txt", seoHandler.Robots)
		v1.GET("/eventos/:slug/resultado", avaliacaoHandler.ResultadoPublico)
		v1.GET("/ingressos/:codigo/:token", cortesiaHandler.IngressoPublico)
		v1.GET("/ingressos/:codigo/:token/qr", qrHandler.PorLink)
		v1.GET("/convites/:token", conviteHandler.Consultar)
		v1.POST("/webhooks/mercadopago", webhookHandler.MercadoPago)

		api := router.Group("/api/public/v1", middleware.LimitarPorJanela(limiteRepo, "api", 300, time.Minute), middleware.ExigirAPIKey(apiKeyRepo))
		api.GET("/eventos", apiPublicaHandler.Eventos)
		api.GET("/eventos/:id", apiPublicaHandler.Evento)
		api.GET("/eventos/:id/ingressos", apiPublicaHandler.Ingressos)
		api.GET("/eventos/:id/participantes", apiPublicaHandler.Participantes)
		api.GET("/eventos/:id/resumo", apiPublicaHandler.Resumo)

		auth := v1.Group("/auth")
		auth.POST("/cadastro", limiteAuth, authHandler.Cadastro)
		auth.POST("/login", limiteAuth, authHandler.Login)
		auth.POST("/codigo", limiteAuth, authHandler.SolicitarCodigo)
		auth.POST("/codigo/verificar", limiteAuth, authHandler.VerificarCodigo)
		auth.POST("/google", limiteAuth, authHandler.Google)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/verificar-email", authHandler.VerificarEmail)
		auth.POST("/esqueci-senha", limiteAuth, authHandler.EsqueciSenha)
		auth.POST("/redefinir-senha", authHandler.RedefinirSenha)

		autenticado := v1.Group("", middleware.ExigirAutenticacao(jwtService))
		autenticado.GET("/me", authHandler.Me)
		autenticado.GET("/me/eventos", contaHandler.MeusEventos)
		autenticado.GET("/me/seguindo", seguirHandler.Meus)
		autenticado.GET("/eventos/:slug/fila", filaHandler.Minhas)
		autenticado.GET("/me/revendas", revendaHandler.Minhas)
		autenticado.GET("/me/talento", talentoHandler.Meu)
		autenticado.GET("/eventos/:slug/meus-votos", concursoHandler.MeusVotos)
		autenticado.GET("/eventos/:slug/meu-qr-participante", palcoHandler.MeuQR)
		autenticado.GET("/palco/eventos/:id", palcoHandler.Estado)
		autenticado.POST("/palco/eventos/:id/fichas/:ficha/status", palcoHandler.Status)
		autenticado.POST("/palco/eventos/:id/camarim/qr", palcoHandler.QR)
		autenticado.POST("/palco/eventos/:id/chamar", palcoHandler.Chamar)
		autenticado.POST("/palco/eventos/:id/proximo", palcoHandler.Proximo)
		autenticado.POST("/palco/eventos/:id/encerrar", palcoHandler.Encerrar)
		autenticado.POST("/palco/eventos/:id/modo", palcoHandler.Modo)
		autenticado.POST("/eventos/:slug/votos", concursoHandler.Votar)
		autenticado.GET("/eventos/:slug/minha-devolutiva", concursoHandler.MinhaDevolutiva)
		autenticado.PUT("/me/talento", talentoHandler.Salvar)
		autenticado.POST("/me/talento/participacoes/:ficha", talentoHandler.Ocultar)
		autenticado.POST("/itens/:id/revenda", revendaHandler.Anunciar)
		autenticado.DELETE("/itens/:id/revenda", revendaHandler.Retirar)
		autenticado.POST("/eventos/:slug/fila", filaHandler.Entrar)
		autenticado.DELETE("/eventos/:slug/fila/:tipo", filaHandler.Sair)
		autenticado.GET("/eventos/:slug/minha-avaliacao", avaliacaoPublicoHandler.Minha)
		autenticado.PUT("/eventos/:slug/minha-avaliacao", avaliacaoPublicoHandler.Salvar)
		autenticado.POST("/seguindo", seguirHandler.Seguir)
		autenticado.DELETE("/seguindo", seguirHandler.Deixar)
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
		autenticado.POST("/pedidos/:id/pix", pedidoHandler.Pix)
		autenticado.POST("/pedidos/:id/cartao", pedidoHandler.Cartao)
		autenticado.POST("/pedidos/:id/verificar", pedidoHandler.Verificar)
		autenticado.GET("/itens/:id/cancelamento", cancelamentoHandler.Simular)
		autenticado.POST("/itens/:id/cancelar", cancelamentoHandler.Cancelar)
		autenticado.POST("/itens/:id/transferir", transferenciaHandler.Transferir)
		autenticado.POST("/checkin/validar", checkinHandler.Validar)
		autenticado.POST("/checkin/sincronizar", checkinHandler.Sincronizar)
		autenticado.GET("/checkin/eventos/:id/pacote", checkinHandler.Pacote)
		autenticado.GET("/checkin/eventos/:id/conflitos", checkinHandler.Conflitos)
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
		org.POST("/eventos/:id/ingressos/por-sessao", tipoIngressoHandler.CriarPorSessao)
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
		org.GET("/eventos/:id/sessoes", sessaoHandler.Listar)
		org.POST("/eventos/:id/sessoes", sessaoHandler.Criar)
		org.POST("/eventos/:id/sessoes/lote", sessaoHandler.CriarLote)
		org.PUT("/eventos/:id/sessoes/:sessaoId", sessaoHandler.Atualizar)
		org.DELETE("/eventos/:id/sessoes/:sessaoId", sessaoHandler.Excluir)
		org.POST("/eventos/:id/sessoes/:sessaoId/cancelar", sessaoHandler.Cancelar)
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
		org.GET("/eventos/:id/fila", filaHandler.Resumo)
		org.GET("/eventos/:id/pixels", pixelsHandler.Obter)
		org.PUT("/eventos/:id/pixels", pixelsHandler.Salvar)
		org.GET("/eventos/:id/perguntas", perguntasHandler.Listar)
		org.GET("/eventos/:id/votacao", concursoHandler.Situacao)
		org.POST("/eventos/:id/telao/link", palcoHandler.LinkTelao)
		org.POST("/eventos/:id/votacao", concursoHandler.DefinirVotacao)
		org.POST("/eventos/:id/devolutiva", concursoHandler.DefinirDevolutiva)
		org.POST("/eventos/:id/perguntas", perguntasHandler.Salvar)
		org.PUT("/eventos/:id/perguntas/:pergunta", perguntasHandler.Salvar)
		org.DELETE("/eventos/:id/perguntas/:pergunta", perguntasHandler.Remover)
		org.GET("/eventos/:id/respostas.csv", perguntasHandler.Exportar)
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
		jobs.POST("/enviar-emails", jobHandler.EnviarEmails)
		jobs.POST("/lembretes", seguirHandler.Lembretes)

		admin := v1.Group("/admin", middleware.ExigirAutenticacao(jwtService), middleware.ExigirAdminPlataforma())
		admin.GET("/repasses", repasseHandler.ListarAdmin)
		admin.POST("/repasses/:id/pagar", repasseHandler.Pagar)
		admin.POST("/repasses/:id/reserva/pagar", repasseHandler.PagarReserva)
		admin.POST("/repasses/:id/reserva/usar", repasseHandler.UsarReserva)
		admin.GET("/organizadores", adminHandler.ListarOrganizadores)
		admin.POST("/organizadores/:id/verificacao", adminHandler.DefinirVerificacao)
		admin.GET("/relatorios", adminHandler.Relatorios)
		admin.GET("/reembolsos", adminHandler.ListarReembolsos)
		admin.GET("/contestacoes", adminHandler.ListarContestacoes)
		admin.GET("/colecoes", publicoHandler.ListarColecoesAdmin)
		admin.GET("/avaliacoes", avaliacaoPublicoHandler.ListarAdmin)
		admin.POST("/avaliacoes/:id/ocultar", avaliacaoPublicoHandler.Ocultar)
		admin.POST("/colecoes", publicoHandler.SalvarColecao)
		admin.PUT("/colecoes/:id", publicoHandler.SalvarColecao)
		admin.DELETE("/colecoes/:id", publicoHandler.ExcluirColecao)
		admin.POST("/reembolsos/:id/reprocessar", adminHandler.ReprocessarReembolso)
	}

	endereco := ":" + cfg.Porta
	slog.Info("iniciando servidor", "plataforma", cfg.NomePlataforma, "endereco", endereco, "ambiente", cfg.AmbienteApp)
	if err := router.Run(endereco); err != nil {
		slog.Error("erro ao iniciar servidor", "erro", err)
		os.Exit(1)
	}
}
