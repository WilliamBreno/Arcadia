package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DiscoLocal grava arquivos em uma pasta local e serve pela própria API
// (ver rota estática registrada em cmd/api/main.go).
type DiscoLocal struct {
	diretorio string
	baseURL   string
}

func NovoDiscoLocal(diretorio, baseURL string) (*DiscoLocal, error) {
	if err := os.MkdirAll(diretorio, 0o755); err != nil {
		return nil, err
	}
	return &DiscoLocal{diretorio: diretorio, baseURL: baseURL}, nil
}

func (d *DiscoLocal) Salvar(nomeArquivo string, conteudo io.Reader) (string, error) {
	caminho := filepath.Join(d.diretorio, nomeArquivo)

	arquivo, err := os.Create(caminho)
	if err != nil {
		return "", err
	}
	defer arquivo.Close()

	if _, err := io.Copy(arquivo, conteudo); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", strings.TrimSuffix(d.baseURL, "/"), nomeArquivo), nil
}
