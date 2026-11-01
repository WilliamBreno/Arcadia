import { BarChart3, CalendarDays, ChevronDown, Code2, Contact, CreditCard, MapPin, Megaphone, Settings2, Sparkles, Trophy, Users } from 'lucide-react'
import { NavLink, useLocation } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { MenuContent, MenuLink, MenuRoot, MenuTrigger } from '@/components/ui/menu'

const grupos = [
  { nome: 'Pessoas', icon: Users, links: [
    { to: '/organizador/equipe', nome: 'Equipe', icon: Users },
    { to: '/organizador/coproducoes', nome: 'Coproduções', icon: Contact },
    { to: '/organizador/circuitos', nome: 'Circuitos', icon: Trophy },
  ] },
  { nome: 'Financeiro', icon: CreditCard, links: [
    { to: '/organizador/repasses', nome: 'Repasses', icon: CreditCard },
    { to: '/organizador/relatorios', nome: 'Relatórios', icon: BarChart3 },
    { to: '/organizador/pro', nome: 'Evve Pro', icon: Sparkles },
  ] },
  { nome: 'Ferramentas', icon: Settings2, links: [
    { to: '/organizador/crm', nome: 'Contatos e e-mail', icon: Megaphone },
    { to: '/organizador/locais', nome: 'Locais', icon: MapPin },
    { to: '/organizador/api', nome: 'Integrações e API', icon: Code2 },
    { to: '/organizador/perfil', nome: 'Perfil do organizador', icon: Settings2 },
  ] },
]

export function OrganizadorNav() {
  const { pathname } = useLocation()
  return <div className="border-b border-border bg-card">
    <nav aria-label="Área do organizador" className="mx-auto flex max-w-6xl flex-wrap items-center gap-1 px-4 py-2 sm:px-6">
      <NavLink to="/organizador/eventos" className="evve-nav-link"><CalendarDays className="size-4" aria-hidden="true" />Eventos</NavLink>
      {grupos.map(({ nome, icon: Icon, links }) => <MenuRoot key={nome}>
        <MenuTrigger render={<Button variant={links.some(l => pathname.startsWith(l.to)) ? 'secondary' : 'ghost'} size="sm" />}><Icon aria-hidden="true" />{nome}<ChevronDown aria-hidden="true" /></MenuTrigger>
        <MenuContent align="start" aria-label={nome}>{links.map(link => <MenuLink key={link.to} to={link.to} icon={link.icon}>{link.nome}</MenuLink>)}</MenuContent>
      </MenuRoot>)}
    </nav>
  </div>
}
