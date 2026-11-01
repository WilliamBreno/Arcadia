import { Dialog } from '@base-ui/react/dialog'
import { ChevronDown, Compass, LayoutDashboard, LogOut, Menu as MenuIcon, MessageCircle, Palette, ShieldCheck, Sparkles, Ticket, UserRound, Users, X } from 'lucide-react'
import { useState } from 'react'
import { Link, NavLink } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'
import { MenuAction, MenuContent, MenuLabel, MenuLink, MenuRoot, MenuSeparator, MenuTrigger } from '@/components/ui/menu'
import { ThemeToggle } from '@/components/theme-toggle'
import { useAuth } from '@/hooks/use-auth'
import { useOrganizador } from '@/hooks/use-organizador'

const NOME_PLATAFORMA = import.meta.env.VITE_NOME_PLATAFORMA ?? 'Evve'

export function SiteHeader() {
  const { usuario, logout, carregando } = useAuth()
  const { organizador, carregando: carregandoOrg } = useOrganizador()
  const [aberto, setAberto] = useState(false)
  const destinoOrganizador = organizador ? '/organizador/eventos' : '/organizador/perfil'
  const links = [
    { to: '/', nome: 'Explorar eventos', icon: Compass },
    { to: '/artistas', nome: 'Artistas', icon: Sparkles },
    ...(usuario ? [
      { to: '/meus-ingressos', nome: 'Meus ingressos', icon: Ticket },
      { to: '/meus-eventos', nome: 'Meus eventos', icon: LayoutDashboard },
      { to: '/meus-artistas', nome: 'Meus artistas', icon: Users },
      { to: '/meu-talento', nome: 'Meu perfil de talento', icon: Palette },
      { to: '/meus-orcamentos', nome: 'Pedidos de contratação', icon: MessageCircle },
      { to: '/conversas-equipe', nome: 'Conversas da equipe', icon: MessageCircle },
      { to: destinoOrganizador, nome: organizador ? 'Painel do organizador' : 'Organizar um evento', icon: LayoutDashboard },
      ...(usuario.papel_plataforma === 'admin_plataforma' ? [{ to: '/admin/repasses', nome: 'Administração', icon: ShieldCheck }] : []),
    ] : []),
    { to: '/ajuda', nome: 'Central de ajuda', icon: MessageCircle },
  ]

  return (
    <header className="sticky top-0 z-30 border-b border-border bg-card/95 backdrop-blur-md">
      <div className="mx-auto flex h-18 max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
        <Link to="/" aria-label={NOME_PLATAFORMA} className="flex shrink-0 items-center rounded-lg">
          <img src="/brand/logo.png" alt={NOME_PLATAFORMA} className="h-7 w-auto dark:hidden sm:h-8" />
          <img src="/brand/logo-dark.png" alt="" aria-hidden="true" className="hidden h-7 w-auto dark:block sm:h-8" />
        </Link>
        <nav aria-label="Navegação principal" className="hidden min-w-0 items-center gap-1 lg:flex">
          <NavLink to="/" end className="evve-nav-link">Explorar</NavLink>
          <NavLink to="/artistas" className="evve-nav-link">Artistas</NavLink>
          {usuario && <>
            <NavLink to="/meus-ingressos" className="evve-nav-link">Meus ingressos</NavLink>
            <NavLink to="/meus-eventos" className="evve-nav-link">Meus eventos</NavLink>
          </>}
        </nav>
        <div className="flex shrink-0 items-center gap-2">
          {!carregando && !carregandoOrg && <div className="hidden sm:block"><Link to={usuario ? destinoOrganizador : '/login'} className={buttonVariants({ variant: 'outline', size: 'sm' })}>
            <LayoutDashboard aria-hidden="true" />{organizador ? 'Painel do organizador' : 'Organizar evento'}
          </Link></div>}
          <ThemeToggle />
          {!carregando && usuario ? <div className="hidden lg:block">
            <MenuRoot>
              <MenuTrigger render={<Button variant="secondary" />} aria-label="Abrir menu da conta">
                <span className="flex size-7 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground" aria-hidden="true">{usuario.nome.slice(0, 1).toUpperCase()}</span>
                <span className="max-w-24 truncate">{usuario.nome.split(' ')[0]}</span><ChevronDown aria-hidden="true" />
              </MenuTrigger>
              <MenuContent aria-label="Minha conta">
                <MenuLabel>Minha conta</MenuLabel>
                <MenuLink to="/meus-artistas" icon={Users}>Meus artistas</MenuLink>
                <MenuLink to="/meu-talento" icon={Palette}>Meu perfil de talento</MenuLink>
                <MenuLink to="/meus-orcamentos" icon={MessageCircle}>Pedidos de contratação</MenuLink>
                <MenuLink to="/conversas-equipe" icon={MessageCircle}>Conversas da equipe</MenuLink>
                {usuario.papel_plataforma === 'admin_plataforma' && <MenuLink to="/admin/repasses" icon={ShieldCheck}>Administração</MenuLink>}
                <MenuSeparator /><MenuLink to="/ajuda" icon={MessageCircle}>Central de ajuda</MenuLink>
                <MenuAction onClick={() => logout()} className="text-destructive"><LogOut className="size-4" aria-hidden="true" />Sair da conta</MenuAction>
              </MenuContent>
            </MenuRoot>
          </div> : !carregando && <Link to="/login" className={buttonVariants({ size: 'sm' })}><UserRound aria-hidden="true" />Entrar</Link>}
          <Dialog.Root open={aberto} onOpenChange={setAberto}>
            <Dialog.Trigger render={<Button variant="outline" size="icon" className="lg:hidden" />} aria-label="Abrir menu de navegação"><MenuIcon aria-hidden="true" /></Dialog.Trigger>
            <Dialog.Portal>
              <Dialog.Backdrop className="fixed inset-0 z-50 bg-black/35 backdrop-blur-sm" />
              <Dialog.Popup className="evve-mobile-menu fixed inset-y-0 right-0 z-50 flex w-full max-w-sm flex-col border-l border-border bg-card p-5 shadow-2xl outline-none">
                <div className="mb-5 flex items-center justify-between gap-3">
                  <div><Dialog.Title className="text-lg font-semibold">Menu</Dialog.Title><Dialog.Description className="text-sm text-muted-foreground">{usuario ? `Olá, ${usuario.nome.split(' ')[0]}` : 'Encontre seu próximo evento'}</Dialog.Description></div>
                  <Dialog.Close render={<Button variant="ghost" size="icon" />} aria-label="Fechar menu"><X aria-hidden="true" /></Dialog.Close>
                </div>
                <nav aria-label="Navegação do celular" className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
                  {links.map(({ to, nome, icon: Icon }) => <NavLink key={to} to={to} end={to === '/'} onClick={() => setAberto(false)} className="evve-nav-link"><Icon className="size-4 shrink-0" aria-hidden="true" />{nome}</NavLink>)}
                </nav>
                {!carregando && <div className="mt-5 border-t border-border pt-4">
                  {usuario ? <Button variant="outline" className="w-full" onClick={() => { setAberto(false); void logout() }}><LogOut aria-hidden="true" />Sair da conta</Button> : <Link to="/login" onClick={() => setAberto(false)} className={buttonVariants({ className: 'w-full' })}><UserRound aria-hidden="true" />Entrar na minha conta</Link>}
                </div>}
              </Dialog.Popup>
            </Dialog.Portal>
          </Dialog.Root>
        </div>
      </div>
    </header>
  )
}
