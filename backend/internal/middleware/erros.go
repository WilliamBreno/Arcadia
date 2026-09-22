package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErroAPI é o formato padronizado de erro retornado pela API.
type ErroAPI struct {
	Erro string `json:"erro"`
}

// TratadorDeErros recupera de panics nos handlers e responde com um erro
// padronizado em vez de derrubar o processo, além de registrar o log.
func TratadorDeErros() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recuperado", "erro", r, "caminho", c.Request.URL.Path)
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErroAPI{Erro: "erro interno do servidor"})
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			ultimoErro := c.Errors.Last()
			slog.Error("erro no handler", "erro", ultimoErro.Err, "caminho", c.Request.URL.Path)
			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, ErroAPI{Erro: "erro interno do servidor"})
			}
		}
	}
}
