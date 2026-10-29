import { useQuery } from '@tanstack/react-query'
import { MapPin } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/lib/api'
import type { ListaEventosResposta } from '@/lib/publico'
import { listarColecoes, obterDescoberta, obterDestaques } from '@/lib/descoberta'
import { ChipLink } from '@/components/chip'
import { EventoCard } from '@/components/evento-card'
import { TrilhoEventos } from '@/components/trilho-eventos'
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

  // Item 6.2: vitrine (cidades, destaques e coleções). Some durante a busca.
  const descoberta = useQuery({ queryKey: ['descoberta', ''], queryFn: () => obterDescoberta() })
  const destaques = useQuery({ queryKey: ['destaques', ''], queryFn: () => obterDestaques() })
  const colecoes = useQuery({ queryKey: ['colecoes'], queryFn: listarColecoes })
  const vitrine = !buscaAplicada

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

      <nav aria-label="Explorar" className="-mx-4 mb-10 flex gap-2 overflow-x-auto px-4 pb-1">
        <ChipLink to="/mapa">
          <MapPin className="size-3.5" aria-hidden="true" /> Perto de mim
        </ChipLink>
        {descoberta.data?.cidades.slice(0, 12).map((c) => (
          <ChipLink key={c.slug} to={`/eventos/${c.slug}`}>
            {c.cidade}/{c.uf}
          </ChipLink>
        ))}
      </nav>

      {vitrine && destaques.data && (
        <>
          <TrilhoEventos titulo="Em alta" subtitulo="Os mais comprados nos últimos 7 dias" eventos={destaques.data.em_alta} ranking />
          <TrilhoEventos titulo="Quase esgotando" eventos={destaques.data.quase_esgotando} etiqueta="Quase esgotando" />
        </>
      )}
      {vitrine &&
        colecoes.data?.map((c) => (
          <TrilhoEventos key={c.slug} titulo={c.titulo} subtitulo={c.descricao} eventos={c.eventos} verTudo={`/c/${c.slug}`} />
        ))}
      {vitrine && destaques.data && <TrilhoEventos titulo="Novos esta semana" eventos={destaques.data.novos} />}

      <h2 className="mb-3 text-xl font-bold text-foreground">{buscaAplicada ? `Resultados para “${buscaAplicada}”` : 'Todos os eventos'}</h2>
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
