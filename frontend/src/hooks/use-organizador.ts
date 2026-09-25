import { useQuery } from '@tanstack/react-query'

import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import type { Organizador } from '@/lib/organizador'

// Perfil de organizador do usuário logado; `null` quando ele ainda não
// criou um (o backend responde 404 nesse caso).
export function buscarPerfilOrganizador() {
  return api<Organizador>('/org/perfil').catch((e) => {
    if (e instanceof ApiError && e.status === 404) return null
    throw e
  })
}

export function useOrganizador() {
  const { usuario } = useAuth()
  const { data, isLoading } = useQuery({
    queryKey: ['org-perfil', usuario?.id],
    queryFn: buscarPerfilOrganizador,
    enabled: !!usuario,
    staleTime: 60_000,
  })
  return { organizador: data ?? null, carregando: !!usuario && isLoading }
}
