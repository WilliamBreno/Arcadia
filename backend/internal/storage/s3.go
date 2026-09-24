package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 grava em qualquer armazenamento compatível com S3 (Cloudflare R2, AWS
// S3, Backblaze B2, Supabase Storage, MinIO). O bucket precisa servir os
// objetos publicamente em baseURL (nomes são aleatórios de 128 bits).
type S3 struct {
	cliente *minio.Client
	bucket  string
	baseURL string
}

type ConfigS3 struct {
	Endpoint  string // ex.: <conta>.r2.cloudflarestorage.com (sem https://)
	Bucket    string
	AccessKey string
	SecretKey string
	Regiao    string
	BaseURL   string // URL pública do bucket (ex.: https://cdn.evve.app)
	UsarSSL   bool
}

func NovoS3(c ConfigS3) (*S3, error) {
	if c.Endpoint == "" || c.Bucket == "" || c.AccessKey == "" || c.SecretKey == "" || c.BaseURL == "" {
		return nil, fmt.Errorf("configuração S3 incompleta (S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY, UPLOADS_BASE_URL)")
	}
	cli, err := minio.New(c.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure: c.UsarSSL,
		Region: c.Regiao,
	})
	if err != nil {
		return nil, err
	}
	return &S3{cliente: cli, bucket: c.Bucket, baseURL: strings.TrimSuffix(c.BaseURL, "/")}, nil
}

func (s *S3) Salvar(nomeArquivo string, conteudo io.Reader) (string, error) {
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()

	tipo := mime.TypeByExtension(filepath.Ext(nomeArquivo))
	if tipo == "" {
		tipo = "application/octet-stream"
	}
	// tamanho -1: envio em partes, sem precisar do tamanho de antemão
	_, err := s.cliente.PutObject(ctx, s.bucket, nomeArquivo, conteudo, -1, minio.PutObjectOptions{
		ContentType:  tipo,
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return "", fmt.Errorf("s3: %w", err)
	}
	return s.baseURL + "/" + nomeArquivo, nil
}
