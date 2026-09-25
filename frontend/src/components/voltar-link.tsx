import { ArrowLeft } from 'lucide-react'
import { Link } from 'react-router-dom'

export function VoltarLink({ to, children }: { to: string; children: string }) {
  return (
    <Link to={to} className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
      <ArrowLeft className="size-4" aria-hidden="true" />
      {children}
    </Link>
  )
}
