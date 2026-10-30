import { useQuery } from '@tanstack/react-query'
import { Trophy } from 'lucide-react'
import { Link } from 'react-router-dom'

import { circuitoDoEventoPublico } from '@/lib/circuito'

// Na página do evento (item 8.5): etapa de circuito e seletivas.
export function SelosCircuito({ slug }: { slug: string }) {
  const { data } = useQuery({
    queryKey: ['circuito-evento', slug],
    queryFn: () => circuitoDoEventoPublico(slug).catch(() => null),
  })
  if (!data) return null
  const finais = data.classifica_para ?? []
  const seletivas = data.seletivas ?? []
  if (!data.circuito && finais.length === 0 && seletivas.length === 0) return null
  return (
    <ul className="mt-4 flex flex-col gap-2 text-sm" aria-label="Circuito e seletivas">
      {data.circuito && (
        <li className="flex items-start gap-2 text-foreground">
          <Trophy className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <span>
            Etapa do{' '}
            <Link to={`/circuitos/${data.circuito.slug}`} className="font-medium underline underline-offset-4">
              {data.circuito.nome}
            </Link>
            : quem fica entre os 10 primeiros soma pontos no ranking.
          </span>
        </li>
      )}
      {finais.map((f) => (
        <li key={f.id} className="flex items-start gap-2 text-foreground">
          <Trophy className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <span>
            Seletiva para{' '}
            <Link to={`/e/${f.slug}`} className="font-medium underline underline-offset-4">
              {f.titulo}
            </Link>
            : {f.vagas === 1 ? 'o 1º lugar' : `os ${f.vagas} primeiros`} de cada categoria vão direto para a final.
          </span>
        </li>
      ))}
      {seletivas.length > 0 && (
        <li className="flex items-start gap-2 text-foreground">
          <Trophy className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <span>
            Final com classificados de{' '}
            {seletivas.map((s, i) => (
              <span key={s.id}>
                {i > 0 && (i === seletivas.length - 1 ? ' e ' : ', ')}
                <Link to={`/e/${s.slug}`} className="underline underline-offset-4">
                  {s.titulo}
                </Link>
              </span>
            ))}
            .
          </span>
        </li>
      )}
    </ul>
  )
}
