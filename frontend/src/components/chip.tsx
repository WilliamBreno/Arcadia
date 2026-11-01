import { Link } from 'react-router-dom'

// Filtros e atalhos seguem a mesma hierarquia visual dos controles.
const base = 'inline-flex min-h-10 shrink-0 items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-ring'
const ativo = 'border-primary bg-primary text-primary-foreground'
const inativo = 'border-border bg-card text-muted-foreground hover:border-primary/40 hover:bg-accent hover:text-accent-foreground'

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
