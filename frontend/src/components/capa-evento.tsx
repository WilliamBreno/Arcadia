import { ImagePlus } from 'lucide-react'
import { useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { enviarArquivo, reduzirFoto } from '@/lib/upload'

// Capa do evento: aparece no topo da página do evento e nos cards da
// vitrine (16:9). A imagem é reduzida no navegador antes de subir.
export function CapaEvento({ valor, aoMudar }: { valor: string; aoMudar: (url: string) => void }) {
  const entrada = useRef<HTMLInputElement>(null)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  const escolher = async (arquivo?: File) => {
    if (!arquivo) return
    setErro(null)
    setEnviando(true)
    try {
      aoMudar(await enviarArquivo(await reduzirFoto(arquivo, 1920)))
    } catch (e) {
      setErro(e instanceof Error ? e.message : 'Não foi possível enviar a imagem.')
    } finally {
      setEnviando(false)
      if (entrada.current) entrada.current.value = ''
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium text-foreground" id="capa-rotulo">
        Imagem de capa
      </span>
      <button
        type="button"
        onClick={() => entrada.current?.click()}
        disabled={enviando}
        aria-labelledby="capa-rotulo"
        aria-describedby="capa-ajuda"
        className="relative flex aspect-video w-full items-center justify-center overflow-hidden rounded-xl border border-dashed border-border bg-muted text-sm text-muted-foreground hover:bg-muted/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        {valor ? (
          <img src={valor} alt="Capa atual do evento" className="size-full object-cover" />
        ) : (
          <span className="flex flex-col items-center gap-2">
            <ImagePlus className="size-6" aria-hidden="true" />
            {enviando ? 'Enviando…' : 'Escolher imagem'}
          </span>
        )}
        {valor && enviando && <span className="absolute inset-0 flex items-center justify-center bg-black/50 text-white">Enviando…</span>}
      </button>
      <input
        ref={entrada}
        type="file"
        accept="image/jpeg,image/png"
        className="sr-only"
        tabIndex={-1}
        aria-hidden="true"
        onChange={(e) => escolher(e.target.files?.[0])}
      />
      <p id="capa-ajuda" className="text-xs text-muted-foreground">
        Horizontal, de preferência 1600 × 900 px (JPEG ou PNG). Aparece no topo da página do evento e nos cards da vitrine.
      </p>
      {valor && (
        <div className="flex gap-2">
          <Button type="button" size="sm" variant="outline" disabled={enviando} onClick={() => entrada.current?.click()}>
            Trocar imagem
          </Button>
          <Button type="button" size="sm" variant="ghost" disabled={enviando} onClick={() => aoMudar('')}>
            Remover
          </Button>
        </div>
      )}
      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
    </div>
  )
}
