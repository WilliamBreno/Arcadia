import { Outlet, useLocation } from 'react-router-dom'
import { OrganizadorNav } from '@/components/organizador-nav'

import { SiteFooter } from '@/components/layout/site-footer'
import { SiteHeader } from '@/components/layout/site-header'
import { FaixaAvisos } from '@/components/faixa-avisos'
import { BannerCookies } from '@/components/banner-cookies'
import { RotaDosPixels } from '@/components/rota-dos-pixels'
import { TopoAoNavegar } from '@/components/layout/topo-ao-navegar'

export function SiteLayout() {
  const { pathname } = useLocation()
  return (
    <div className="flex min-h-svh flex-col">
      <FaixaAvisos />
      <SiteHeader />
      {pathname.startsWith('/organizador/') && <OrganizadorNav />}
      <div className="flex-1">
        <Outlet />
      </div>
      <SiteFooter />
      <BannerCookies />
      <RotaDosPixels />
      <TopoAoNavegar />
    </div>
  )
}
