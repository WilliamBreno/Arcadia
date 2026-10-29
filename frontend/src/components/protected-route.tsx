import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { useAuth } from '@/hooks/use-auth'
import { Carregando } from '@/components/carregando'

export function ProtectedRoute() {
  const { usuario, carregando } = useAuth()
  const location = useLocation()

  if (carregando) {
    return <Carregando />
  }

  if (!usuario) {
    // Guarda também os parâmetros (ex.: ?titulo= vindo de "Quanto custa").
    return <Navigate to="/login" state={{ de: location.pathname + location.search }} replace />
  }

  return <Outlet />
}
