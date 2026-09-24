package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// APIPublicaHandler serve a API de integrações (somente leitura, escopo do
// organizador dono da chave) e o gerenciamento das chaves (só do dono).
type APIPublicaHandler struct {
	organizadorHandler *OrganizadorHandler
	chaves             *repository.APIKeyRepository
	eventos            *repository.EventoRepository
	tipos              *repository.TipoIngressoRepository
	itens              *repository.ItemPedidoRepository
}

func NovoAPIPublicaHandler(o *OrganizadorHandler, c *repository.APIKeyRepository, e *repository.EventoRepository,
	t *repository.TipoIngressoRepository, i *repository.ItemPedidoRepository) *APIPublicaHandler {
	return &APIPublicaHandler{organizadorHandler: o, chaves: c, eventos: e, tipos: t, itens: i}
}

// --- gestão de chaves (JWT, só dono) ---

type criarChaveRequest struct {
	Nome string `json:"nome" binding:"required"`
}

// CriarChave é POST /org/api-keys: a chave completa aparece UMA vez.
func (h *APIPublicaHandler) CriarChave(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	var req criarChaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe um nome para a chave"})
		return
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar chave"})
		return
	}
	chave := "arc_" + hex.EncodeToString(buf)
	k := &repository.APIKey{OrganizadorID: org.ID, Nome: req.Nome, Prefixo: chave[:10], Hash: middleware.HashAPIKey(chave), Ativo: true, CriadoEm: time.Now()}
	if err := h.chaves.Criar(k); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar chave"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": k.ID, "nome": k.Nome, "chave": chave, "aviso": "guarde esta chave agora; ela não será exibida de novo"})
}

// ListarChaves é GET /org/api-keys (só prefixo, nunca a chave).
func (h *APIPublicaHandler) ListarChaves(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	lista, err := h.chaves.ListarPorOrganizador(org.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar chaves"})
		return
	}
	resp := make([]gin.H, 0, len(lista))
	for _, k := range lista {
		resp = append(resp, gin.H{"id": k.ID, "nome": k.Nome, "prefixo": k.Prefixo, "ativo": k.Ativo, "criado_em": k.CriadoEm, "ultimo_uso_em": k.UltimoUsoEm})
	}
	c.JSON(http.StatusOK, resp)
}

// RevogarChave é DELETE /org/api-keys/:id.
func (h *APIPublicaHandler) RevogarChave(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	ok2, err := h.chaves.Revogar(org.ID, id)
	if err != nil || !ok2 {
		c.JSON(http.StatusNotFound, gin.H{"erro": "chave não encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- API pública (chave de API) ---

func eventoAPI(e *domain.Evento) gin.H {
	return gin.H{"id": e.ID, "titulo": e.Titulo, "slug": e.Slug, "status": e.Status, "inicio_em": e.InicioEm, "fim_em": e.FimEm,
		"categoria": e.Categoria, "tipo_acesso": e.TipoAcesso, "publicado_em": e.PublicadoEm}
}

// eventoDaChave carrega o evento :id garantindo que é do dono da chave.
func (h *APIPublicaHandler) eventoDaChave(c *gin.Context) (*domain.Evento, bool) {
	orgID := c.GetInt64(middleware.ChaveContextoOrganizadorAPI)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return nil, false
	}
	e, err := h.eventos.BuscarPorID(id)
	if err != nil || e.OrganizadorID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return nil, false
	}
	return e, true
}

// Eventos é GET /api/public/v1/eventos.
func (h *APIPublicaHandler) Eventos(c *gin.Context) {
	lista, err := h.eventos.ListarPorOrganizador(c.GetInt64(middleware.ChaveContextoOrganizadorAPI))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar eventos"})
		return
	}
	resp := make([]gin.H, 0, len(lista))
	for i := range lista {
		resp = append(resp, eventoAPI(&lista[i]))
	}
	c.JSON(http.StatusOK, gin.H{"dados": resp})
}

// Evento é GET /api/public/v1/eventos/:id.
func (h *APIPublicaHandler) Evento(c *gin.Context) {
	e, ok := h.eventoDaChave(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"dados": eventoAPI(e)})
}

// Ingressos é GET /api/public/v1/eventos/:id/ingressos: tipos com estoque.
func (h *APIPublicaHandler) Ingressos(c *gin.Context) {
	e, ok := h.eventoDaChave(c)
	if !ok {
		return
	}
	tipos, err := h.tipos.ListarPorEvento(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar ingressos"})
		return
	}
	ids := make([]int64, len(tipos))
	for i := range tipos {
		ids[i] = tipos[i].ID
	}
	ocupados, err := h.itens.ContarAtivosPorTipos(nil, ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao contar estoque"})
		return
	}
	resp := make([]gin.H, 0, len(tipos))
	for _, t := range tipos {
		resp = append(resp, gin.H{"id": t.ID, "nome": t.Nome, "preco_centavos": t.PrecoCentavos, "quantidade": t.Quantidade,
			"ocupados": ocupados[t.ID], "disponiveis": int64(t.Quantidade) - ocupados[t.ID], "meia_entrada": t.MeiaEntrada, "ativo": t.Ativo})
	}
	c.JSON(http.StatusOK, gin.H{"dados": resp})
}

// Participantes é GET /api/public/v1/eventos/:id/participantes: portadores
// de ingresso pago/utilizado (nome, e-mail, código, status) — dados pessoais
// entregues ao próprio organizador; nunca inclui qr_token.
func (h *APIPublicaHandler) Participantes(c *gin.Context) {
	e, ok := h.eventoDaChave(c)
	if !ok {
		return
	}
	itens, err := h.itens.ListarVendasPorEvento(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar participantes"})
		return
	}
	resp := make([]gin.H, 0, len(itens))
	for _, i := range itens {
		resp = append(resp, gin.H{"codigo": i.Codigo, "nome": i.TitularNome, "email": i.TitularEmail, "ingresso": i.TipoIngressoNome,
			"status": i.Status, "cortesia": i.Cortesia, "criado_em": i.CriadoEm, "utilizado_em": i.UtilizadoEm})
	}
	c.JSON(http.StatusOK, gin.H{"dados": resp})
}

// Resumo é GET /api/public/v1/eventos/:id/resumo: contadores de check-in.
func (h *APIPublicaHandler) Resumo(c *gin.Context) {
	e, ok := h.eventoDaChave(c)
	if !ok {
		return
	}
	r, err := h.itens.ResumoCheckin(e.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao calcular resumo"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dados": gin.H{"total_pagos": r.TotalPagos, "total_utilizados": r.TotalUtilizados}})
}
