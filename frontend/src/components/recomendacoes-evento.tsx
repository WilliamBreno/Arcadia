import { useQuery } from '@tanstack/react-query'

import { TrilhoEventos } from '@/components/trilho-eventos'
import { api } from '@/lib/api'
import type { EventoPublicoItem } from '@/lib/publico'

type Recomendados = { tambem_compraram: EventoPublicoItem[]; parecidos: EventoPublicoItem[] }

// Item 10.5: no fim da página do evento, quem comprou este também comprou
// e eventos parecidos (mesma categoria e cidade).
export function RecomendacoesDoEvento({ slug }: { slug: string }) {
  const { data } = useQuery({
    queryKey: ['recomendados', slug],
    queryFn: () => api<Recomendados>(`/eventos/${slug}/recomendados`, { semAuth: true }),
    retry: false,
    staleTime: 5 * 60_000,
  })
  if (!data || (data.tambem_compraram.length === 0 && data.parecidos.length === 0)) return null
  return (
    <section aria-label="Outros eventos" className="mt-10">
      {data.tambem_compraram.length > 0 && <TrilhoEventos titulo="Quem comprou este também comprou" eventos={data.tambem_compraram} />}
      {data.parecidos.length > 0 && <TrilhoEventos titulo="Parecidos" eventos={data.parecidos} />}
    </section>
  )
}
