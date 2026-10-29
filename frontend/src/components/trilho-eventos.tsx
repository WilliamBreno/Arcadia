import { Link } from 'react-router-dom'

import type { EventoPublicoItem } from '@/lib/publico'
import { EventoCard } from '@/components/evento-card'

type Props = {
  titulo: string
  eventos: EventoPublicoItem[]
  subtitulo?: string
  verTudo?: string // link "Ver tudo"
  ranking?: boolean // número grande ao lado de cada card ("Em alta")
  etiqueta?: string
}

// Fileira horizontal de eventos (item 6.2). Some sozinha quando vazia, para
// a página não mostrar seções sem conteúdo.
export function TrilhoEventos({ titulo, eventos, subtitulo, verTudo, ranking, etiqueta }: Props) {
  if (eventos.length === 0) return null
  return (
    <section className="mb-10" aria-label={titulo}>
      <div className="mb-3 flex items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-foreground text-balance">{titulo}</h2>
          {subtitulo && <p className="text-sm text-muted-foreground">{subtitulo}</p>}
        </div>
        {verTudo && (
          <Link to={verTudo} className="shrink-0 text-sm font-medium text-foreground underline-offset-4 hover:underline">
            Ver tudo
          </Link>
        )}
      </div>
      <ol className="-mx-4 flex snap-x snap-mandatory gap-4 overflow-x-auto px-4 pb-2" style={{ scrollbarWidth: 'thin' }}>
        {eventos.map((e, i) => (
          <li key={e.slug} className="flex shrink-0 snap-start items-end gap-1">
            {ranking && (
              <span
                aria-hidden="true"
                className="select-none text-7xl leading-none font-black tabular-nums text-foreground/90"
                style={{ letterSpacing: '-0.06em' }}
              >
                {i + 1}
              </span>
            )}
            <div className="w-64">
              <EventoCard evento={e} etiqueta={etiqueta} />
            </div>
          </li>
        ))}
      </ol>
    </section>
  )
}
