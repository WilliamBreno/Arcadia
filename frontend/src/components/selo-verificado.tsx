import { BadgeCheck } from 'lucide-react'

// Selo do organizador verificado pelo admin (item 5.4): documento e chave
// Pix conferidos.
export function SeloVerificado({ className = '' }: { className?: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary ${className}`}
      title="Documento e dados de recebimento conferidos pela plataforma"
    >
      <BadgeCheck className="size-3.5" aria-hidden="true" />
      Verificado
    </span>
  )
}
