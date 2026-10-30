import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { api, ApiError } from '@/lib/api'

type AdicionalPortaria = { id: number; nome: string; titular: string; codigo: string; status: 'pago' | 'utilizado'; entregue_em: string | null }

// Item 10.3: adicionais na busca manual da portaria (entregar sem o QR).
export function BuscaAdicionais({ eventoId, busca }: { eventoId: number; busca: string }) {
  const queryClient = useQueryClient()
  const termo = busca.trim()
  const { data } = useQuery({
    queryKey: ['adicionais-portaria', eventoId, termo],
    enabled: termo.length >= 2,
    queryFn: () => api<AdicionalPortaria[]>(`/checkin/eventos/${eventoId}/adicionais?q=${encodeURIComponent(termo)}`),
  })
  const [erro, setErro] = useState<string | null>(null)
  if (!data || data.length === 0) return null

  const entregar = async (codigo: string) => {
    setErro(null)
    try {
      await api(`/checkin/eventos/${eventoId}/adicionais/entregar`, { method: 'POST', body: { codigo } })
      queryClient.invalidateQueries({ queryKey: ['adicionais-portaria', eventoId] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível entregar.')
    }
  }

  return (
    <div className="mt-3 flex flex-col gap-2">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Adicionais</p>
      {data.map((a) => (
        <div key={a.id} className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
          <div className="min-w-0">
            <p className="text-sm font-medium text-foreground">
              {a.nome} · {a.titular}
            </p>
            <p className="text-xs text-muted-foreground">{a.codigo}</p>
          </div>
          {a.status === 'utilizado' ? (
            <span className="rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs text-yellow-800">entregue</span>
          ) : (
            <Button size="sm" onClick={() => entregar(a.codigo)}>
              Entregar
            </Button>
          )}
        </div>
      ))}
      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
    </div>
  )
}
