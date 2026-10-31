import { useQuery } from '@tanstack/react-query'
import { AtSign, BadgeCheck, CalendarDays, Medal, Play, Users } from 'lucide-react'
import { useEffect } from 'react'
import { Link, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { VoltarBotao } from '@/components/voltar-link'
import { ApiError } from '@/lib/api'
import { obterArtista, textoAnos, type ApresentacaoArtista, type Midia } from '@/lib/artistas'

const dataCurta = (iso: string | null) =>
  iso ? new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' }).replace('.', '') : ''

function iniciais(nome: string) {
  return nome
    .split(/\s+/)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase() ?? '')
    .join('')
}

function ItemMidia({ m }: { m: Midia }) {
  if (m.tipo === 'foto') {
    return (
      <figure className="overflow-hidden rounded-xl bg-muted">
        <img src={m.url} alt={m.legenda || 'Foto de apresentação'} loading="lazy" className="aspect-square w-full object-cover" />
        {m.legenda && <figcaption className="px-3 py-2 text-xs text-muted-foreground">{m.legenda}</figcaption>}
      </figure>
    )
  }
  if (m.embed) {
    return (
      <figure className="overflow-hidden rounded-xl bg-muted sm:col-span-2">
        <iframe
          src={m.embed}
          title={m.legenda || 'Vídeo de apresentação'}
          loading="lazy"
          allow="accelerometer; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
          className="aspect-video w-full"
        />
        {m.legenda && <figcaption className="px-3 py-2 text-xs text-muted-foreground">{m.legenda}</figcaption>}
      </figure>
    )
  }
  return (
    <a
      href={m.url}
      target="_blank"
      rel="noopener noreferrer"
      className="flex aspect-square flex-col items-center justify-center gap-2 rounded-xl border border-border bg-muted p-4 text-center transition-colors hover:bg-accent"
    >
      <span className="flex size-12 items-center justify-center rounded-full bg-foreground text-background">
        <Play className="size-5" aria-hidden="true" />
      </span>
      <span className="text-sm font-medium text-foreground">Ver vídeo no {m.plataforma === 'tiktok' ? 'TikTok' : 'Instagram'}</span>
      {m.legenda && <span className="text-xs text-muted-foreground">{m.legenda}</span>}
    </a>
  )
}

function ListaApresentacoes({ itens }: { itens: ApresentacaoArtista[] }) {
  return (
    <ul className="flex flex-col divide-y divide-border rounded-xl border border-border">
      {itens.map((a) => (
        <li key={a.id}>
          <Link to={`/e/${a.evento_slug}`} className="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-muted">
            <CalendarDays className="size-5 shrink-0 text-muted-foreground" aria-hidden="true" />
            <span className="min-w-0 flex-1">
              <span className="block truncate font-medium text-foreground">{a.evento_titulo}</span>
              <span className="block text-sm text-muted-foreground">
                {dataCurta(a.evento_inicio)}
                {a.cidade && ` · ${a.cidade}/${a.uf}`}
              </span>
            </span>
          </Link>
        </li>
      ))}
    </ul>
  )
}

// Item 11.1: perfil público de grupo ou artista solo (/artista/:slug).
export default function ArtistaPublico() {
  const { slug = '' } = useParams()
  const { data, isLoading, error } = useQuery({ queryKey: ['artista', slug], queryFn: () => obterArtista(slug), retry: false })

  useEffect(() => {
    if (!data) return
    document.title = `${data.nome} | Evve`
    let meta = document.querySelector<HTMLMetaElement>('meta[name="robots"]')
    if (!data.indexar) {
      if (!meta) {
        meta = document.createElement('meta')
        meta.name = 'robots'
        document.head.appendChild(meta)
      }
      meta.content = 'noindex'
    }
    return () => {
      if (meta && !data.indexar) meta.content = 'index, follow'
    }
  }, [data])

  if (isLoading) return <Carregando />
  if (error instanceof ApiError || !data) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-foreground">Perfil não encontrado</h1>
        <p className="mt-2 text-muted-foreground">O link pode estar errado ou o perfil foi ocultado.</p>
        <Link to="/artistas" className="mt-6 inline-block text-sm font-medium underline underline-offset-4">
          Ver artistas
        </Link>
      </main>
    )
  }

  const totalIntegrantes = data.integrantes.length + data.outros_integrantes

  return (
    <main className="mx-auto max-w-5xl pb-16">
      <div className="relative">
        <div className="aspect-video max-h-80 w-full overflow-hidden bg-muted sm:rounded-b-2xl">
          {data.capa_url && <img src={data.capa_url} alt="" className="size-full object-cover" />}
        </div>
        <div className="absolute left-4 top-4 rounded-full bg-background/90 px-3 py-1 shadow-sm">
          <VoltarBotao fallback="/artistas" />
        </div>
      </div>

      <header className="relative z-10 px-4">
        <div className="-mt-12 flex flex-col gap-4 sm:-mt-14 sm:flex-row sm:items-end sm:justify-between">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:gap-4">
            <span className="flex size-24 shrink-0 items-center justify-center overflow-hidden rounded-full border-4 border-background bg-muted text-2xl font-bold text-foreground sm:size-28">
              {data.logo_url ? <img src={data.logo_url} alt="" className="size-full object-cover" /> : iniciais(data.nome)}
            </span>
            <div className="sm:pb-1">
              <p className="text-sm text-muted-foreground">
                {data.solo ? `Artista solo · ${data.tipo_nome}` : `Grupo de ${data.tipo_nome.toLowerCase()}`}
                {data.cidade && ` · ${data.cidade}/${data.uf}`}
              </p>
              <h1 className="flex items-center gap-2 text-2xl font-bold text-foreground text-balance sm:text-3xl">
                {data.nome}
                {data.verificado && <BadgeCheck className="size-6 shrink-0 text-foreground" aria-label="Instagram verificado pela Evve" />}
              </h1>
            </div>
          </div>
          {data.instagram && (
            <a
              href={`https://www.instagram.com/${data.instagram}/`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center gap-2 self-start rounded-full bg-foreground px-5 py-2.5 text-sm font-semibold text-background transition-opacity hover:opacity-90 sm:self-auto"
            >
              <AtSign className="size-4" aria-hidden="true" />@{data.instagram}
            </a>
          )}
        </div>
        {(data.tiktok || data.youtube) && (
          <p className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
            {data.tiktok && (
              <a href={`https://www.tiktok.com/@${data.tiktok}`} target="_blank" rel="noopener noreferrer" className="hover:text-foreground">
                TikTok @{data.tiktok}
              </a>
            )}
            {data.youtube && (
              <a href={`https://www.youtube.com/@${data.youtube}`} target="_blank" rel="noopener noreferrer" className="hover:text-foreground">
                YouTube @{data.youtube}
              </a>
            )}
          </p>
        )}
        {data.bio && <p className="mt-4 max-w-2xl whitespace-pre-line text-foreground text-pretty">{data.bio}</p>}
      </header>

      {totalIntegrantes > 0 && (
        <section aria-labelledby="titulo-integrantes" className="mt-10 px-4">
          <h2 id="titulo-integrantes" className="flex items-center gap-2 text-lg font-semibold text-foreground">
            <Users className="size-5" aria-hidden="true" />
            {data.solo ? 'Quem é' : `Integrantes (${totalIntegrantes})`}
          </h2>
          <ul className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
            {data.integrantes.map((i, n) => (
              <li key={`${i.nome_artistico}-${n}`} className="flex flex-col items-center rounded-2xl border border-border p-4 text-center">
                <span className="flex size-20 items-center justify-center overflow-hidden rounded-full bg-muted text-lg font-semibold text-foreground">
                  {i.foto_url ? <img src={i.foto_url} alt="" className="size-full object-cover" loading="lazy" /> : iniciais(i.nome_artistico)}
                </span>
                <span className="mt-3 font-medium text-foreground">{i.nome_artistico}</span>
                {i.funcao && <span className="text-sm text-muted-foreground">{i.funcao}</span>}
                {i.anos > 0 && <span className="mt-1 text-xs text-muted-foreground">{textoAnos(i.anos, data.tipo)}</span>}
                {i.instagram && (
                  <a
                    href={`https://www.instagram.com/${i.instagram}/`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-2 inline-flex items-center gap-1 text-xs font-medium text-foreground underline-offset-4 hover:underline"
                  >
                    <AtSign className="size-3.5" aria-hidden="true" />@{i.instagram}
                  </a>
                )}
              </li>
            ))}
          </ul>
          {data.outros_integrantes > 0 && (
            <p className="mt-3 text-sm text-muted-foreground">
              + {data.outros_integrantes} {data.outros_integrantes === 1 ? 'integrante' : 'integrantes'}
            </p>
          )}
        </section>
      )}

      {data.midias.length > 0 && (
        <section aria-labelledby="titulo-portfolio" className="mt-10 px-4">
          <h2 id="titulo-portfolio" className="text-lg font-semibold text-foreground">
            Apresentações
          </h2>
          <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
            {data.midias.map((m) => (
              <ItemMidia key={m.id} m={m} />
            ))}
          </div>
        </section>
      )}

      {data.proximas.length > 0 && (
        <section aria-labelledby="titulo-proximas" className="mt-10 px-4">
          <h2 id="titulo-proximas" className="text-lg font-semibold text-foreground">
            Próximas apresentações
          </h2>
          <div className="mt-4">
            <ListaApresentacoes itens={data.proximas} />
          </div>
        </section>
      )}

      {data.resultados.length > 0 && (
        <section aria-labelledby="titulo-resultados" className="mt-10 px-4">
          <h2 id="titulo-resultados" className="flex items-center gap-2 text-lg font-semibold text-foreground">
            <Medal className="size-5" aria-hidden="true" />
            Concursos
          </h2>
          <ul className="mt-4 flex flex-col divide-y divide-border rounded-xl border border-border">
            {data.resultados.map((r) => (
              <li key={r.evento_slug + r.tipo}>
                <Link to={`/e/${r.evento_slug}`} className="flex items-center justify-between gap-4 px-4 py-3 transition-colors hover:bg-muted">
                  <span className="min-w-0">
                    <span className="block truncate font-medium text-foreground">{r.evento_titulo}</span>
                    <span className="block text-sm text-muted-foreground">{dataCurta(r.data)}</span>
                  </span>
                  {r.posicao ? (
                    <span className="shrink-0 rounded-full bg-foreground px-3 py-1 text-sm font-semibold text-background">{r.posicao}º lugar</span>
                  ) : (
                    <span className="shrink-0 text-sm text-muted-foreground">Participou</span>
                  )}
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}

      {data.passadas.length > 0 && (
        <section aria-labelledby="titulo-passadas" className="mt-10 px-4">
          <h2 id="titulo-passadas" className="text-lg font-semibold text-foreground">
            Já se apresentou em
          </h2>
          <div className="mt-4">
            <ListaApresentacoes itens={data.passadas} />
          </div>
        </section>
      )}
    </main>
  )
}
