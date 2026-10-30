import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { api } from '@/lib/api'
import type { MeuEvento } from '@/lib/conta'
import { Card, CardContent } from '@/components/ui/card'
import { useSeguindo } from '@/lib/seguir'
import { Carregando } from '@/components/carregando'

const rotuloSelo: Record<MeuEvento['selos'][number], string> = {
  organizador: 'Organizador',
  jurado: 'Jurado',
  participante: 'Participante',
  participante_especial: 'Participante especial',
  staff: 'Staff',
  ingresso: 'Ingresso',
}

const corSelo: Record<MeuEvento['selos'][number], string> = {
  organizador: 'bg-primary/10 text-primary',
  jurado: 'bg-accent text-accent-foreground',
  participante: 'bg-secondary text-secondary-foreground',
  participante_especial: 'bg-secondary text-secondary-foreground',
  staff: 'bg-muted text-muted-foreground',
  ingresso: 'bg-muted text-muted-foreground',
}

export default function MeusEventos() {
  const { data: eventos, isLoading } = useQuery({
    queryKey: ['meus-eventos'],
    queryFn: () => api<MeuEvento[]>('/me/eventos'),
  })

  if (isLoading) return <Carregando />

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <h1 className="mb-6 text-2xl font-semibold text-foreground">Meus eventos</h1>

      {/* Item 8.1: atalho para o perfil público de talento. */}
      <Link
        to="/meu-talento"
        className="mb-6 flex items-center justify-between gap-3 rounded-xl border border-border p-4 transition-colors hover:border-foreground/30"
      >
        <span>
          <span className="block font-medium text-foreground">Meu perfil de talento</span>
          <span className="block text-sm text-muted-foreground">Sua página com personagens e colocações para compartilhar.</span>
        </span>
        <span aria-hidden="true" className="text-muted-foreground">
          →
        </span>
      </Link>

      {eventos?.length === 0 && <p className="text-muted-foreground">Você ainda não tem nenhum papel em eventos.</p>}

      <div className="flex flex-col gap-3">
        {eventos?.map((e) => (
          <Link key={e.evento_id} to={`/e/${e.slug}`}>
            <Card className="transition-colors hover:border-foreground/20">
              <CardContent className="flex items-center justify-between py-4">
                <div>
                  <p className="font-medium text-foreground">{e.titulo}</p>
                  <p className="text-sm text-muted-foreground">
                    {e.inicio_em && new Date(e.inicio_em).toLocaleDateString('pt-BR')}
                  </p>
                </div>
                <div className="flex flex-wrap justify-end gap-1">
                  {e.selos.map((selo) => (
                    <span key={selo} className={`rounded-full px-2 py-0.5 text-xs ${corSelo[selo]}`}>
                      {rotuloSelo[selo]}
                    </span>
                  ))}
                </div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
      <VoceSegue />
    </main>
  )
}

// Quem a pessoa segue (item 6.4): cada um gera um e-mail a cada evento novo.
function VoceSegue() {
  const { lista, deixar } = useSeguindo()
  if (lista.length === 0) return null
  return (
    <section className="mt-12" aria-labelledby="titulo-segue">
      <h2 id="titulo-segue" className="text-xl font-semibold text-foreground">
        Você segue
      </h2>
      <p className="mb-4 text-sm text-muted-foreground">Você recebe um e-mail quando sai um evento novo de cada um.</p>
      <ul className="flex flex-col gap-2">
        {lista.map((s) => (
          <li key={s.tipo + s.alvo} className="flex items-center justify-between gap-3 rounded-lg bg-muted px-3 py-2">
            <span className="text-foreground">
              {s.rotulo || s.alvo} <span className="text-xs text-muted-foreground">({s.tipo === 'organizador' ? 'organizador' : 'categoria'})</span>
            </span>
            <button
              type="button"
              onClick={() => deixar.mutate({ tipo: s.tipo, alvo: s.alvo })}
              className="text-sm font-medium text-foreground underline underline-offset-4"
            >
              Deixar de seguir
            </button>
          </li>
        ))}
      </ul>
    </section>
  )
}
