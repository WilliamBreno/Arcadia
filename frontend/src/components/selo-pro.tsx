import { Sparkles } from 'lucide-react'

// Selo do organizador com Evve Pro (item 10.4).
export function SeloPro({ className = '' }: { className?: string }) {
  return (
    <span className={`inline-flex items-center gap-1 rounded-full bg-foreground px-2 py-0.5 text-xs font-medium text-background ${className}`} title="Organizador com Evve Pro">
      <Sparkles className="size-3.5" aria-hidden="true" />
      Pro
    </span>
  )
}
