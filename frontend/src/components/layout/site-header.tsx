import { Link } from 'react-router-dom'

import { ThemeToggle } from '@/components/theme-toggle'

const NOME_PLATAFORMA = import.meta.env.VITE_NOME_PLATAFORMA ?? 'Arcadia'

export function SiteHeader() {
  return (
    <header className="border-border sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
        <Link to="/" className="text-lg font-semibold text-foreground">
          {NOME_PLATAFORMA}
        </Link>
        <ThemeToggle />
      </div>
    </header>
  )
}
