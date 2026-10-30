import { useEffect } from 'react'
import { useLocation, useNavigationType } from 'react-router-dom'

// Ao abrir uma página nova, começa do topo. No "voltar" (POP) o navegador
// devolve a rolagem de antes; links com âncora (#) rolam até a âncora.
export function TopoAoNavegar() {
  const { pathname, hash } = useLocation()
  const tipo = useNavigationType()
  useEffect(() => {
    if (tipo !== 'POP' && !hash) window.scrollTo(0, 0)
  }, [pathname, hash, tipo])
  return null
}
