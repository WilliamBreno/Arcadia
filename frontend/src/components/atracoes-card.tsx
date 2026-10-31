import { useQuery, useQueryClient } from '@tanstack/react-query'
import { BadgeCheck, Search, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { api, ApiError } from '@/lib/api'
import { nomeTipoArtista, type TipoArtista } from '@/lib/artistas'

export type AtracaoEvento = {
  grupo_id: number
  slug: string
  nome: string
  tipo: TipoArtista
  solo: boolean
  logo_url: string
  instagram: string
  verificado: boolean
  status: '' | 'convidado' | 'confirmado' | 'recusado'
}

const rotulo = { convidado: 'Aguardando o grupo confirmar', confirmado: 'Confirmado', recusado: 'Recusou', '': '' }

function Avatar({ a }: { a: AtracaoEvento }) {
  return (
    <span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted text-sm font-semibold text-foreground">
      {a.logo_url ? <img src={a.logo_url} alt="" className="size-full object-cover" /> : a.nome.slice(0, 1).toUpperCase()}
    </span>
  )
}

// Item 11.1: o organizador convida grupos e artistas para as atrações do
// evento; só aparecem na página depois que o líder confirma.
export function AtracoesCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const chave = ['org-atracoes', eventoId]
  const { data } = useQuery({ queryKey: chave, queryFn: () => api<AtracaoEvento[]>(`/org/eventos/${eventoId}/atracoes`) })
  const [termo, setTermo] = useState('')
  const [busca, setBusca] = useState('')
  const [erro, setErro] = useState<string | null>(null)

  useEffect(() => {
    const t = setTimeout(() => setBusca(termo.trim()), 300)
    return () => clearTimeout(t)
  }, [termo])
  const resultados = useQuery({
    queryKey: ['org-busca-artistas', busca],
    enabled: busca.length >= 2,
    queryFn: () => api<AtracaoEvento[]>(`/org/artistas/busca?q=${encodeURIComponent(busca)}`),
  })

  const convidar = async (slug: string) => {
    setErro(null)
    try {
      queryClient.setQueryData(chave, await api<AtracaoEvento[]>(`/org/eventos/${eventoId}/atracoes`, { method: 'POST', body: { grupo_slug: slug } }))
      setTermo('')
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível convidar.')
    }
  }
  const remover = async (grupoId: number) => {
    setErro(null)
    try {
      queryClient.setQueryData(chave, await api<AtracaoEvento[]>(`/org/eventos/${eventoId}/atracoes/${grupoId}`, { method: 'DELETE' }))
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível tirar.')
    }
  }

  const jaNaLista = new Set(data?.map((a) => a.grupo_id))

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Atrações</CardTitle>
        <p className="text-sm text-muted-foreground">
          Convide grupos e artistas da{' '}
          <Link to="/artistas" className="underline underline-offset-4">
            vitrine de artistas
          </Link>
          . Eles aparecem na página do evento depois que o líder confirma.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {data && data.length > 0 && (
          <ul className="flex flex-col divide-y divide-border rounded-xl border border-border">
            {data.map((a) => (
              <li key={a.grupo_id} className="flex items-center justify-between gap-3 px-3 py-2.5">
                <div className="flex min-w-0 items-center gap-3">
                  <Avatar a={a} />
                  <div className="min-w-0">
                    <Link to={`/artista/${a.slug}`} className="flex items-center gap-1 truncate text-sm font-medium text-foreground underline-offset-4 hover:underline">
                      {a.nome}
                      {a.verificado && <BadgeCheck className="size-4 shrink-0" aria-label="Instagram verificado" />}
                    </Link>
                    <p className="text-xs text-muted-foreground">
                      {nomeTipoArtista(a.tipo)} · {rotulo[a.status]}
                    </p>
                  </div>
                </div>
                <Button size="sm" variant="ghost" aria-label={`Tirar ${a.nome}`} onClick={() => remover(a.grupo_id)}>
                  <X className="size-4" aria-hidden="true" />
                </Button>
              </li>
            ))}
          </ul>
        )}
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
          <Input className="pl-8" placeholder="Buscar grupo ou artista pelo nome ou @" aria-label="Buscar artista" value={termo} onChange={(e) => setTermo(e.target.value)} />
        </div>
        {busca.length >= 2 && resultados.data && (
          <ul className="flex flex-col gap-1">
            {resultados.data.length === 0 && <li className="text-sm text-muted-foreground">Nenhum artista com esse nome.</li>}
            {resultados.data.map((a) => (
              <li key={a.grupo_id} className="flex items-center justify-between gap-3 rounded-lg px-2 py-1.5 hover:bg-muted">
                <span className="flex min-w-0 items-center gap-3">
                  <Avatar a={a} />
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium text-foreground">{a.nome}</span>
                    <span className="block text-xs text-muted-foreground">
                      {a.solo ? 'Solo · ' : ''}
                      {nomeTipoArtista(a.tipo)}
                      {a.instagram && ` · @${a.instagram}`}
                    </span>
                  </span>
                </span>
                <Button size="sm" variant="outline" disabled={jaNaLista.has(a.grupo_id)} onClick={() => convidar(a.slug)}>
                  {jaNaLista.has(a.grupo_id) ? 'Convidado' : 'Convidar'}
                </Button>
              </li>
            ))}
          </ul>
        )}
        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

// Item 11.1: atrações confirmadas na página do evento.
export function AtracoesDoEvento({ slug }: { slug: string }) {
  const { data } = useQuery({
    queryKey: ['atracoes-evento', slug],
    queryFn: () => api<AtracaoEvento[]>(`/eventos/${slug}/atracoes`, { semAuth: true }),
    retry: false,
  })
  if (!data || data.length === 0) return null
  return (
    <section aria-labelledby="titulo-atracoes" className="mt-8">
      <h2 id="titulo-atracoes" className="mb-3 text-xl font-semibold text-foreground">
        Atrações
      </h2>
      <ul className="-mx-4 flex gap-4 overflow-x-auto px-4 pb-2">
        {data.map((a) => (
          <li key={a.grupo_id} className="w-28 shrink-0">
            <Link to={`/artista/${a.slug}`} className="group flex flex-col items-center gap-2 text-center">
              <span className="flex size-24 items-center justify-center overflow-hidden rounded-full bg-muted text-2xl font-semibold text-foreground transition-transform group-hover:scale-105 motion-reduce:transition-none">
                {a.logo_url ? <img src={a.logo_url} alt="" className="size-full object-cover" loading="lazy" /> : a.nome.slice(0, 1).toUpperCase()}
              </span>
              <span className="flex items-center gap-1 text-sm font-medium text-foreground">
                <span className="line-clamp-2">{a.nome}</span>
                {a.verificado && <BadgeCheck className="size-3.5 shrink-0" aria-label="Instagram verificado" />}
              </span>
              <span className="text-xs text-muted-foreground">{nomeTipoArtista(a.tipo)}</span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
