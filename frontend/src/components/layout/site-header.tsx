import { Link } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'
import { ThemeToggle } from '@/components/theme-toggle'
import { useAuth } from '@/hooks/use-auth'

const NOME_PLATAFORMA = import.meta.env.VITE_NOME_PLATAFORMA ?? 'Arcadia'

export function SiteHeader() {
  const { usuario, logout, carregando } = useAuth()

  return (
    <header className="border-border sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
        <Link to="/" className="text-lg font-semibold text-foreground">
          {NOME_PLATAFORMA}
        </Link>
        <nav className="flex items-center gap-2">
          {!carregando && usuario && (
            <>
              <Link to="/meus-ingressos" className="px-2 text-sm text-muted-foreground hover:text-foreground">
                Meus ingressos
              </Link>
              <Link to="/meus-eventos" className="px-2 text-sm text-muted-foreground hover:text-foreground">
                Meus eventos
              </Link>
              <Link to="/organizador/eventos" className="px-2 text-sm text-muted-foreground hover:text-foreground">
                Painel do organizador
              </Link>
            </>
          )}
          {!carregando && usuario ? (
            <Button variant="ghost" size="sm" onClick={() => logout()}>
              Sair
            </Button>
          ) : (
            !carregando && (
              <Link to="/login" className={buttonVariants({ variant: 'ghost', size: 'sm' })}>
                Entrar
              </Link>
            )
          )}
          <ThemeToggle />
        </nav>
      </div>
    </header>
  )
}
