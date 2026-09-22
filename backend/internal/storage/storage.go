package storage

import "io"

// Armazenamento salva um arquivo e devolve a URL pública dele.
//
// Implementação atual: disco local (DiscoLocal) — serve para dev, mas
// não funciona em hosts com filesystem efêmero (Render, por exemplo).
// Antes de ir para produção, trocar por um cliente S3-compatível
// apontado para Supabase Storage ou Cloudflare R2 (decisão pendente,
// seção 4/14 do plano) implementando esta mesma interface.
type Armazenamento interface {
	Salvar(nomeArquivo string, conteudo io.Reader) (url string, err error)
}
