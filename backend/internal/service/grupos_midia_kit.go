package service

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/storage"
)

func (s *GruposService) ConfigurarImagensMidiaKit(baseURL, diretorio string) {
	s.kitUploadsBaseURL, s.kitUploadsDir = strings.TrimRight(baseURL, "/"), diretorio
}

var nomeFotoKit = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,110}\.(jpg|jpeg|png)$`)

// Só acessa imagens no armazenamento configurado, nunca URLs arbitrárias do perfil.
// Sem redirecionamentos, caminho livre, credenciais ou query string.
func nomeImagemKit(base, imagemURL string) string {
	b, e1 := url.Parse(base)
	u, e2 := url.Parse(imagemURL)
	if e1 != nil || e2 != nil || base == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
		return ""
	}
	if u.Scheme != b.Scheme || u.Host != b.Host {
		return ""
	}
	prefixo := strings.TrimRight(b.Path, "/") + "/"
	if !strings.HasPrefix(u.Path, prefixo) {
		return ""
	}
	nome := strings.TrimPrefix(u.Path, prefixo)
	if !nomeFotoKit.MatchString(nome) {
		return ""
	}
	return nome
}

func (s *GruposService) imagemMidiaKit(imagemURL string) []byte {
	nome := nomeImagemKit(s.kitUploadsBaseURL, imagemURL)
	if nome == "" {
		return nil
	}
	b, _, err := storage.LerDoBanco(s.db, nome)
	if err != nil {
		return nil
	}
	if len(b) == 0 && s.kitUploadsDir != "" {
		if f, e := os.Open(filepath.Join(s.kitUploadsDir, nome)); e == nil {
			b, _ = io.ReadAll(io.LimitReader(f, 5*1024*1024+1))
			f.Close()
		}
	}
	if len(b) == 0 && (strings.HasPrefix(s.kitUploadsBaseURL, "https://") || strings.HasPrefix(s.kitUploadsBaseURL, "http://")) {
		client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		resp, e := client.Get(imagemURL)
		if e != nil {
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil
		}
		b, _ = io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024+1))
	}
	return normalizarFotoKit(b)
}

func normalizarFotoKit(b []byte) []byte {
	if len(b) == 0 || len(b) > 5*1024*1024 {
		return nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 20_000_000 {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	var out bytes.Buffer
	if jpeg.Encode(&out, img, &jpeg.Options{Quality: 80}) != nil {
		return nil
	}
	return out.Bytes()
}
