import { obterAccessToken } from '@/lib/api'

const BASE_URL = `${import.meta.env.VITE_API_URL ?? '/api'}/v1`

// enviarArquivo faz upload multipart (POST /uploads) e devolve a URL
// pública. Fica fora de lib/api.ts porque não é JSON — multipart precisa
// de FormData, sem Content-Type manual (o browser define o boundary).
export async function enviarArquivo(arquivo: File): Promise<string> {
  const formData = new FormData()
  formData.append('arquivo', arquivo)

  const token = obterAccessToken()
  const resposta = await fetch(`${BASE_URL}/uploads`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    credentials: 'include',
    body: formData,
  })

  const dados = await resposta.json().catch(() => null)
  if (!resposta.ok) {
    throw new Error(dados?.erro ?? 'Erro ao enviar arquivo')
  }
  return dados.url as string
}
