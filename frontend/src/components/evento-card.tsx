import { Link } from 'react-router-dom'

import { formatarCentavos } from '@/lib/evento'
import type { EventoPublicoItem } from '@/lib/publico'

type Props = {
  evento: EventoPublicoItem
  // Selo sobre a capa, ex.: "Quase esgotando" (item 6.2).
  etiqueta?: string
  // Texto extra no rodapé, ex.: "a 3 km" no mapa.
  detalhe?: string
}

export function EventoCard({ evento, etiqueta, detalhe }: Props) {
  const data = evento.inicio_em
    ? new Date(evento.inicio_em).toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short' })
    : null

  return (
    <Link
      to={`/e/${evento.slug}`}
      className="group flex flex-col overflow-hidden rounded-xl border border-border transition-colors hover:border-foreground/20 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      <div className="relative aspect-video w-full bg-muted">
        {evento.capa_url && (
          <img src={evento.capa_url} alt="" loading="lazy" decoding="async" className="h-full w-full object-cover" />
        )}
        {etiqueta && (
          <span className="absolute left-2 top-2 rounded-full bg-foreground px-2.5 py-0.5 text-xs font-medium text-background">
            {etiqueta}
          </span>
        )}
      </div>
      <div className="flex flex-1 flex-col gap-1 p-3">
        {data && <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{data}</p>}
        <p className="line-clamp-2 font-medium text-foreground">{evento.titulo}</p>
        {(evento.cidade || evento.uf) && (
          <p className="text-sm text-muted-foreground">
            {evento.cidade}
            {evento.cidade && evento.uf ? ' — ' : ''}
            {evento.uf}
          </p>
        )}
        <p className="mt-auto flex items-baseline justify-between gap-2 pt-2 text-sm font-medium text-foreground">
          <span className="flex flex-col">
            {evento.gratuito ? (
              'Gratuito'
            ) : evento.preco_a_partir_centavos != null ? (
              <>
                <span>A partir de {formatarCentavos(evento.preco_final_a_partir_centavos ?? evento.preco_a_partir_centavos)}</span>
                {evento.preco_final_a_partir_centavos != null && evento.preco_final_a_partir_centavos > evento.preco_a_partir_centavos && (
                  <span className="text-xs font-normal text-muted-foreground">
                    inclui {formatarCentavos(evento.preco_final_a_partir_centavos - evento.preco_a_partir_centavos)} de taxa
                  </span>
                )}
              </>
            ) : (
              ''
            )}
          </span>
          {detalhe && <span className="text-xs font-normal text-muted-foreground">{detalhe}</span>}
        </p>
      </div>
    </Link>
  )
}
