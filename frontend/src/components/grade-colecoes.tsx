import { Link } from 'react-router-dom'

import { IconeColecao } from '@/components/icone-colecao'
import type { ColecaoGrade } from '@/lib/descoberta'

// Item 11.7: atalho de coleção com ícone sobre um círculo claro.
export function AtalhoColecao({ colecao, compacto = false }: { colecao: ColecaoGrade; compacto?: boolean }) {
  return (
    <Link
      to={`/c/${colecao.slug}`}
      className="group flex flex-col items-center gap-2 rounded-2xl p-1 text-center outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <span
        className={`flex items-center justify-center rounded-full bg-muted text-foreground transition-transform duration-200 group-hover:scale-105 group-active:scale-95 motion-reduce:transition-none ${
          compacto ? 'size-20' : 'size-24 sm:size-28'
        }`}
      >
        <IconeColecao nome={colecao.icone} className={compacto ? 'size-10' : 'size-12 sm:size-14'} />
      </span>
      <span className={`font-medium text-foreground text-balance ${compacto ? 'text-xs' : 'text-sm'}`}>{colecao.titulo}</span>
    </Link>
  )
}

export function GradeColecoes({ colecoes }: { colecoes: ColecaoGrade[] }) {
  return (
    <ul className="grid grid-cols-3 gap-x-3 gap-y-6 sm:grid-cols-4 lg:grid-cols-6">
      {colecoes.map((c) => (
        <li key={c.slug}>
          <AtalhoColecao colecao={c} />
        </li>
      ))}
    </ul>
  )
}
