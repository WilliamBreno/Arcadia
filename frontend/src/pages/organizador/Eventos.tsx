import { useQuery } from '@tanstack/react-query'
import { Link, Navigate } from 'react-router-dom'

import { api } from '@/lib/api'
import { useOrganizador } from '@/hooks/use-organizador'
import { Carregando } from '@/components/carregando'
import type { Evento } from '@/lib/evento'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { AvisoPix } from '@/components/campos-pix'
import { MessageCircle, Plus } from 'lucide-react'

const rotuloStatus: Record<Evento['status'], string> = {
  rascunho: 'Rascunho',
  publicado: 'Publicado',
  encerrado: 'Encerrado',
  cancelado: 'Cancelado',
}

export default function Eventos() {
  const { organizador, carregando } = useOrganizador()
  const { data: eventos, isLoading } = useQuery({
    queryKey: ['org-eventos'],
    queryFn: () => api<Evento[]>('/org/eventos'),
    enabled: !!organizador,
  })

  if (carregando) return <Carregando />
  if (!organizador) return <Navigate to="/organizador/perfil" replace />

  return (
    <main className="mx-auto max-w-4xl px-4 py-10 sm:py-12">
      <AvisoPix />
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div><p className="mb-1 text-xs font-semibold uppercase tracking-widest text-primary">Área do organizador</p><h1 className="text-2xl font-semibold text-foreground">Meus eventos</h1><p className="mt-2 text-sm text-muted-foreground">Crie, acompanhe e prepare seu próximo evento.</p></div>
        <div className="evve-actions">
          <Link to="/conversas-equipe" className={buttonVariants({ variant: 'outline' })}><MessageCircle aria-hidden="true" />Chat da equipe</Link>
          <Link to="/organizador/eventos/novo" className={buttonVariants()}>
            <Plus aria-hidden="true" />Novo evento
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
