// Seguir organizador e categoria (item 6.4).
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'

export type TipoSeguir = 'organizador' | 'categoria'
export type Seguindo = { tipo: TipoSeguir; alvo: string; rotulo: string; criado_em: string }

export function useSeguindo() {
  const { usuario } = useAuth()
  const queryClient = useQueryClient()
  const lista = useQuery({
    queryKey: ['me-seguindo', usuario?.id],
    queryFn: () => api<Seguindo[]>('/me/seguindo'),
    enabled: !!usuario,
    staleTime: 60_000,
  })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['me-seguindo'] })
  const seguir = useMutation({
    mutationFn: (v: { tipo: TipoSeguir; alvo: string; rotulo?: string }) => api('/seguindo', { method: 'POST', body: v }),
    onSuccess: invalidar,
  })
  const deixar = useMutation({
    mutationFn: (v: { tipo: TipoSeguir; alvo: string }) =>
      api(`/seguindo?tipo=${encodeURIComponent(v.tipo)}&alvo=${encodeURIComponent(v.alvo)}`, { method: 'DELETE' }),
    onSuccess: invalidar,
  })
  const segue = (tipo: TipoSeguir, alvo: string) => !!lista.data?.some((s) => s.tipo === tipo && s.alvo === alvo)
  return { lista: lista.data ?? [], segue, seguir, deixar, logado: !!usuario }
}

// Link "adicionar ao Google Agenda" (o servidor também gera o .ics).
export function linkGoogleAgenda(titulo: string, inicio: string | null, fim: string | null, onde: string, url: string) {
  if (!inicio) return null
  const ini = new Date(inicio)
  const f = fim && new Date(fim) > ini ? new Date(fim) : new Date(ini.getTime() + 3 * 3600_000)
  const fmt = (d: Date) => d.toISOString().replace(/[-:]/g, '').replace(/\.\d{3}/, '')
  const p = new URLSearchParams({ action: 'TEMPLATE', text: titulo, dates: `${fmt(ini)}/${fmt(f)}`, location: onde, details: `Ingressos e detalhes: ${url}` })
  return `https://calendar.google.com/calendar/render?${p.toString()}`
}
