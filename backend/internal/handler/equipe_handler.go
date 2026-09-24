package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// EquipeHandler cuida da equipe do organizador e dos relatórios avançados
// (item 3.5). Tudo aqui é só do dono: equipe define quem opera os eventos e
// o relatório traz receita.
type EquipeHandler struct {
	organizadorHandler *OrganizadorHandler
	membros            *repository.OrganizadorMembroRepository
	usuarios           *repository.UsuarioRepository
	itensPedido        *repository.ItemPedidoRepository
	mailCliente        *mail.Cliente
	plataforma         string
	frontendURL        string
}

func NovoEquipeHandler(
	o *OrganizadorHandler, m *repository.OrganizadorMembroRepository, u *repository.UsuarioRepository,
	i *repository.ItemPedidoRepository, mc *mail.Cliente, plataforma, frontendURL string,
) *EquipeHandler {
	return &EquipeHandler{organizadorHandler: o, membros: m, usuarios: u, itensPedido: i, mailCliente: mc, plataforma: plataforma, frontendURL: frontendURL}
}

// Listar é GET /org/equipe.
func (h *EquipeHandler) Listar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	lista, err := h.membros.Listar(org.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao listar equipe"})
		return
	}
	resp := make([]gin.H, 0, len(lista))
	for _, m := range lista {
		resp = append(resp, gin.H{"usuario_id": m.UsuarioID, "nome": m.Nome, "email": m.Email, "papel": m.Papel, "desde": m.CriadoEm})
	}
	c.JSON(http.StatusOK, resp)
}

type adicionarMembroRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// Adicionar é POST /org/equipe: dá acesso de gestor (eventos, ingressos,
// participantes, check-in...) a alguém que já tem conta. Membro NÃO acessa
// perfil/Pix, financeiro, repasses, relatórios de receita, cancelamento de
// evento nem a própria equipe.
func (h *EquipeHandler) Adicionar(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	var req adicionarMembroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "informe um e-mail válido"})
		return
	}
	usuario, err := h.usuarios.BuscarPorEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "usuário não encontrado — peça pra essa pessoa criar uma conta primeiro"})
		return
	}
	if usuario.ID == org.UsuarioID {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "você já é o dono deste organizador"})
		return
	}
	if h.membros.Existe(org.ID, usuario.ID) {
		c.JSON(http.StatusConflict, gin.H{"erro": "essa pessoa já está na equipe"})
		return
	}
	if err := h.membros.Adicionar(&repository.OrganizadorMembro{OrganizadorID: org.ID, UsuarioID: usuario.ID, Papel: "gestor", CriadoEm: time.Now()}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao adicionar à equipe"})
		return
	}
	_ = h.mailCliente.Enviar(usuario.Email, "Você entrou na equipe de "+org.NomePublico+" — "+h.plataforma,
		"<p>Olá, "+usuario.Nome+"!</p><p>Você foi adicionado à equipe de <strong>"+org.NomePublico+"</strong> e já pode gerenciar os eventos em "+h.frontendURL+"/organizador/eventos.</p>")
	c.JSON(http.StatusCreated, gin.H{"usuario_id": usuario.ID, "nome": usuario.Nome, "email": usuario.Email, "papel": "gestor"})
}

// Remover é DELETE /org/equipe/:usuarioId.
func (h *EquipeHandler) Remover(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	usuarioID, err := strconv.ParseInt(c.Param("usuarioId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "id inválido"})
		return
	}
	removido, err := h.membros.Remover(org.ID, usuarioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao remover"})
		return
	}
	if !removido {
		c.JSON(http.StatusNotFound, gin.H{"erro": "membro não encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Relatorios é GET /org/relatorios?de=YYYY-MM-DD&ate=YYYY-MM-DD.
func (h *EquipeHandler) Relatorios(c *gin.Context) {
	org, ok := h.organizadorHandler.ObterOrganizadorDono(c)
	if !ok {
		return
	}
	eventos, err := h.itensPedido.RelatorioPorEvento(org.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar relatório"})
		return
	}
	dias, err := h.itensPedido.VendasPorDia(org.ID, parseDataQuery(c.Query("de")), parseDataQuery(c.Query("ate")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar relatório"})
		return
	}

	var totalVendidos, totalReceita, totalCheckins int64
	porEvento := make([]gin.H, 0, len(eventos))
	for _, e := range eventos {
		totalVendidos += e.Vendidos
		totalReceita += e.Receita
		totalCheckins += e.Checkins
		comparecimento := 0.0
		if e.Vendidos+e.Cortesias > 0 {
			comparecimento = float64(e.Checkins) / float64(e.Vendidos+e.Cortesias)
		}
		porEvento = append(porEvento, gin.H{
			"evento_id": e.EventoID, "titulo": e.Titulo, "status": e.Status, "vendidos": e.Vendidos, "cortesias": e.Cortesias,
			"receita_centavos": e.Receita, "checkins": e.Checkins, "cancelados": e.Cancelados, "comparecimento": comparecimento,
		})
	}
	porDia := make([]gin.H, 0, len(dias))
	for _, d := range dias {
		porDia = append(porDia, gin.H{"dia": d.Dia, "quantidade": d.Quantidade, "receita_centavos": d.Receita})
	}
	c.JSON(http.StatusOK, gin.H{
		"totais":         gin.H{"vendidos": totalVendidos, "receita_centavos": totalReceita, "checkins": totalCheckins},
		"por_evento":     porEvento,
		"vendas_por_dia": porDia,
	})
}
