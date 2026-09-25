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
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
        <Link to="/" aria-label={NOME_PLATAFORMA} className="flex items-center">
          <img src="/brand/logo.png" alt={NOME_PLATAFORMA} className="h-8 w-auto dark:hidden" />
          <img src="/brand/logo-dark.png" alt="" aria-hidden="true" className="hidden h-8 w-auto dark:block" />
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
              {organizador ? (
                <Link to="/organizador/eventos" className="px-2 text-sm text-muted-foreground hover:text-foreground">
                  Painel do organizador
                </Link>
              ) : (
                !carregandoOrg && (
                  <Link to="/organizador/perfil" className="px-2 text-sm font-medium text-primary hover:underline">
                    Seja um organizador
                  </Link>
                )
              )}
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
