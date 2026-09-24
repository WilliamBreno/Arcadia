import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

import { api } from '@/lib/api'
import type { ListaEventosResposta } from '@/lib/publico'
import { EventoCard } from '@/components/evento-card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const NOME_PLATAFORMA = import.meta.env.VITE_NOME_PLATAFORMA ?? 'Evve'

const atalhos = [
  { valor: '', rotulo: 'Todos' },
  { valor: 'hoje', rotulo: 'Hoje' },
  { valor: 'amanha', rotulo: 'Amanhã' },
  { valor: 'fim-de-semana', rotulo: 'Fim de semana' },
]

export default function Home() {
  const [busca, setBusca] = useState('')
  const [buscaAplicada, setBuscaAplicada] = useState('')
  const [atalho, setAtalho] = useState('')
  const [gratuito, setGratuito] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['eventos-publicos', buscaAplicada, atalho, gratuito],
    queryFn: () => {
      const params = new URLSearchParams()
      if (buscaAplicada) params.set('q', buscaAplicada)
      if (atalho) params.set('atalho', atalho)
      if (gratuito) params.set('gratuito', 'true')
      return api<ListaEventosResposta>(`/eventos?${params.toString()}`)
    },
  })

  return (
    <main className="mx-auto max-w-5xl px-4 py-12">
      <section className="mb-10 flex flex-col items-center gap-4 text-center">
        <h1 className="sr-only">{NOME_PLATAFORMA}</h1>
        <img src="/brand/logo.png" alt="" aria-hidden="true" className="h-16 w-auto dark:hidden" />
        <img src="/brand/logo-dark.png" alt="" aria-hidden="true" className="hidden h-16 w-auto dark:block" />
        <p className="max-w-md text-muted-foreground">
          Encontre eventos, compre ingressos e participe de concursos de cosplay, dança, canto e atuação.
        </p>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            setBuscaAplicada(busca)
          }}
          className="flex w-full max-w-md gap-2"
        >
          <Input
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
            placeholder="Buscar eventos…"
            aria-label="Buscar eventos"
          />
          <Button type="submit">Buscar</Button>
        </form>
      </section>

      <section className="mb-6 flex flex-wrap items-center gap-2">
        {atalhos.map((a) => (
          <button
            key={a.valor}
            type="button"
            onClick={() => setAtalho(a.valor)}
            className={`rounded-full border px-3 py-1 text-sm transition-colors ${
              atalho === a.valor
                ? 'border-primary bg-primary text-primary-foreground'
                : 'border-border text-muted-foreground hover:text-foreground'
            }`}
          >
            {a.rotulo}
          </button>
        ))}
        <button
          type="button"
          onClick={() => setGratuito((v) => !v)}
          className={`rounded-full border px-3 py-1 text-sm transition-colors ${
            gratuito ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-muted-foreground hover:text-foreground'
          }`}
        >
          Gratuitos
        </button>
      </section>

      {isLoading && <p className="text-muted-foreground">Carregando…</p>}
      {data?.eventos.length === 0 && (
        <p className="text-muted-foreground">Nenhum evento encontrado com esses filtros.</p>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
        {data?.eventos.map((evento) => (
          <EventoCard key={evento.slug} evento={evento} />
        ))}
      </div>
    </main>
  )
}
