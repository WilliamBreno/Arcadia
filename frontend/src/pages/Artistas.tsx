import { useQuery } from '@tanstack/react-query'
import { BadgeCheck, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { api } from '@/lib/api'
import { nomeTipoArtista, tiposArtista, type ArtistaItem } from '@/lib/artistas'

const iniciais = (nome: string) =>
  nome
    .split(/\s+/)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase() ?? '')
    .join('')

// Item 11.1: vitrine de grupos e artistas solo (/artistas).
export default function Artistas() {
  const [params, setParams] = useSearchParams()
  const tipo = params.get('tipo') ?? ''
  const uf = params.get('uf') ?? ''
  const q = params.get('q') ?? ''
  const pagina = Number(params.get('pagina') ?? '1') || 1
  const [busca, setBusca] = useState(q)

  useEffect(() => {
    document.title = 'Artistas | Evve'
  }, [])

  const { data, isLoading } = useQuery({
    queryKey: ['artistas', tipo, uf, q, pagina],
    queryFn: () => {
      const s = new URLSearchParams()
      if (tipo) s.set('tipo', tipo)
      if (uf) s.set('uf', uf)
      if (q) s.set('q', q)
      if (pagina > 1) s.set('pagina', String(pagina))
      return api<{ itens: ArtistaItem[]; total: number }>(`/artistas?${s}`, { semAuth: true })
    },
  })

  const mudar = (chave: string, valor: string) => {
    const p = new URLSearchParams(params)
    if (valor) p.set(chave, valor)
    else p.delete(chave)
    p.delete('pagina')
    setParams(p)
  }

  return (
    <main className="mx-auto max-w-5xl px-4 py-10">
      <h1 className="text-3xl font-bold text-foreground sm:text-4xl">Artistas</h1>
      <p className="mt-2 max-w-2xl text-muted-foreground">Grupos de dança, cosplayers, cantores e companhias de teatro. Veja o trabalho e chame pelo Instagram.</p>

      <form
        className="mt-6 flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          mudar('q', busca.trim())
        }}
      >
        <Input value={busca} onChange={(e) => setBusca(e.target.value)} placeholder="Buscar pelo nome" aria-label="Buscar artistas" />
        <Button type="submit" aria-label="Buscar">
          <Search className="size-4" aria-hidden="true" />
        </Button>
      </form>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <div role="group" aria-label="Tipo" className="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1">
          {[{ valor: '', nome: 'Todos' }, ...tiposArtista].map((t) => (
            <button
              key={t.valor}
              type="button"
              aria-pressed={tipo === t.valor}
              onClick={() => mudar('tipo', t.valor)}
              className={`shrink-0 rounded-full border px-3 py-1.5 text-sm transition-colors ${
                tipo === t.valor ? 'border-foreground bg-foreground text-background' : 'border-border text-foreground hover:bg-muted'
              }`}
            >
              {t.nome}
            </button>
          ))}
        </div>
        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          UF
          <Input className="h-8 w-16 uppercase" maxLength={2} value={uf} onChange={(e) => mudar('uf', e.target.value.toUpperCase())} aria-label="Estado (UF)" />
        </label>
      </div>

      <div className="mt-8">
        {isLoading ? (
          <Carregando />
        ) : !data || data.itens.length === 0 ? (
          <p className="rounded-xl border border-border p-6 text-muted-foreground">Nenhum artista encontrado com esses filtros.</p>
        ) : (
          <>
            <p className="mb-3 text-sm text-muted-foreground">{data.total === 1 ? '1 artista' : `${data.total} artistas`}</p>
            <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {data.itens.map((a) => (
                <li key={a.slug}>
                  <Link to={`/artista/${a.slug}`} className="group block overflow-hidden rounded-2xl border border-border transition-colors hover:bg-muted">
                    <div className="aspect-video bg-muted">
                      {a.capa_url && <img src={a.capa_url} alt="" loading="lazy" className="size-full object-cover" />}
                    </div>
                    <div className="flex items-center gap-3 p-4">
                      <span className="-mt-10 flex size-14 shrink-0 items-center justify-center overflow-hidden rounded-full border-4 border-background bg-muted font-semibold text-foreground">
                        {a.logo_url ? <img src={a.logo_url} alt="" className="size-full object-cover" loading="lazy" /> : iniciais(a.nome)}
                      </span>
                      <span className="min-w-0">
                        <span className="flex items-center gap-1 font-semibold text-foreground">
                          <span className="truncate">{a.nome}</span>
                          {a.verificado && <BadgeCheck className="size-4 shrink-0" aria-label="Instagram verificado" />}
                        </span>
                        <span className="block truncate text-sm text-muted-foreground">
                          {a.solo ? `Solo · ${nomeTipoArtista(a.tipo)}` : `${nomeTipoArtista(a.tipo)} · ${a.integrantes} ${a.integrantes === 1 ? 'integrante' : 'integrantes'}`}
                          {a.uf && ` · ${a.cidade ? `${a.cidade}/` : ''}${a.uf}`}
                        </span>
                      </span>
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
            {(pagina > 1 || data.total > pagina * 24) && (
              <nav aria-label="Páginas" className="mt-6 flex items-center justify-center gap-3">
                <Button variant="outline" disabled={pagina <= 1} onClick={() => setParams((p) => (p.set('pagina', String(pagina - 1)), p))}>
                  Anterior
                </Button>
                <span className="text-sm text-muted-foreground">Página {pagina}</span>
                <Button variant="outline" disabled={data.total <= pagina * 24} onClick={() => setParams((p) => (p.set('pagina', String(pagina + 1)), p))}>
                  Próxima
                </Button>
              </nav>
            )}
          </>
        )}
      </div>

      <p className="mt-10 rounded-2xl bg-muted p-5 text-sm text-foreground">
        Tem um grupo ou faz cosplay?{' '}
        <Link to="/meus-artistas" className="font-medium underline underline-offset-4">
          Crie o seu perfil de artista
        </Link>{' '}
        e mostre o seu trabalho.
      </p>
    </main>
  )
}
