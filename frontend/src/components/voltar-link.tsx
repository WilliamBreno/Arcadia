import { ArrowLeft } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'

const classe =
  'mb-4 inline-flex min-h-10 items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring'

// Volta para uma tela fixa (a "tela de cima" desta).
export function VoltarLink({ to, children }: { to: string; children: string }) {
  return (
    <Link to={to} className={classe}>
      <ArrowLeft className="size-4" aria-hidden="true" />
      {children}
    </Link>
  )
}

// Volta para onde a pessoa estava. Se ela chegou direto (link compartilhado,
// e-mail, aba nova), não há histórico no site: vai para o `fallback`.
export function VoltarBotao({ fallback, rotulo = 'Voltar' }: { fallback: string; rotulo?: string }) {
  const navigate = useNavigate()
  const voltar = () => {
    const idx = (window.history.state as { idx?: number } | null)?.idx ?? 0
    if (idx > 0) navigate(-1)
    else navigate(fallback)
  }
  return (
    <button type="button" onClick={voltar} className={classe}>
      <ArrowLeft className="size-4" aria-hidden="true" />
      {rotulo}
    </button>
  )
}
