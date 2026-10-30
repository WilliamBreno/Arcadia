package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/storage"
)

var nomeUpload = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,120}$`)

// ServirUpload entrega GET /uploads/:nome: primeiro do banco (padrão) e,
// se não estiver lá, da pasta local (arquivos de antes da mudança). Os
// nomes são aleatórios e nunca mudam de conteúdo: cache de um ano.
func ServirUpload(db *gorm.DB, pasta string) gin.HandlerFunc {
	return func(c *gin.Context) {
		nome := c.Param("nome")
		if !nomeUpload.MatchString(nome) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("X-Content-Type-Options", "nosniff")
		if dados, tipo, err := storage.LerDoBanco(db, nome); err == nil && dados != nil {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.Data(http.StatusOK, tipo, dados)
			return
		}
		caminho := filepath.Join(pasta, nome)
		if info, err := os.Stat(caminho); err == nil && !info.IsDir() {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.File(caminho)
			return
		}
		c.Status(http.StatusNotFound)
	}
}
