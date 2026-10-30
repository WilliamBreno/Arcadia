import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { api } from '@/lib/api'
import type { AvisoStatus } from '@/lib/ajuda'

// Item 9.9: faixa no topo do site enquanto houver aviso de incidente aberto.
export function FaixaAvisos() {
  const { data } = useQuery({
    queryKey: ['avisos-status'],
    queryFn: () => api<AvisoStatus[]>('/status/avisos', { semAuth: true }),
    refetchInterval: 120_000,
    retry: false,
  })
  const aviso = data?.[0]
  if (!aviso) return null
  const cor = aviso.nivel === 'falha' ? 'bg-red-700 text-white' : 'bg-yellow-400 text-black'
  return (
    <div role="status" className={`px-4 py-2 text-center text-sm ${cor}`}>
      <strong>{aviso.titulo}</strong>
      {aviso.mensagem ? ` — ${aviso.mensagem}` : ''}{' '}
      <Link to="/status" className="underline underline-offset-4">
        Ver status
      </Link>
    </div>
  )
}
