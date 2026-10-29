import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { Link, useParams } from 'react-router-dom'

import { ApiError } from '@/lib/api'
import { obterColecao } from '@/lib/descoberta'
import { EventoCard } from '@/components/evento-card'
import { Carregando } from '@/components/carregando'

// Coleção curada pelo admin (item 6.2): /c/:slug.
export default function ColecaoPublica() {
  const { slug = '' } = useParams()
  const { data, isLoading, error } = useQuery({ queryKey: ['colecao', slug], queryFn: () => obterColecao(slug) })

  useEffect(() => {
    if (data) document.title = `${data.titulo} | Evve`
  }, [data])

  if (isLoading) return <Carregando />
  if (error instanceof ApiError && error.status === 404) {
    return (
      <main className="mx-auto max-w-5xl px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-foreground">Coleção não encontrada</h1>
        <Link to="/" className="mt-6 inline-block text-sm font-medium underline underline-offset-4">
          Voltar para o início
        </Link>
      </main>
    )
  }
  if (!data) return null

  return (
    <main className="mx-auto max-w-5xl px-4 py-10">
      <p className="text-sm font-medium uppercase tracking-wide text-muted-foreground">Coleção</p>
      <h1 className="mt-1 text-3xl font-bold text-foreground text-balance sm:text-4xl">{data.titulo}</h1>
      {data.descricao && <p className="mt-2 max-w-2xl text-muted-foreground text-pretty">{data.descricao}</p>}
      {data.eventos.length === 0 && (
        <p className="mt-8 rounded-xl border border-border p-6 text-muted-foreground">Os eventos desta coleção já aconteceram.</p>
      )}
      <div className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
        {data.eventos.map((e) => (
          <EventoCard key={e.slug} evento={e} />
        ))}
      </div>
    </main>
  )
}
