import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'
import { enviarFoto, galeria, galeriaEu, marcarFoto, pedirRemocao, type Foto } from '@/lib/galeria'
import { reduzirFoto } from '@/lib/upload'

// Galeria oficial do evento (item 8.7): fotos do organizador e dos
// fotógrafos convidados. Adulto com perfil de talento se marca; qualquer
// pessoa pede "não quero aparecer" e a foto some na hora.
export function GaleriaDoEvento({ slug }: { slug: string }) {
  const { usuario } = useAuth()
  const queryClient = useQueryClient()
  const { data: fotos } = useQuery({ queryKey: ['galeria', slug], queryFn: () => galeria(slug).catch(() => [] as Foto[]) })
  const { data: eu } = useQuery({
    queryKey: ['galeria-eu', slug, usuario?.id],
    enabled: !!usuario,
    queryFn: () => galeriaEu(slug).catch(() => ({ pode_enviar: false, marcadas: [] as number[] })),
  })
  const [aberta, setAberta] = useState<number | null>(null)
  const [enviando, setEnviando] = useState<{ feitas: number; total: number } | null>(null)
  const [erro, setErro] = useState<string | null>(null)

  const recarregar = () =>
    Promise.all([queryClient.invalidateQueries({ queryKey: ['galeria', slug] }), queryClient.invalidateQueries({ queryKey: ['galeria-eu', slug] })])

  const enviar = async (lista: FileList | null) => {
    if (!lista?.length) return
    setErro(null)
    const arquivos = [...lista]
    let falhas = 0
    let ultimo = ''
    for (let i = 0; i < arquivos.length; i++) {
      setEnviando({ feitas: i, total: arquivos.length })
      try {
        await enviarFoto(slug, await reduzirFoto(arquivos[i]))
      } catch (e) {
        falhas++
        ultimo = e instanceof Error ? e.message : 'erro ao enviar'
      }
    }
    setEnviando(null)
    if (falhas) setErro(`${falhas} de ${arquivos.length} foto(s) não foram enviadas: ${ultimo}`)
    recarregar()
  }

  if (!fotos || (fotos.length === 0 && !eu?.pode_enviar)) return null
  const indice = aberta === null ? -1 : fotos.findIndex((f) => f.id === aberta)

  return (
    <section id="galeria" aria-labelledby="titulo-galeria" className="mt-8 scroll-mt-20">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="titulo-galeria" className="text-xl font-semibold text-foreground">
          Galeria oficial
        </h2>
        {fotos.length > 0 && <span className="text-sm text-muted-foreground">{fotos.length} foto(s)</span>}
      </div>

      {eu?.pode_enviar && (
        <div className="mt-3 flex flex-wrap items-center gap-3 rounded-lg border border-dashed border-border p-3 text-sm">
          <label className="cursor-pointer font-medium text-primary underline-offset-4 hover:underline">
            Enviar fotos
            <input type="file" accept="image/jpeg,image/png" multiple className="sr-only" disabled={!!enviando} onChange={(e) => enviar(e.target.files)} />
          </label>
          <span className="text-muted-foreground" aria-live="polite">
            {enviando ? `Enviando ${enviando.feitas + 1} de ${enviando.total}…` : 'JPEG ou PNG. As fotos são reduzidas e os dados de localização são apagados.'}
          </span>
          {erro && (
            <p role="alert" className="w-full text-destructive">
              {erro}
            </p>
          )}
        </div>
      )}

      {fotos.length > 0 && (
        <ul className="mt-3 grid grid-cols-3 gap-1 sm:grid-cols-4">
          {fotos.map((f) => (
            <li key={f.id}>
              <button
                type="button"
                onClick={() => setAberta(f.id)}
                className="block aspect-square w-full overflow-hidden rounded-md bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                aria-label={`Abrir foto de ${f.credito}`}
              >
                <img src={f.url} alt="" loading="lazy" decoding="async" className="size-full object-cover transition-opacity hover:opacity-90" />
              </button>
            </li>
          ))}
        </ul>
      )}

      {indice >= 0 && (
        <VerFoto
          key={fotos[indice].id}
          foto={fotos[indice]}
          marcada={!!eu?.marcadas.includes(fotos[indice].id)}
          logado={!!usuario}
          anterior={indice > 0 ? () => setAberta(fotos[indice - 1].id) : undefined}
          proxima={indice < fotos.length - 1 ? () => setAberta(fotos[indice + 1].id) : undefined}
          fechar={() => setAberta(null)}
          aoMudar={recarregar}
        />
      )}
    </section>
  )
}

function VerFoto({
  foto,
  marcada,
  logado,
  anterior,
  proxima,
  fechar,
  aoMudar,
}: {
  foto: Foto
  marcada: boolean
  logado: boolean
  anterior?: () => void
  proxima?: () => void
  fechar: () => void
  aoMudar: () => Promise<unknown>
}) {
  const ref = useRef<HTMLDialogElement>(null)
  const [remover, setRemover] = useState(false)
  const [motivo, setMotivo] = useState('')
  const [msg, setMsg] = useState<string | null>(null)

  useEffect(() => {
    const d = ref.current
    if (d && !d.open) d.showModal()
  }, [])
  useEffect(() => {
    const tecla = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft') anterior?.()
      if (e.key === 'ArrowRight') proxima?.()
    }
    window.addEventListener('keydown', tecla)
    return () => window.removeEventListener('keydown', tecla)
  }, [anterior, proxima])

  const marcar = async () => {
    setMsg(null)
    try {
      await marcarFoto(foto.id, !marcada)
      await aoMudar()
    } catch (e) {
      setMsg(e instanceof ApiError ? e.message : 'Não foi possível agora.')
    }
  }
  const confirmarRemocao = async () => {
    try {
      await pedirRemocao(foto.id, motivo)
      await aoMudar()
      fechar()
    } catch (e) {
      setMsg(e instanceof ApiError ? e.message : 'Não foi possível agora.')
    }
  }

  return (
    <dialog
      ref={ref}
      onClose={fechar}
      aria-label="Foto da galeria"
      className="m-auto w-full max-w-4xl rounded-xl bg-background p-0 text-foreground backdrop:bg-black/80"
      style={{ maxHeight: '92svh' }}
    >
      <div className="relative bg-black">
        <img src={foto.url} alt={`Foto de ${foto.credito}`} className="mx-auto block w-auto object-contain" style={{ maxHeight: '70svh' }} />
        <button type="button" onClick={() => ref.current?.close()} aria-label="Fechar" className="absolute right-2 top-2 rounded-full bg-black/60 p-2 text-white">
          <X className="size-5" aria-hidden="true" />
        </button>
        {anterior && (
          <button type="button" onClick={anterior} aria-label="Foto anterior" className="absolute left-2 top-1/2 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white">
            <ChevronLeft className="size-5" aria-hidden="true" />
          </button>
        )}
        {proxima && (
          <button type="button" onClick={proxima} aria-label="Próxima foto" className="absolute right-2 top-1/2 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white">
            <ChevronRight className="size-5" aria-hidden="true" />
          </button>
        )}
      </div>
      <div className="flex flex-col gap-2 p-4 text-sm">
        <p className="text-muted-foreground">Foto: {foto.credito}</p>
        {foto.marcados.length > 0 && (
          <p className="text-foreground">
            Na foto:{' '}
            {foto.marcados.map((m, i) => (
              <span key={m.slug}>
                {i > 0 && ', '}
                <Link to={`/t/${m.slug}`} className="underline underline-offset-4">
                  {m.nome}
                </Link>
              </span>
            ))}
          </p>
        )}
        <div className="flex flex-wrap gap-2">
          {logado && (
            <Button size="sm" variant={marcada ? 'outline' : 'default'} onClick={marcar}>
              {marcada ? 'Tirar minha marcação' : 'Sou eu nesta foto'}
            </Button>
          )}
          {!remover && (
            <Button size="sm" variant="ghost" onClick={() => setRemover(true)}>
              Não quero aparecer
            </Button>
          )}
        </div>
        {remover && (
          <div className="flex flex-col gap-2 rounded-lg border border-border p-3">
            <label htmlFor="motivo-remocao" className="text-foreground">
              A foto sai da galeria agora e o organizador revisa. Quer contar por quê? (opcional)
            </label>
            <textarea
              id="motivo-remocao"
              rows={2}
              maxLength={500}
              value={motivo}
              onChange={(e) => setMotivo(e.target.value)}
              className="rounded-lg border border-border bg-background px-2.5 py-2"
            />
            <div className="flex gap-2">
              <Button size="sm" variant="destructive" onClick={confirmarRemocao}>
                Tirar a foto da galeria
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setRemover(false)}>
                Cancelar
              </Button>
            </div>
          </div>
        )}
        {msg && (
          <p role="alert" className="text-destructive">
            {msg}{' '}
            {msg.includes('perfil de talento') && (
              <Link to="/meu-talento" className="underline underline-offset-4">
                Criar perfil
              </Link>
            )}
          </p>
        )}
      </div>
    </dialog>
  )
}
