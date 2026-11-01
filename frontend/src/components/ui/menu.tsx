import type { ComponentProps, ReactNode } from 'react'
import { Menu } from '@base-ui/react/menu'
import type { LucideIcon } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import { cn } from 'cn'

export const MenuRoot = Menu.Root
export const MenuTrigger = Menu.Trigger

export function MenuContent({ children, align = 'end', ...props }: ComponentProps<typeof Menu.Popup> & { align?: 'start' | 'end' }) {
  return <Menu.Portal>
    <Menu.Positioner sideOffset={8} align={align} className="z-50 outline-none">
      <Menu.Popup {...props} className={cn('evve-menu-popup w-64 overflow-y-auto rounded-2xl border border-border bg-popover p-1.5 text-popover-foreground shadow-xl outline-none', props.className)}>
        {children}
      </Menu.Popup>
    </Menu.Positioner>
  </Menu.Portal>
}

export function MenuLink({ to, children, icon: Icon }: { to: string; children: ReactNode; icon?: LucideIcon }) {
  const { pathname } = useLocation()
  const ativo = pathname === to || (to !== '/organizador/eventos' && pathname.startsWith(to + '/'))
  return <Menu.LinkItem render={<Link to={to} />} closeOnClick aria-current={ativo ? 'page' : undefined} className={cn('evve-menu-item', ativo && 'bg-accent font-semibold text-accent-foreground')}>
    {Icon && <Icon className="size-4 shrink-0" aria-hidden="true" />}{children}
  </Menu.LinkItem>
}

export function MenuAction({ className, ...props }: ComponentProps<typeof Menu.Item>) {
  return <Menu.Item {...props} className={cn('evve-menu-item', className)} />
}

export function MenuLabel({ children }: { children: ReactNode }) {
  return <div className="px-3 pb-1.5 pt-3 text-xs font-semibold tracking-wide text-muted-foreground">{children}</div>
}

export function MenuSeparator() { return <div role="separator" className="my-1.5 h-px bg-border" /> }
