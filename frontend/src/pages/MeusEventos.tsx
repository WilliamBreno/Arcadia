import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { api } from '@/lib/api'
import type { MeuEvento } from '@/lib/conta'
import { Card, CardContent } from '@/components/ui/card'
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
    </main>
  )
}
