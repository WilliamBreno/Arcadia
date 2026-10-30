import { Link } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'
import { ThemeToggle } from '@/components/theme-toggle'
import { useAuth } from '@/hooks/use-auth'
import { useOrganizador } from '@/hooks/use-organizador'

const NOME_PLATAFORMA = import.meta.env.VITE_NOME_PLATAFORMA ?? 'Evve'

export function SiteHeader() {
  const { usuario, logout, carregando } = useAuth()
  const { organizador, carregando: carregandoOrg } = useOrganizador()

  return (
    <header className="border-border sticky top-0 z-10 border-b bg-background/80 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between gap-1 px-2.5 sm:px-4">
        <Link to="/" aria-label={NOME_PLATAFORMA} className="flex shrink-0 items-center">
          <img src="/brand/logo.png" alt={NOME_PLATAFORMA} className="h-6 w-auto sm:h-8 dark:hidden" />
          <img src="/brand/logo-dark.png" alt="" aria-hidden="true" className="hidden h-6 w-auto sm:h-8 dark:block" />
        </Link>
        <nav className="flex min-w-0 items-center gap-0.5 sm:gap-2">
          {!carregando && usuario && (
            <>
              <Link to="/meus-ingressos" className="whitespace-nowrap px-1 text-sm text-muted-foreground hover:text-foreground sm:px-2">
                <span className="hidden sm:inline">Meus ingressos</span>
                <span className="sm:hidden">Ingressos</span>
              </Link>
              <Link to="/meus-eventos" className="whitespace-nowrap px-1 text-sm text-muted-foreground hover:text-foreground sm:px-2">
                <span className="hidden sm:inline">Meus eventos</span>
                <span className="sm:hidden">Eventos</span>
              </Link>
              {organizador ? (
                <Link to="/organizador/eventos" className="whitespace-nowrap px-1 text-sm text-muted-foreground hover:text-foreground sm:px-2">
                  <span className="hidden sm:inline">Painel do organizador</span>
                <span className="sm:hidden">Painel</span>
                </Link>
              ) : (
                !carregandoOrg && (
                  <Link to="/organizador/perfil" className="whitespace-nowrap px-1 text-sm font-medium text-primary hover:underline sm:px-2">
                    <span className="hidden sm:inline">Seja um organizador</span>
                <span className="sm:hidden">Organizar</span>
                  </Link>
                )
              )}
            </>
          )}
          {!carregando && usuario ? (
            <Button variant="ghost" size="sm" className="px-1.5 sm:px-2.5" onClick={() => logout()}>
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
