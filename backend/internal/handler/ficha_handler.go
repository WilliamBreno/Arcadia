package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type FichaHandler struct {
	eventos            *repository.EventoRepository
	organizadorHandler *OrganizadorHandler
	service            *service.FichaService
}

func NovoFichaHandler(eventos *repository.EventoRepository, organizadorHandler *OrganizadorHandler, s *service.FichaService) *FichaHandler {
	return &FichaHandler{eventos: eventos, organizadorHandler: organizadorHandler, service: s}
}

// eventoPorSlugDaURL resolve o :slug da rota (compartilhado com as rotas
// públicas de evento no router) para o ID interno usado pelo service.
func (h *FichaHandler) eventoPorSlugDaURL(c *gin.Context) (int64, bool) {
	evento, err := h.eventos.BuscarPorSlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		return 0, false
	}
	return evento.ID, true
}

type fichaResposta struct {
	ID                 int64           `json:"id"`
	Papel              string          `json:"papel"`
	Nome               string          `json:"nome"`
	NomeArtistico      string          `json:"nome_artistico"`
	Instagram          string          `json:"instagram"`
	DataNascimento     *time.Time      `json:"data_nascimento"`
	FotoURL            string          `json:"foto_url"`
	Telefone           string          `json:"telefone"`
	Status             string          `json:"status"`
	MotivoRejeicao     string          `json:"motivo_rejeicao,omitempty"`
	OrdemApresentacao  *int            `json:"ordem_apresentacao"`
	TipoApresentacao   *string         `json:"tipo_apresentacao"`
	Dados              json.RawMessage `json:"dados"`
	ResponsavelNome    string          `json:"responsavel_nome,omitempty"`
	ResponsavelContato string          `json:"responsavel_contato,omitempty"`
}

func paraFichaResposta(f *domain.FichaParticipacao) fichaResposta {
	var tipoApresentacao *string
	if f.TipoApresentacao != nil {
		s := string(*f.TipoApresentacao)
		tipoApresentacao = &s
	}
	var dados json.RawMessage
	if len(f.Dados) > 0 {
		dados = json.RawMessage(f.Dados)
	}
	return fichaResposta{
		ID:                 f.ID,
		Papel:              string(f.Papel),
		Nome:               f.Nome,
		NomeArtistico:      f.NomeArtistico,
		Instagram:          f.Instagram,
		DataNascimento:     f.DataNascimento,
		FotoURL:            f.FotoURL,
		Telefone:           f.Telefone,
		Status:             string(f.Status),
		MotivoRejeicao:     f.MotivoRejeicao,
		OrdemApresentacao:  f.OrdemApresentacao,
		TipoApresentacao:   tipoApresentacao,
		Dados:              dados,
		ResponsavelNome:    f.ResponsavelNome,
		ResponsavelContato: f.ResponsavelContato,
	}
}

// fichaJuradoResposta é a visão do jurado (seção 3 do plano): nome, nome
// artístico, Instagram, idade (não a data de nascimento exata), foto,
// tipo de apresentação e dados da apresentação. Nunca telefone,
// responsável ou autorização.
type fichaJuradoResposta struct {
	ID                int64           `json:"id"`
	Nome              string          `json:"nome"`
	NomeArtistico     string          `json:"nome_artistico"`
	Instagram         string          `json:"instagram"`
	Idade             *int            `json:"idade"`
	FotoURL           string          `json:"foto_url"`
	OrdemApresentacao *int            `json:"ordem_apresentacao"`
	TipoApresentacao  *string         `json:"tipo_apresentacao"`
	Dados             json.RawMessage `json:"dados"`
}

func paraFichaJuradoResposta(f *domain.FichaParticipacao) fichaJuradoResposta {
	var tipoApresentacao *string
	if f.TipoApresentacao != nil {
		s := string(*f.TipoApresentacao)
		tipoApresentacao = &s
	}
	var dados json.RawMessage
	if len(f.Dados) > 0 {
		dados = json.RawMessage(f.Dados)
	}
	var idade *int
	if f.DataNascimento != nil {
		anos := time.Now().Year() - f.DataNascimento.Year()
		if time.Now().YearDay() < f.DataNascimento.YearDay() {
			anos--
		}
		idade = &anos
	}
	return fichaJuradoResposta{
		ID:                f.ID,
		Nome:              f.Nome,
		NomeArtistico:     f.NomeArtistico,
		Instagram:         f.Instagram,
		Idade:             idade,
		FotoURL:           f.FotoURL,
		OrdemApresentacao: f.OrdemApresentacao,
		TipoApresentacao:  tipoApresentacao,
		Dados:             dados,
	}
}

type minhaFichaRequest struct {
	Nome                      string         `json:"nome" binding:"required,min=2"`
	NomeArtistico             string         `json:"nome_artistico"`
	Instagram                 string         `json:"instagram"`
	DataNascimento            *time.Time     `json:"data_nascimento"`
	FotoURL                   string         `json:"foto_url"`
	Telefone                  string         `json:"telefone"`
	TipoApresentacao          *string        `json:"tipo_apresentacao"`
	Dados                     map[string]any `json:"dados"`
	ResponsavelNome           string         `json:"responsavel_nome"`
	ResponsavelContato        string         `json:"responsavel_contato"`
	AutorizacaoResponsavelURL string         `json:"autorizacao_responsavel_url"`
}

func (req minhaFichaRequest) paraDados() service.FichaDados {
	var tipoApresentacao *domain.TipoApresentacao
	if req.TipoApresentacao != nil {
		t := domain.TipoApresentacao(*req.TipoApresentacao)
		tipoApresentacao = &t
	}
	var dadosJSON datatypes.JSON
	if req.Dados != nil {
		if bruto, err := json.Marshal(req.Dados); err == nil {
			dadosJSON = datatypes.JSON(bruto)
		}
	}
	return service.FichaDados{
		Nome:                      req.Nome,
		NomeArtistico:             req.NomeArtistico,
		Instagram:                 req.Instagram,
		DataNascimento:            req.DataNascimento,
		FotoURL:                   req.FotoURL,
		Telefone:                  req.Telefone,
		TipoApresentacao:          tipoApresentacao,
		Dados:                     dadosJSON,
		ResponsavelNome:           req.ResponsavelNome,
		ResponsavelContato:        req.ResponsavelContato,
		AutorizacaoResponsavelURL: req.AutorizacaoResponsavelURL,
	}
}

// qualPapel resolve se a ficha consultada é de jurado ou participante —
// um usuário pode ter as duas num mesmo evento (papéis independentes),
// então o chamador informa via query "?papel=jurado|participante".
func qualPapel(c *gin.Context) domain.Papel {
	if c.Query("papel") == "jurado" {
		return domain.PapelJurado
	}
	return domain.PapelParticipante
}

func (h *FichaHandler) ObterMinha(c *gin.Context) {
	eventoID, ok := h.eventoPorSlugDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	ficha, err := h.service.ObterMinha(eventoID, usuarioID, qualPapel(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "ficha não encontrada"})
		return
	}
	c.JSON(http.StatusOK, paraFichaResposta(ficha))
}

func (h *FichaHandler) AtualizarMinha(c *gin.Context) {
	eventoID, ok := h.eventoPorSlugDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	var req minhaFichaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	ficha, err := h.service.AtualizarMinha(eventoID, usuarioID, qualPapel(c), req.paraDados())
	if err != nil {
		if errors.Is(err, service.ErrMenorSemAutorizacao) {
			c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"erro": "ficha não encontrada"})
		return
	}
	c.JSON(http.StatusOK, paraFichaResposta(ficha))
}

// --- Inscrição aberta (participante) ---

func (h *FichaHandler) Inscrever(c *gin.Context) {
	eventoID, ok := h.eventoPorSlugDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	var req minhaFichaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	ficha, err := h.service.Inscrever(eventoID, usuarioID, req.paraDados())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMenorSemAutorizacao):
			c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		case errors.Is(err, service.ErrInscricaoNaoPermitida), errors.Is(err, service.ErrInscricaoForaDoPrazo), errors.Is(err, service.ErrJaInscrito):
			c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
		default:
			c.JSON(http.StatusNotFound, gin.H{"erro": "evento não encontrado"})
		}
		return
	}
	c.JSON(http.StatusCreated, paraFichaResposta(ficha))
}

// --- Organizador: aprovação de participantes ---

func (h *FichaHandler) ListarDoOrganizador(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}

	fichas, err := h.service.ListarDoOrganizador(organizador.ID, eventoID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"erro": "evento não pertence a este organizador"})
		return
	}

	resposta := make([]fichaResposta, 0, len(fichas))
	for i := range fichas {
		resposta = append(resposta, paraFichaResposta(&fichas[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

func (h *FichaHandler) Aprovar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	fichaID, ok := parseParamInt64(c, "fichaId")
	if !ok {
		return
	}

	ficha, err := h.service.Aprovar(organizador.ID, eventoID, fichaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "ficha não encontrada"})
		return
	}
	c.JSON(http.StatusOK, paraFichaResposta(ficha))
}

type rejeitarFichaRequest struct {
	Motivo string `json:"motivo"`
}

func (h *FichaHandler) Rejeitar(c *gin.Context) {
	organizador, ok := h.organizadorHandler.ObterOrganizadorAtual(c)
	if !ok {
		return
	}
	eventoID, ok := idDaURL(c)
	if !ok {
		return
	}
	fichaID, ok := parseParamInt64(c, "fichaId")
	if !ok {
		return
	}

	var req rejeitarFichaRequest
	_ = c.ShouldBindJSON(&req)

	ficha, err := h.service.Rejeitar(organizador.ID, eventoID, fichaID, req.Motivo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "ficha não encontrada"})
		return
	}
	c.JSON(http.StatusOK, paraFichaResposta(ficha))
}

// --- Jurado (leitura) ---

func (h *FichaHandler) ListarParaJurado(c *gin.Context) {
	eventoID, ok := h.eventoPorSlugDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	fichas, err := h.service.ListarParaJurado(usuarioID, eventoID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"erro": "você não é jurado confirmado deste evento"})
		return
	}

	resposta := make([]fichaJuradoResposta, 0, len(fichas))
	for i := range fichas {
		resposta = append(resposta, paraFichaJuradoResposta(&fichas[i]))
	}
	c.JSON(http.StatusOK, resposta)
}

func (h *FichaHandler) ObterParaJurado(c *gin.Context) {
	eventoID, ok := h.eventoPorSlugDaURL(c)
	if !ok {
		return
	}
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)
	fichaID, ok := parseParamInt64(c, "fichaId")
	if !ok {
		return
	}

	ficha, err := h.service.ObterParaJurado(usuarioID, eventoID, fichaID)
	if err != nil {
		if errors.Is(err, service.ErrNaoEhJuradoConfirmado) {
			c.JSON(http.StatusForbidden, gin.H{"erro": "você não é jurado confirmado deste evento"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"erro": "ficha não encontrada"})
		return
	}
	c.JSON(http.StatusOK, paraFichaJuradoResposta(ficha))
}
