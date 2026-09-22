package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

const ChaveContextoUsuarioID = "usuario_id"
const ChaveContextoPapelPlataforma = "papel_plataforma"

// ExigirAutenticacao valida o access token (Bearer JWT) e injeta o
// usuário autenticado no contexto da requisição.
func ExigirAutenticacao(jwtService *service.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		cabecalho := c.GetHeader("Authorization")
		partes := strings.SplitN(cabecalho, " ", 2)
		if len(partes) != 2 || partes[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "não autenticado"})
			return
		}

		claims, err := jwtService.ValidarAccessToken(partes[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "sessão expirada"})
			return
		}

		c.Set(ChaveContextoUsuarioID, claims.UsuarioID)
		c.Set(ChaveContextoPapelPlataforma, claims.PapelPlataforma)
		c.Next()
	}
}

// ExigirAdminPlataforma deve ser usado após ExigirAutenticacao.
func ExigirAdminPlataforma() gin.HandlerFunc {
	return func(c *gin.Context) {
		papel, _ := c.Get(ChaveContextoPapelPlataforma)
		if papel != domain.PapelAdminPlataforma {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"erro": "acesso restrito ao admin da plataforma"})
			return
		}
		c.Next()
	}
}
