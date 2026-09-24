import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { api } from '@/lib/api'
import type { Evento } from '@/lib/evento'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

const rotuloStatus: Record<Evento['status'], string> = {
  rascunho: 'Rascunho',
  publicado: 'Publicado',
  encerrado: 'Encerrado',
  cancelado: 'Cancelado',
}

export default function Eventos() {
  const { data: eventos, isLoading } = useQuery({
    queryKey: ['org-eventos'],
    queryFn: () => api<Evento[]>('/org/eventos'),
  })

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Meus eventos</h1>
        <div className="flex gap-2">
          <Link to="/organizador/repasses" className={buttonVariants({ variant: 'outline' })}>
            Repasses
          </Link>
          <Link to="/organizador/eventos/novo" className={buttonVariants()}>
            Novo evento
          </Link>
        </div>
      </div>

      {isLoading && <p className="text-muted-foreground">Carregando…</p>}
      {eventos?.length === 0 && <p className="text-muted-foreground">Nenhum evento criado ainda.</p>}

      <div className="flex flex-col gap-3">
        {eventos?.map((evento) => (
          <Link key={evento.id} to={`/organizador/eventos/${evento.id}`}>
            <Card className="transition-colors hover:border-foreground/20">
              <CardContent className="flex items-center justify-between py-4">
                <div>
                  <p className="font-medium text-foreground">{evento.titulo}</p>
                  <p className="text-sm text-muted-foreground">
                    {evento.inicio_em ? new Date(evento.inicio_em).toLocaleDateString('pt-BR') : 'Sem data'}
                  </p>
                </div>
                <span className="rounded-full border border-border px-2.5 py-0.5 text-xs text-muted-foreground">
                  {rotuloStatus[evento.status]}
                </span>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </main>
  )
}
