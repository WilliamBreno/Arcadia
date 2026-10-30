import { useQuery } from '@tanstack/react-query'
import { Medal, Share2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { VoltarBotao } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { rotuloTipo, type PerfilPublico } from '@/lib/talento'
import { Button } from '@/components/ui/button'
import { Carregando } from '@/components/carregando'

const corMedalha: Record<number, string> = { 1: '#C9A227', 2: '#9AA3AD', 3: '#B0703A' }

// Perfil público do talento (item 8.1): /t/:slug. Perfil de menor chega da
// API só com o essencial e com "não indexar".
export default function PerfilTalento() {
  const { slug = '' } = useParams()
  const [copiado, setCopiado] = useState(false)
  const {
    data: p,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['talento', slug],
    queryFn: () => api<PerfilPublico>(`/talentos/${slug}`, { semAuth: true }),
    retry: false,
  })

  useEffect(() => {
    if (!p) return
    document.title = `${p.nome_artistico} | Evve`
    if (p.indexar) return
    const meta = document.createElement('meta')
    meta.name = 'robots'
    meta.content = 'noindex, nofollow'
    document.head.appendChild(meta)
    return () => meta.remove()
  }, [p])

  if (isLoading) return <Carregando />
  if (error instanceof ApiError || !p) {
    return (
      <main className="mx-auto max-w-xl px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-foreground">Perfil não encontrado</h1>
        <p className="mt-2 text-muted-foreground">Ele pode ter sido ocultado pela pessoa.</p>
      </main>
    )
  }

  const podios = p.participacoes.filter((x) => x.posicao && x.posicao <= 3).length
  const melhor = Math.min(...p.participacoes.map((x) => x.posicao ?? Infinity))
  const redes = [
    p.instagram && { nome: 'Instagram', url: `https://instagram.com/${p.instagram}` },
    p.tiktok && { nome: 'TikTok', url: `https://www.tiktok.com/@${p.tiktok}` },
    p.youtube && { nome: 'YouTube', url: `https://www.youtube.com/@${p.youtube}` },
  ].filter(Boolean) as { nome: string; url: string }[]

  const compartilhar = async () => {
    const url = window.location.href
    try {
      if (navigator.share) await navigator.share({ title: p.nome_artistico, url })
      else {
        await navigator.clipboard.writeText(url)
        setCopiado(true)
        window.setTimeout(() => setCopiado(false), 2000)
      }
    } catch {
      // a pessoa cancelou o compartilhamento
    }
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-10">
      <VoltarBotao fallback="/" />
      <header className="flex flex-col items-center gap-4 text-center sm:flex-row sm:items-end sm:text-left">
        {p.foto_url ? (
          <img src={p.foto_url} alt={`Foto de ${p.nome_artistico}`} className="size-32 shrink-0 rounded-full object-cover" />
        ) : (
          <div
            aria-hidden="true"
            className="flex size-32 shrink-0 items-center justify-center rounded-full bg-muted text-4xl font-bold text-foreground"
          >
            {p.nome_artistico.slice(0, 1).toUpperCase()}
          </div>
        )}
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <h1 className="text-3xl font-bold text-balance text-foreground sm:text-4xl">{p.nome_artistico}</h1>
          {p.cidade && (
            <p className="text-muted-foreground">
              {p.cidade}
              {p.uf ? `/${p.uf}` : ''}
            </p>
          )}
          <div className="mt-2 flex flex-wrap justify-center gap-2 sm:justify-start">
            {redes.map((r) => (
              <a
                key={r.nome}
                href={r.url}
                target="_blank"
                rel="noopener noreferrer"
                className="rounded-full border border-border px-3 py-1 text-sm text-foreground hover:bg-muted"
              >
                {r.nome}
              </a>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={compartilhar}>
              <Share2 className="size-4" aria-hidden="true" />
              {copiado ? 'Link copiado' : 'Compartilhar'}
            </Button>
          </div>
        </div>
      </header>

      {p.bio && <p className="mt-6 whitespace-pre-line text-pretty text-foreground">{p.bio}</p>}

      <dl className="mt-8 grid grid-cols-3 gap-3">
        {[
          ['Concursos', String(p.participacoes.length)],
          ['Pódios', String(podios)],
          ['Melhor colocação', Number.isFinite(melhor) ? `${melhor}º` : '—'],
        ].map(([rotulo, valor]) => (
          <div key={rotulo} className="rounded-xl border border-border p-3 text-center">
            <dt className="text-xs text-muted-foreground">{rotulo}</dt>
            <dd className="text-2xl font-semibold tabular-nums text-foreground">{valor}</dd>
          </div>
        ))}
      </dl>

      <section className="mt-8" aria-labelledby="titulo-concursos">
        <h2 id="titulo-concursos" className="mb-3 text-xl font-semibold text-foreground">
          Concursos
        </h2>
        {p.participacoes.length === 0 && <p className="text-muted-foreground">Nenhum concurso por aqui ainda.</p>}
        <ul className="flex flex-col gap-3">
          {p.participacoes.map((x) => (
            <li key={x.ficha_id} className="flex items-center gap-4 rounded-xl border border-border p-4">
              <div className="flex size-12 shrink-0 flex-col items-center justify-center" aria-hidden={!x.posicao}>
                {x.posicao && x.posicao <= 3 ? (
                  <Medal className="size-8" style={{ color: corMedalha[x.posicao] }} aria-hidden="true" />
                ) : null}
                {x.posicao ? <span className="text-sm font-semibold tabular-nums text-foreground">{x.posicao}º</span> : null}
              </div>
              <div className="min-w-0 flex-1">
                <p className="font-medium text-foreground">
                  {x.personagem || rotuloTipo[x.tipo] || 'Apresentação'}
                  {x.obra ? <span className="font-normal text-muted-foreground"> · {x.obra}</span> : null}
                </p>
                <Link to={`/e/${x.evento_slug}`} className="text-sm text-muted-foreground underline-offset-4 hover:underline">
                  {x.evento_titulo}
                </Link>
                <p className="text-xs text-muted-foreground">
                  {rotuloTipo[x.tipo] ?? ''}
                  {x.data ? ` · ${new Date(x.data).toLocaleDateString('pt-BR', { month: 'short', year: 'numeric' })}` : ''}
                  {x.posicao && x.total ? ` · ${x.posicao}º de ${x.total}` : ''}
                </p>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </main>
  )
}
