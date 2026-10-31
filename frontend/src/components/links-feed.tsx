import { CalendarPlus, Rss } from 'lucide-react'

import { urlApi } from '@/lib/api'

// Item 10.5: assinar os próximos eventos no calendário (ICS) ou num leitor RSS.
export function LinksFeed({ caminho, className = '' }: { caminho: string; className?: string }) {
  const absoluto = (formato: 'ics' | 'rss') => new URL(urlApi(`${caminho}/eventos.${formato}`), window.location.origin).toString()
  const webcal = absoluto('ics').replace(/^https?:/, 'webcal:')
  return (
    <p className={`flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground ${className}`}>
      <a href={webcal} className="inline-flex items-center gap-1 hover:text-foreground">
        <CalendarPlus className="size-4" aria-hidden="true" />
        Assinar no calendário
      </a>
      <a href={absoluto('rss')} className="inline-flex items-center gap-1 hover:text-foreground">
        <Rss className="size-4" aria-hidden="true" />
        RSS
      </a>
    </p>
  )
}
