import { useQuery } from '@tanstack/react-query'
import { Download } from 'lucide-react'
import { QRCodeCanvas } from 'qrcode.react'
import { useEffect, useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { api } from '@/lib/api'
import type { EventoDetalhe } from '@/lib/publico'

const L = 1080
const A = 1920

function quebrarLinhas(ctx: CanvasRenderingContext2D, texto: string, largura: number, maxLinhas: number) {
  const palavras = texto.split(/\s+/)
  const linhas: string[] = []
  let atual = ''
  for (const p of palavras) {
    const teste = atual ? `${atual} ${p}` : p
    if (ctx.measureText(teste).width > largura && atual) {
      linhas.push(atual)
      atual = p
    } else atual = teste
  }
  if (atual) linhas.push(atual)
  if (linhas.length > maxLinhas) {
    const cortadas = linhas.slice(0, maxLinhas)
    cortadas[maxLinhas - 1] = `${cortadas[maxLinhas - 1].replace(/\s+\S*$/, '')}…`
    return cortadas
  }
  return linhas
}

function carregarImagem(src: string) {
  return new Promise<HTMLImageElement | null>((ok) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => ok(img)
    img.onerror = () => ok(null)
    img.src = src
  })
}

// Item 10.5: arte do evento no formato Stories (1080 × 1920), desenhada no
// navegador, para baixar e postar no Instagram, WhatsApp ou TikTok.
export function ArteStories({ slug }: { slug: string }) {
  const { data, isError } = useQuery({ queryKey: ['evento-publico', slug], queryFn: () => api<EventoDetalhe>(`/eventos/${slug}`, { semAuth: true }), retry: false })
  const canvas = useRef<HTMLCanvasElement>(null)
  const qr = useRef<HTMLCanvasElement>(null)
  const [pronta, setPronta] = useState(false)
  const link = `${window.location.origin}/e/${slug}`

  useEffect(() => {
    const c = canvas.current
    const ctx = c?.getContext('2d')
    if (!c || !ctx || !data) return
    let cancelado = false
    ;(async () => {
      const e = data.evento
      const cor = e.cor_tema || '#111111'
      ctx.fillStyle = cor
      ctx.fillRect(0, 0, L, A)
      // Capa no topo (16:9 em largura total), escurecida embaixo para o texto.
      if (e.capa_url) {
        const img = await carregarImagem(e.capa_url)
        if (cancelado) return
        if (img) {
          const h = Math.round((L * 9) / 16)
          const escala = Math.max(L / img.width, h / img.height)
          const w = img.width * escala
          const hh = img.height * escala
          ctx.drawImage(img, (L - w) / 2, 140 + (h - hh) / 2, w, hh)
        }
      }
      ctx.fillStyle = 'rgba(0,0,0,0.35)'
      ctx.fillRect(0, 0, L, A)
      ctx.fillStyle = '#ffffff'
      ctx.font = '600 44px system-ui, sans-serif'
      const data_ = e.inicio_em ? new Date(e.inicio_em).toLocaleString('pt-BR', { weekday: 'long', day: '2-digit', month: 'long', hour: '2-digit', minute: '2-digit' }) : ''
      let y = 140 + Math.round((L * 9) / 16) + 110
      ctx.font = '800 92px system-ui, sans-serif'
      for (const linha of quebrarLinhas(ctx, e.titulo, L - 160, 4)) {
        ctx.fillText(linha, 80, y)
        y += 104
      }
      y += 30
      ctx.font = '500 48px system-ui, sans-serif'
      if (data_) {
        for (const linha of quebrarLinhas(ctx, data_.charAt(0).toUpperCase() + data_.slice(1), L - 160, 2)) {
          ctx.fillText(linha, 80, y)
          y += 64
        }
        y += 6
      }
      const onde = data.local ? `${data.local.nome}${data.local.cidade ? ` · ${data.local.cidade}` : ''}` : 'Online'
      for (const linha of quebrarLinhas(ctx, onde, L - 160, 2)) {
        ctx.fillText(linha, 80, y)
        y += 64
      }
      // QR e chamada no rodapé.
      const tamQR = 300
      ctx.fillStyle = '#ffffff'
      ctx.fillRect(80, A - tamQR - 200, tamQR + 40, tamQR + 40)
      if (qr.current) ctx.drawImage(qr.current, 100, A - tamQR - 180, tamQR, tamQR)
      // Chamada à direita do QR, quebrada para caber na largura que sobra.
      const xTexto = 460
      const largura = L - xTexto - 60
      let yt = A - tamQR - 140
      ctx.font = '700 52px system-ui, sans-serif'
      for (const linha of quebrarLinhas(ctx, 'Garanta o seu ingresso', largura, 2)) {
        ctx.fillText(linha, xTexto, yt)
        yt += 62
      }
      yt += 16
      ctx.font = '400 36px system-ui, sans-serif'
      for (const linha of quebrarLinhas(ctx, 'Aponte a câmera para o QR ou acesse o link na bio', largura, 3)) {
        ctx.fillText(linha, xTexto, yt)
        yt += 46
      }
      setPronta(true)
    })()
    return () => {
      cancelado = true
    }
  }, [data])

  const baixar = () => {
    const c = canvas.current
    if (!c) return
    try {
      const a = document.createElement('a')
      a.download = `stories-${slug}.png`
      a.href = c.toDataURL('image/png')
      a.click()
    } catch {
      // capa de outro site sem permissão: a arte sai sem baixar
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Arte para Stories</CardTitle>
        <p className="text-sm text-muted-foreground">Imagem 1080 × 1920 com a capa, a data, o local e um QR para a página de ingressos. Baixe e poste.</p>
      </CardHeader>
      <CardContent className="flex flex-col items-start gap-3">
        {isError && <p className="text-sm text-muted-foreground">A arte fica disponível quando o evento for publicado.</p>}
        <canvas hidden={isError} ref={canvas} width={L} height={A} className="h-auto w-48 rounded-lg border border-border" aria-label="Prévia da arte para Stories" role="img" />
        <QRCodeCanvas ref={qr} value={link} size={300} className="hidden" />
        <Button size="sm" onClick={baixar} disabled={!pronta} hidden={isError}>
          <Download className="size-4" aria-hidden="true" />
          Baixar a arte (PNG)
        </Button>
      </CardContent>
    </Card>
  )
}
