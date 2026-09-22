import { Link } from 'react-router-dom'

import { formatarCentavos } from '@/lib/evento'
import type { EventoPublicoItem } from '@/lib/publico'

export function EventoCard({ evento }: { evento: EventoPublicoItem }) {
  const data = evento.inicio_em
    ? new Date(evento.inicio_em).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })
    : null

  return (
    <Link
      to={`/e/${evento.slug}`}
      className="group flex flex-col overflow-hidden rounded-xl border border-border transition-colors hover:border-foreground/20"
    >
      <div className="aspect-video w-full bg-muted">
        {evento.capa_url && (
          <img src={evento.capa_url} alt="" className="h-full w-full object-cover" />
        )}
      </div>
      <div className="flex flex-1 flex-col gap-1 p-3">
        {data && <p className="text-xs font-medium text-primary">{data}</p>}
        <p className="line-clamp-2 font-medium text-foreground">{evento.titulo}</p>
        {(evento.cidade || evento.uf) && (
          <p className="text-sm text-muted-foreground">
            {evento.cidade}
            {evento.cidade && evento.uf ? ' — ' : ''}
            {evento.uf}
          </p>
        )}
        <p className="mt-auto pt-2 text-sm font-medium text-foreground">
          {evento.gratuito
            ? 'Gratuito'
            : evento.preco_a_partir_centavos != null
              ? `A partir de ${formatarCentavos(evento.preco_a_partir_centavos)}`
              : ''}
        </p>
      </div>
    </Link>
  )
}
