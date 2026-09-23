package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type EventoHandler struct {
	organizadorHandler  *OrganizadorHandler
	service             *service.EventoService
	cancelamentoService *service.CancelamentoService
}

func NovoEventoHandler(organizadorHandler *OrganizadorHandler, s *service.EventoService, cancelamentoService *service.CancelamentoService) *EventoHandler {
	return &EventoHandler{organizadorHandler: organizadorHandler, service: s, cancelamentoService: cancelamentoService}
}

type eventoResposta struct {
	ID                        int64      `json:"id"`
	LocalID                   *int64     `json:"local_id"`
	Titulo                    string     `json:"titulo"`
	Slug                      string     `json:"slug"`
	Descricao                 string     `json:"descricao"`
	Categoria                 string     `json:"categoria"`
	CapaURL                   string     `json:"capa_url"`
	InicioEm                  *time.Time `json:"inicio_em"`
	FimEm                     *time.Time `json:"fim_em"`
	Timezone                  string     `json:"timezone"`
	ClassificacaoEtaria       string     `json:"classificacao_etaria"`
	Visibilidade              string     `json:"visibilidade"`
	TipoAcesso                string     `json:"tipo_acesso"`
	Status                    string     `json:"status"`
	ModoParticipantes         string     `json:"modo_participantes"`
	InscricaoTalentosInicio   *time.Time `json:"inscricao_talentos_inicio"`
	InscricaoTalentosFim      *time.Time `json:"inscricao_talentos_fim"`
	CapacidadeTotal           *int       `json:"capacidade_total"`
	GarantiaHabilitada        bool       `json:"garantia_habilitada"`
	PoliticaCancelamentoTexto string     `json:"politica_cancelamento_texto"`
	MaxItensPorPedido         int        `json:"max_itens_por_pedido"`
	PublicadoEm               *time.Time `json:"publicado_em"`
}

func paraEventoResposta(e *domain.Evento) eventoResposta {
	return eventoResposta{
		ID:                        e.ID,
		LocalID:                   e.LocalID,
		Titulo:                    e.Titulo,
		Slug:                      e.Slug,
		Descricao:                 e.Descricao,
		Categoria:                 e.Categoria,
		CapaURL:                   e.CapaURL,
		InicioEm:                  e.InicioEm,
		FimEm:                     e.FimEm,
		Timezone:                  e.Timezone,
		ClassificacaoEtaria:       e.ClassificacaoEtaria,
		Visibilidade:              string(e.Visibilidade),
		TipoAcesso:                string(e.TipoAcesso),
		Status:                    string(e.Status),
		ModoParticipantes:         string(e.ModoParticipantes),
		InscricaoTalentosInicio:   e.InscricaoTalentosInicio,
		InscricaoTalentosFim:      e.InscricaoTalentosFim,
		CapacidadeTotal:           e.CapacidadeTotal,
		GarantiaHabilitada:        e.GarantiaHabilitada,
		PoliticaCancelamentoTexto: e.PoliticaCancelamentoTexto,
		MaxItensPorPedido:         e.MaxItensPorPedido,
		PublicadoEm:               e.PublicadoEm,
	}
}

type eventoRequest struct {
	Titulo                    string     `json:"titulo" binding:"required,min=2"`
	LocalID                   *int64     `json:"local_id"`
	Descricao                 string     `json:"descricao"`
	Categoria                 string     `json:"categoria"`
	CapaURL                   string     `json:"capa_url"`
	InicioEm                  *time.Time `json:"inicio_em"`
	FimEm                     *time.Time `json:"fim_em"`
	Timezone                  string     `json:"timezone"`
	ClassificacaoEtaria       string     `json:"classificacao_etaria"`
	Visibilidade              string     `json:"visibilidade" binding:"omitempty,oneof=publico nao_listado privado"`
	TipoAcesso                string     `json:"tipo_acesso" binding:"omitempty,oneof=ingresso cadastro"`
	ModoParticipantes         string     `json:"modo_participantes" binding:"omitempty,oneof=nenhum convite inscricao_aberta ambos"`
	InscricaoTalentosInicio   *time.Time `json:"inscricao_talentos_inicio"`
	InscricaoTalentosFim      *time.Time `json:"inscricao_talentos_fim"`
	CapacidadeTotal           *int       `json:"capacidade_total"`
	GarantiaHabilitada        *bool      `json:"garantia_habilitada"`
	PoliticaCancelamentoTexto string     `json:"politica_cancelamento_texto"`
	MaxItensPorPedido         int        `json:"max_itens_por_pedido"`
}

func (req eventoRequest) paraDados() service.EventoDados {
	garantia := true
	if req.GarantiaHabilitada != nil {
		garantia = *req.GarantiaHabilitada
	}
	return service.EventoDados{
		Titulo:                    req.Titulo,
		LocalID:                   req.LocalID,
		Descricao:                 req.Descricao,
		Categoria:                 req.Categoria,
		CapaURL:                   req.CapaURL,
		InicioEm:                  req.InicioEm,
		FimEm:                     req.FimEm,
		Timezone:                  req.Timezone,
		ClassificacaoEtaria:       req.ClassificacaoEtaria,
		Visibilidade:              domain.Visibilidade(req.Visibilidade),
		TipoAcesso:                domain.TipoAcesso(req.TipoAcesso),
		ModoParticipantes:         domain.ModoParticipantes(req.ModoParticipantes),
		InscricaoTalentosInicio:   req.InscricaoTalentosInicio,
		InscricaoTalentosFim:      req.InscricaoTalentosFim,
		CapacidadeTotal:           req.CapacidadeTotal,
		GarantiaHabilitada:        garantia,
		PoliticaCancelamentoTexto: req.PoliticaCancelamentoTexto,
		MaxItensPorPedido:         req.MaxItensPorPedido,
	}
}

func (h *EventoHandler) Listar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}

	eventos, err := h.service.Listar(organizador.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar eventos"})
		return
	}

	resposta := make([]eventoResposta, 0, len(eventos))
	for i := range eventos {
		resposta = append(resposta, paraEventoResposta(&eventos[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

func (h *EventoHandler) Criar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}

	var req eventoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	evento, err := h.service.Criar(organizador.ID, req.paraDados())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao criar evento"})
		return
	}

	c.JSON(http.StatusCreated, paraEventoResposta(evento))
}

func (h *EventoHandler) Obter(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	evento, err := h.service.BuscarDoOrganizador(organizador.ID, eventoID)
	if err != nil {
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusOK, paraEventoResposta(evento))
}

func (h *EventoHandler) Atualizar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	var req eventoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	evento, err := h.service.Atualizar(organizador.ID, eventoID, req.paraDados())
	if err != nil {
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusOK, paraEventoResposta(evento))
}

func (h *EventoHandler) Publicar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	evento, problemas, err := h.service.Publicar(organizador.ID, eventoID)
	if err != nil {
		if errors.Is(err, service.ErrPublicacaoInvalida) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "evento não atende aos requisitos para publicação", "problemas": problemas})
			return
		}
		if errors.Is(err, service.ErrEventoNaoEhRascunho) {
			c.JSON(http.StatusConflict, gin.H{"erro": "evento não está em rascunho"})
			return
		}
		h.responderErro(c, err)
		return
	}

	c.JSON(http.StatusOK, paraEventoResposta(evento))
}

func (h *EventoHandler) responderErro(c *gin.Context, err error) {
	if errors.Is(err, service.ErrEventoNaoPertenceAoOrganizador) {
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
}
