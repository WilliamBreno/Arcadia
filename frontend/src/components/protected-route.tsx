import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { useAuth } from '@/hooks/use-auth'

export function ProtectedRoute() {
  const { usuario, carregando } = useAuth()
  const location = useLocation()

  if (carregando) {
    return <div className="p-8 text-center text-muted-foreground">Carregando…</div>
  }

  if (!usuario) {
    return <Navigate to="/login" state={{ de: location.pathname }} replace />
  }

  return <Outlet />
}
