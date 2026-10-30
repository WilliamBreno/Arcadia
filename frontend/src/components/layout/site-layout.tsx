import { Outlet } from 'react-router-dom'

import { SiteFooter } from '@/components/layout/site-footer'
import { SiteHeader } from '@/components/layout/site-header'
import { BannerCookies } from '@/components/banner-cookies'
import { RotaDosPixels } from '@/components/rota-dos-pixels'
import { TopoAoNavegar } from '@/components/layout/topo-ao-navegar'

export function SiteLayout() {
  return (
    <div className="flex min-h-svh flex-col">
      <SiteHeader />
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
