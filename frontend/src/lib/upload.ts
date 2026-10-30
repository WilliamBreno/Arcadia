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

// enviarFotoGaleria manda uma foto para a galeria do evento (item 8.7).
export async function enviarFotoGaleria(slug: string, arquivo: File): Promise<{ id: number; url: string }> {
  const formData = new FormData()
  formData.append('arquivo', arquivo)
  const token = obterAccessToken()
  const resposta = await fetch(`${BASE_URL}/eventos/${slug}/galeria`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    credentials: 'include',
    body: formData,
  })
  const dados = await resposta.json().catch(() => null)
  if (!resposta.ok) throw new Error(dados?.erro ?? 'Erro ao enviar a foto')
  return dados
}

// reduzirFoto diminui no navegador para no máximo 2048 px (JPEG 85%) antes
// de enviar: sobe mais rápido no celular e cabe no limite do proxy. O
// servidor confere e regrava de novo (tira metadados). Se o navegador não
// conseguir ler a imagem, manda o arquivo original.
export async function reduzirFoto(arquivo: File, lado = 2048): Promise<File> {
  try {
    const bitmap = await createImageBitmap(arquivo)
    const escala = Math.min(1, lado / Math.max(bitmap.width, bitmap.height))
    const w = Math.round(bitmap.width * escala)
    const h = Math.round(bitmap.height * escala)
    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    canvas.getContext('2d')?.drawImage(bitmap, 0, 0, w, h)
    bitmap.close()
    const blob = await new Promise<Blob | null>((ok) => canvas.toBlob(ok, 'image/jpeg', 0.85))
    if (!blob) return arquivo
    return new File([blob], arquivo.name.replace(/\.\w+$/, '') + '.jpg', { type: 'image/jpeg' })
  } catch {
    return arquivo
  }
}
