import { useRef, useState } from 'react'
import { QRCodeCanvas } from 'qrcode.react'
import { api } from '@/lib/api'
import { baseMural, fotoMural, type PaginaMural } from '@/lib/mural'
import { Button } from '@/components/ui/button'

function linhas(ctx: CanvasRenderingContext2D, texto: string, x: number, y: number, largura: number, altura: number, max: number) {
 let atual = ''; const partes: string[] = []
 for (const p of texto.split(/\s+/)) {
   const proxima = atual ? `${atual} ${p}` : p
   if (ctx.measureText(proxima).width > largura && atual) { partes.push(atual); atual = p }
   else atual = proxima
 }
 if (atual) partes.push(atual)
 partes.slice(0,max).forEach((linha,i) => ctx.fillText(i === max-1 && partes.length > max ? `${linha.replace(/\s+\S*$/, '')}…` : linha,x,y+i*altura,largura))
}
export function MuralStories({ slug }: { slug: string }) {
 const canvas = useRef<HTMLCanvasElement>(null), qr = useRef<HTMLCanvasElement>(null), ids = useRef<number[]>([])
 const [ocupado, setOcupado] = useState(false), [pronta, setPronta] = useState(false), [erro, setErro] = useState('')
 const gerar = async () => {
   setOcupado(true); setPronta(false); setErro('')
   try {
     const pagina = await api<PaginaMural>(baseMural(slug), { semAuth: true }), posts = pagina.destaques.slice(0,4)
     if (!posts.length) throw new Error('Destaque pelo menos uma publicação aprovada para gerar a arte.')
     const c = canvas.current!, ctx = c.getContext('2d')!
     ctx.fillStyle = '#102f28'; ctx.fillRect(0,0,1080,1920)
     ctx.fillStyle = '#8de3b4'; ctx.font = '600 36px system-ui'; ctx.fillText('EVVE · MELHORES MOMENTOS',64,96)
     ctx.fillStyle = '#ffffff'; ctx.font = '700 66px system-ui'; linhas(ctx,pagina.titulo,64,190,952,78,2)
     for (let i = 0; i < posts.length; i++) {
       const p = posts[i], unica = posts.length === 1, faixa = posts.length === 2 || (posts.length === 3 && i === 2)
       const x = unica || faixa ? 64 : 64 + (i % 2) * 488, y = unica ? 350 : faixa ? 350 + (posts.length === 2 ? i : 1) * 570 : 350 + Math.floor(i / 2) * 570
       const largura = unica || faixa ? 952 : 464, altura = unica ? 1120 : 540
       ctx.fillStyle = '#ffffff'; ctx.fillRect(x,y,largura,altura)
       let textoY = y + (unica ? 180 : 80), textoX = x + 24, larguraTexto = largura - 48
       if (p.fotos.length) {
         const blob = await fotoMural(slug,p.id,p.fotos[0]), img = await createImageBitmap(blob)
         const fotoL = unica ? 920 : 432, fotoA = unica ? 760 : faixa ? 508 : 350
         const escala = Math.max(fotoL/img.width,fotoA/img.height), w = img.width * escala, h = img.height * escala
         ctx.save(); ctx.beginPath(); ctx.rect(x+16,y+16,fotoL,fotoA); ctx.clip(); ctx.drawImage(img,x+16+(fotoL-w)/2,y+16+(fotoA-h)/2,w,h); ctx.restore(); img.close()
         if (faixa) { textoX = x+480; larguraTexto = 448; textoY = y+100 } else textoY = y + (unica ? 850 : 410)
       }
       ctx.fillStyle = '#152e27'; ctx.font = `600 ${unica ? 40 : 30}px system-ui`; linhas(ctx,p.texto || 'Um momento para guardar.',textoX,textoY,larguraTexto,unica ? 54 : 42,p.fotos.length && !faixa ? unica ? 4 : 2 : unica ? 12 : 8)
       ctx.fillStyle = '#53675d'; ctx.font = '400 24px system-ui'; ctx.fillText(p.autor,textoX,y+altura-30,larguraTexto)
     }
     ctx.fillStyle = '#ffffff'; ctx.fillRect(64,1570,250,250); if (qr.current) ctx.drawImage(qr.current,80,1586,218,218)
     ctx.fillStyle = '#ffffff'; ctx.font = '700 44px system-ui'; linhas(ctx,'Veja todos os momentos no mural',354,1640,660,56,3)
     ctx.font = '400 28px system-ui'; ctx.fillText('Aponte a câmera para o QR',354,1840)
     ids.current = posts.map(p => p.id); setPronta(true)
   } catch(e) { setErro(e instanceof Error ? e.message : 'Não foi possível gerar a arte.') }
   finally { setOcupado(false) }
 }
 const baixar = async () => {
   setOcupado(true); setErro('')
   try {
     const atual = await api<PaginaMural>(baseMural(slug), { semAuth: true })
     if (!ids.current.every(id => atual.destaques.some(p => p.id === id))) { setPronta(false); throw new Error('Os destaques mudaram. Gere uma nova prévia antes de baixar.') }
     const a = document.createElement('a'); a.download = `melhores-momentos-${slug}.png`; a.href = canvas.current!.toDataURL('image/png'); a.click()
   } catch(e) { setErro(e instanceof Error ? e.message : 'Não foi possível baixar a arte.') }
   finally { setOcupado(false) }
 }
 return <section className="flex flex-col items-start gap-3 rounded-xl border border-border p-4"><h2 className="font-semibold">Melhores momentos para Stories</h2><p className="text-sm text-muted-foreground">Arte 1080 × 1920 com até quatro destaques aprovados e QR para o mural. Confira os direitos de imagem antes de compartilhar.</p><div className="flex flex-wrap gap-2"><Button type="button" variant="outline" disabled={ocupado} onClick={gerar}>{ocupado ? 'Preparando…' : 'Gerar prévia dos melhores momentos'}</Button>{pronta && <Button type="button" disabled={ocupado} onClick={baixar}>Baixar Stories (PNG)</Button>}</div><canvas ref={canvas} width={1080} height={1920} hidden={!pronta} className="h-auto w-48 rounded-lg border border-border" role="img" aria-label="Prévia dos melhores momentos para Stories" /><QRCodeCanvas ref={qr} value={`${window.location.origin}/e/${slug}/mural`} size={218} className="hidden" />{erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}</section>
}
