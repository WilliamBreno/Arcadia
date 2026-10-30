import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'

import { pixelsCarregados, podeFicar } from '@/lib/pixels'

// Item 7.5: ao sair das páginas do evento com pixels carregados, recarrega a
// página de destino. Assim os scripts de anúncio deixam de existir e o
// organizador não mede nada fora do evento dele.
export function RotaDosPixels() {
  const { pathname, search, hash } = useLocation()
  useEffect(() => {
    const p = pixelsCarregados()
    if (p && !podeFicar(pathname, p.slug)) window.location.replace(pathname + search + hash)
  }, [pathname, search, hash])
  return null
}
