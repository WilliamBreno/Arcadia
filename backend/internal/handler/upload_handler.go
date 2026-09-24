package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"image"
	"image/jpeg"
	_ "image/png" // registra o decoder de PNG em image.Decode
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/WilliamBreno/Arcadia/backend/internal/storage"
)

const (
	tamanhoMaximoImagemBytes = 5 * 1024 * 1024
	tamanhoMaximoAudioBytes  = 10 * 1024 * 1024
)

type UploadHandler struct {
	armazenamento storage.Armazenamento
}

func NovoUploadHandler(armazenamento storage.Armazenamento) *UploadHandler {
	return &UploadHandler{armazenamento: armazenamento}
}

func nomeAleatorio(extensao string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + extensao, nil
}

// Criar recebe multipart/form-data (campo "arquivo") e devolve a URL
// pública. Imagens (JPEG/PNG — WebP fica de fora por não ter decoder na
// stdlib do Go) são decodificadas e recodificadas como JPEG, o que já
// remove EXIF (localização, dispositivo etc.) por não copiar metadados,
// só os pixels (requisito de privacidade da seção 12 do plano).
// Redimensionar fica para quando houver uma lib de imagem no projeto;
// por ora só valida o tamanho máximo.
func (h *UploadHandler) Criar(c *gin.Context) {
	arquivo, cabecalho, err := c.Request.FormFile("arquivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "envie o arquivo no campo 'arquivo'"})
		return
	}
	defer arquivo.Close()

	contentType := cabecalho.Header.Get("Content-Type")

	switch contentType {
	case "image/jpeg", "image/png":
		h.salvarImagem(c, arquivo, cabecalho.Size)
	case "audio/mpeg", "audio/wav", "audio/mp4":
		h.salvarAudio(c, arquivo, cabecalho.Size, contentType)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"erro": "tipo de arquivo não suportado"})
	}
}

func (h *UploadHandler) salvarImagem(c *gin.Context, arquivo io.Reader, tamanho int64) {
	if tamanho > tamanhoMaximoImagemBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"erro": "imagem maior que 5MB"})
		return
	}

	img, _, err := image.Decode(arquivo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "arquivo de imagem inválido"})
		return
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao processar imagem"})
		return
	}

	nome, err := nomeAleatorio(".jpg")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar nome do arquivo"})
		return
	}

	url, err := h.armazenamento.Salvar(nome, &buf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar arquivo"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": url})
}

func (h *UploadHandler) salvarAudio(c *gin.Context, arquivo io.Reader, tamanho int64, contentType string) {
	if tamanho > tamanhoMaximoAudioBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"erro": "áudio maior que 10MB"})
		return
	}

	// Content-Type é declarado pelo cliente; confere a assinatura do arquivo.
	cabeca := make([]byte, 12)
	n, _ := io.ReadFull(arquivo, cabeca)
	if !assinaturaDeAudio(cabeca[:n], contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "o arquivo não parece ser um áudio válido"})
		return
	}
	arquivo = io.MultiReader(bytes.NewReader(cabeca[:n]), arquivo)

	extensao := map[string]string{"audio/mpeg": ".mp3", "audio/wav": ".wav", "audio/mp4": ".m4a"}[contentType]
	nome, err := nomeAleatorio(extensao)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao gerar nome do arquivo"})
		return
	}

	url, err := h.armazenamento.Salvar(nome, arquivo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"erro": "erro ao salvar arquivo"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"url": url})
}

// assinaturaDeAudio confere os primeiros bytes contra o formato declarado.
func assinaturaDeAudio(cabeca []byte, contentType string) bool {
	switch contentType {
	case "audio/mpeg":
		return bytes.HasPrefix(cabeca, []byte("ID3")) || (len(cabeca) >= 2 && cabeca[0] == 0xFF && cabeca[1]&0xE0 == 0xE0)
	case "audio/wav":
		return len(cabeca) >= 12 && string(cabeca[0:4]) == "RIFF" && string(cabeca[8:12]) == "WAVE"
	case "audio/mp4":
		return len(cabeca) >= 8 && string(cabeca[4:8]) == "ftyp"
	}
	return false
}
