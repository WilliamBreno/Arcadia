import { Link } from 'react-router-dom'

// Chip em pílula (filtros de categoria/data e atalhos de cidade). Preto
// quando ativo, contorno fino quando não — sem cor de destaque (item 6.2).
const base = 'inline-flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1 text-sm transition-colors'
const ativo = 'border-primary bg-primary text-primary-foreground'
const inativo = 'border-border text-muted-foreground hover:border-foreground/30 hover:text-foreground'

export function ChipLink({ to, ativoAgora, children }: { to: string; ativoAgora?: boolean; children: React.ReactNode }) {
  return (
    <Link to={to} aria-current={ativoAgora ? 'page' : undefined} className={`${base} ${ativoAgora ? ativo : inativo}`}>
      {children}
    </Link>
  )
}

export function ChipBotao({ ativoAgora, onClick, children }: { ativoAgora?: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button type="button" aria-pressed={ativoAgora} onClick={onClick} className={`${base} ${ativoAgora ? ativo : inativo}`}>
      {children}
    </button>
  )
}
