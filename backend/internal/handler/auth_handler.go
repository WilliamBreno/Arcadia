package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/config"
	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/middleware"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

type AuthHandler struct {
	usuarios *repository.UsuarioRepository
	auth     *service.AuthService
	google   *service.GoogleAuthService
	mail     *mail.Cliente
	cfg      config.Config
}

func NovoAuthHandler(usuarios *repository.UsuarioRepository, auth *service.AuthService, google *service.GoogleAuthService, mailCliente *mail.Cliente, cfg config.Config) *AuthHandler {
	return &AuthHandler{usuarios: usuarios, auth: auth, google: google, mail: mailCliente, cfg: cfg}
}

const nomeCookieRefresh = "refresh_token"

func (h *AuthHandler) definirCookieRefresh(c *gin.Context, token string, ttl time.Duration) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     nomeCookieRefresh,
		Value:    token,
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(ttl),
		HttpOnly: true,
		Secure:   h.cfg.AmbienteApp == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) limparCookieRefresh(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     nomeCookieRefresh,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.AmbienteApp == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

type usuarioResposta struct {
	ID              int64  `json:"id"`
	Nome            string `json:"nome"`
	Email           string `json:"email"`
	AvatarURL       string `json:"avatar_url"`
	EmailVerificado bool   `json:"email_verificado"`
	PapelPlataforma string `json:"papel_plataforma"`
}

func paraUsuarioResposta(u *domain.Usuario) usuarioResposta {
	return usuarioResposta{
		ID:              u.ID,
		Nome:            u.Nome,
		Email:           u.Email,
		AvatarURL:       u.AvatarURL,
		EmailVerificado: u.EmailVerificadoEm != nil,
		PapelPlataforma: string(u.PapelPlataforma),
	}
}

func (h *AuthHandler) enviarEmailVerificacao(usuario *domain.Usuario) {
	token, err := h.auth.SolicitarVerificacaoEmail(usuario)
	if err != nil {
		return
	}
	link := fmt.Sprintf("%s/verificar-email?token=%s", h.cfg.FrontendURL, token)
	corpo := fmt.Sprintf(`<p>Olá, %s!</p><p>Confirme seu e-mail em %s: <a href="%s">%s</a></p>`, usuario.Nome, h.cfg.NomePlataforma, link, link)
	_ = h.mail.Enviar(usuario.Email, "Confirme seu e-mail — "+h.cfg.NomePlataforma, corpo)
}

// --- POST /auth/cadastro ---

type cadastroRequest struct {
	Nome  string `json:"nome" binding:"required,min=2"`
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required,min=8"`
}

func (h *AuthHandler) Cadastro(c *gin.Context) {
	var req cadastroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuario, err := h.auth.Cadastrar(req.Nome, req.Email, req.Senha)
	if err != nil {
		if errors.Is(err, service.ErrEmailJaCadastrado) {
			c.JSON(http.StatusConflict, gin.H{"erro": "e-mail já cadastrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao cadastrar"})
		return
	}

	h.enviarEmailVerificacao(usuario)

	accessToken, refreshToken, err := h.auth.EmitirTokens(usuario)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao autenticar"})
		return
	}

	h.definirCookieRefresh(c, refreshToken, h.auth.RefreshTTL())
	c.JSON(http.StatusCreated, gin.H{"access_token": accessToken, "usuario": paraUsuarioResposta(usuario)})
}

// --- POST /auth/login ---

type loginRequest struct {
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuario, err := h.auth.Login(req.Email, req.Senha)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "email ou senha inválidos"})
		return
	}

	accessToken, refreshToken, err := h.auth.EmitirTokens(usuario)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao autenticar"})
		return
	}

	h.definirCookieRefresh(c, refreshToken, h.auth.RefreshTTL())
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "usuario": paraUsuarioResposta(usuario)})
}

// --- POST /auth/google ---

type googleRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func (h *AuthHandler) Google(c *gin.Context) {
	if !h.google.Habilitado() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"erro": "login com Google não configurado"})
		return
	}

	var req googleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	perfil, err := h.google.ValidarIDToken(c.Request.Context(), req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "id_token do Google inválido"})
		return
	}

	usuario, err := h.auth.LoginOuCriarComGoogle(perfil.GoogleID, perfil.Email, perfil.Nome, perfil.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao autenticar com Google"})
		return
	}

	accessToken, refreshToken, err := h.auth.EmitirTokens(usuario)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao autenticar"})
		return
	}

	h.definirCookieRefresh(c, refreshToken, h.auth.RefreshTTL())
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "usuario": paraUsuarioResposta(usuario)})
}

// --- POST /auth/refresh ---

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(nomeCookieRefresh)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "não autenticado"})
		return
	}

	usuario, accessToken, novoRefreshToken, err := h.auth.RenovarTokens(refreshToken)
	if err != nil {
		h.limparCookieRefresh(c)
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "sessão expirada"})
		return
	}

	h.definirCookieRefresh(c, novoRefreshToken, h.auth.RefreshTTL())
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "usuario": paraUsuarioResposta(usuario)})
}

// --- POST /auth/logout ---

func (h *AuthHandler) Logout(c *gin.Context) {
	if refreshToken, err := c.Cookie(nomeCookieRefresh); err == nil && refreshToken != "" {
		_ = h.auth.Logout(refreshToken)
	}
	h.limparCookieRefresh(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- POST /auth/verificar-email ---

type verificarEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *AuthHandler) VerificarEmail(c *gin.Context) {
	var req verificarEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	if err := h.auth.VerificarEmail(req.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "token inválido ou expirado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- POST /auth/esqueci-senha ---

type esqueciSenhaRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) EsqueciSenha(c *gin.Context) {
	var req esqueciSenhaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	usuario, token, err := h.auth.SolicitarRedefinicaoSenha(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar solicitação"})
		return
	}

	// Resposta genérica sempre — não revela se o e-mail existe.
	if usuario != nil {
		link := fmt.Sprintf("%s/redefinir-senha?token=%s", h.cfg.FrontendURL, token)
		corpo := fmt.Sprintf(`<p>Olá, %s!</p><p>Redefina sua senha em %s: <a href="%s">%s</a></p><p>O link expira em 1 hora.</p>`, usuario.Nome, h.cfg.NomePlataforma, link, link)
		_ = h.mail.Enviar(usuario.Email, "Redefinir senha — "+h.cfg.NomePlataforma, corpo)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- POST /auth/redefinir-senha ---

type redefinirSenhaRequest struct {
	Token string `json:"token" binding:"required"`
	Senha string `json:"senha" binding:"required,min=8"`
}

func (h *AuthHandler) RedefinirSenha(c *gin.Context) {
	var req redefinirSenhaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "dados inválidos"})
		return
	}

	if err := h.auth.RedefinirSenha(req.Token, req.Senha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "token inválido ou expirado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- GET /me ---

func (h *AuthHandler) Me(c *gin.Context) {
	usuarioID := c.GetInt64(middleware.ChaveContextoUsuarioID)

	usuario, err := h.usuarios.BuscarPorID(usuarioID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"erro": "usuário não encontrado"})
		return
	}

	c.JSON(http.StatusOK, paraUsuarioResposta(usuario))
}
