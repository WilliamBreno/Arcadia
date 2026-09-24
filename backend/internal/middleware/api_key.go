package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

const ChaveContextoOrganizadorAPI = "organizador_api_id"

// HashAPIKey é o SHA-256 em hex da chave em texto.
func HashAPIKey(chave string) string {
	h := sha256.Sum256([]byte(chave))
	return hex.EncodeToString(h[:])
}

// ExigirAPIKey autentica a API pública: header X-API-Key ou
// "Authorization: Bearer arc_...". Injeta o organizador dono da chave.
func ExigirAPIKey(repo *repository.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		chave := c.GetHeader("X-API-Key")
		if chave == "" {
			if partes := strings.SplitN(c.GetHeader("Authorization"), " ", 2); len(partes) == 2 && partes[0] == "Bearer" {
				chave = partes[1]
			}
		}
		if !strings.HasPrefix(chave, "arc_") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "chave de API ausente ou inválida"})
			return
		}
		k, err := repo.BuscarAtivaPorHash(HashAPIKey(chave))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "chave de API ausente ou inválida"})
			return
		}
		c.Set(ChaveContextoOrganizadorAPI, k.OrganizadorID)
		c.Next()
	}
}
