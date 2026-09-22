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
	eventos *repository.EventoRepository
	service *service.FichaService
}

func NovoFichaHandler(eventos *repository.EventoRepository, s *service.FichaService) *FichaHandler {
	return &FichaHandler{eventos: eventos, service: s}
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
