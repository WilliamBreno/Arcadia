package storage

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// Banco guarda os uploads no Postgres (tabela arquivos). Serve para
// produção sem bucket: diferente do disco do servidor, não some a cada
// deploy. Os arquivos são servidos por GET /uploads/:nome (ServirArquivo).
type Banco struct {
	db      *gorm.DB
	baseURL string
}

// limiteArquivoBanco: as imagens já chegam reduzidas; isto só barra exagero.
const limiteArquivoBanco = 12 << 20

var nomeValido = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,120}$`)

func NovoBanco(db *gorm.DB, baseURL string) *Banco {
	return &Banco{db: db, baseURL: strings.TrimSuffix(baseURL, "/")}
}

// TipoPorExtensao: o tipo servido, pela extensão do nome gerado.
func TipoPorExtensao(nome string) string {
	switch strings.ToLower(filepath.Ext(nome)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".m4a":
		return "audio/mp4"
	}
	return "application/octet-stream"
}

func (b *Banco) Salvar(nomeArquivo string, conteudo io.Reader) (string, error) {
	if !nomeValido.MatchString(nomeArquivo) {
		return "", errors.New("nome de arquivo inválido")
	}
	dados, err := io.ReadAll(io.LimitReader(conteudo, limiteArquivoBanco+1))
	if err != nil {
		return "", err
	}
	if len(dados) > limiteArquivoBanco {
		return "", errors.New("arquivo grande demais")
	}
	err = b.db.Exec(`INSERT INTO arquivos (nome, content_type, tamanho, conteudo) VALUES (?, ?, ?, ?)`,
		nomeArquivo, TipoPorExtensao(nomeArquivo), len(dados), dados).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", b.baseURL, nomeArquivo), nil
}

// Ler devolve o arquivo guardado (nil se não existe).
func LerDoBanco(db *gorm.DB, nome string) (conteudo []byte, contentType string, err error) {
	if !nomeValido.MatchString(nome) {
		return nil, "", nil
	}
	var linha struct {
		ContentType string
		Conteudo    []byte
	}
	res := db.Raw(`SELECT content_type, conteudo FROM arquivos WHERE nome = ?`, nome).Scan(&linha)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, "", res.Error
	}
	return linha.Conteudo, linha.ContentType, nil
}
