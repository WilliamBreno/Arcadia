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

// Capa do evento: tamanho ideal 1600 × 900 px (16:9).
export const CAPA_LARGURA = 1600
export const CAPA_ALTURA = 900

export type CapaAjustada = { arquivo: File; original: { w: number; h: number }; final: { w: number; h: number }; corte: string; pequena: boolean }

// ajustarCapa recorta o centro da foto no formato 16:9 e reduz para no
// máximo 1600 × 900 (JPEG 85%), para a capa ficar igual na página do evento
// e nos cards. Avisa se a foto original for pequena (pode ficar borrada).
export async function ajustarCapa(arquivo: File): Promise<CapaAjustada> {
  const bitmap = await createImageBitmap(arquivo)
  const { width: w, height: h } = bitmap
  const alvo = CAPA_LARGURA / CAPA_ALTURA
  let sx = 0
  let sy = 0
  let sw = w
  let sh = h
  let corte = 'nenhum corte'
  if (w / h > alvo + 0.01) {
    sw = Math.round(h * alvo)
    sx = Math.round((w - sw) / 2)
    corte = 'laterais cortadas'
  } else if (w / h < alvo - 0.01) {
    sh = Math.round(w / alvo)
    sy = Math.round((h - sh) / 2)
    corte = 'partes de cima e de baixo cortadas'
  }
  const fw = Math.min(CAPA_LARGURA, sw)
  const fh = Math.round(fw / alvo)
  const canvas = document.createElement('canvas')
  canvas.width = fw
  canvas.height = fh
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Seu navegador não conseguiu ajustar a imagem.')
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(bitmap, sx, sy, sw, sh, 0, 0, fw, fh)
  bitmap.close()
  const blob = await new Promise<Blob | null>((ok) => canvas.toBlob(ok, 'image/jpeg', 0.85))
  if (!blob) throw new Error('Não foi possível ajustar a imagem.')
  return {
    arquivo: new File([blob], 'capa.jpg', { type: 'image/jpeg' }),
    original: { w, h },
    final: { w: fw, h: fh },
    corte,
    pequena: sw < 1200,
  }
}
