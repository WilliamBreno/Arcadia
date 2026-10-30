import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { aoMudarEscolha, lerEscolha, salvarEscolha } from '@/lib/consentimento'
import { Button } from '@/components/ui/button'

// Aviso de cookies (item 7.5). Aceitar e recusar têm o mesmo peso; sem
// escolha, nada de marketing carrega. Só os pixels de anúncio dos
// organizadores dependem disso: o resto do site funciona igual.
export function BannerCookies() {
  const [aberto, setAberto] = useState(() => lerEscolha() === null)
  useEffect(() => aoMudarEscolha(() => setAberto(false)), [])
  if (!aberto) return null
  return (
    <div
      role="region"
      aria-label="Aviso de cookies"
      className="fixed inset-x-0 bottom-0 z-40 border-t border-border bg-background"
    >
      <div className="mx-auto flex max-w-5xl flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-foreground">
          Alguns organizadores medem seus anúncios com pixels (Meta, Google, TikTok) nas páginas dos eventos deles. Só ligamos se
          você aceitar.{' '}
          <Link to="/cookies" className="underline underline-offset-4">
            Saiba mais
          </Link>
        </p>
        <div className="flex shrink-0 gap-2">
          <Button variant="outline" onClick={() => salvarEscolha(false)}>
            Recusar
          </Button>
          <Button variant="outline" onClick={() => salvarEscolha(true)}>
            Aceitar
          </Button>
        </div>
      </div>
    </div>
  )
}
