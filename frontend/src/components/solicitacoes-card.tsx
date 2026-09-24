import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

type Solicitacao = {
  id: number
  usuario_nome: string
  usuario_email: string
  status: 'pendente' | 'aprovada' | 'rejeitada' | 'lista_espera'
}

const rotulo = { pendente: 'Pendente', aprovada: 'Aprovada', rejeitada: 'Rejeitada', lista_espera: 'Lista de espera' }

export function SolicitacoesCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [erro, setErro] = useState<string | null>(null)

  const { data } = useQuery({
    queryKey: ['org-evento-solicitacoes', eventoId],
    queryFn: () => api<Solicitacao[]>(`/org/eventos/${eventoId}/solicitacoes`),
  })

  const decidir = async (id: number, decisao: 'aprovar' | 'rejeitar' | 'lista_espera') => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/solicitacoes/${id}`, { method: 'POST', body: { decisao } })
      queryClient.invalidateQueries({ queryKey: ['org-evento-solicitacoes', eventoId] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao decidir')
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Solicitações da plateia</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <p className="text-xs text-muted-foreground">
          Aprovar sem vaga livre coloca a pessoa na lista de espera; quando uma vaga abre (reserva expirada, cancelamento, rejeição), o
          próximo da fila é aprovado e avisado por e-mail.
        </p>
        {data?.length === 0 && <p className="text-sm text-muted-foreground">Nenhuma solicitação ainda.</p>}
        {data?.map((s) => (
          <div key={s.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <div>
              <p className="text-sm font-medium text-foreground">{s.usuario_nome}</p>
              <p className="text-xs text-muted-foreground">
                {s.usuario_email} · {rotulo[s.status]}
              </p>
            </div>
            <div className="flex gap-2">
              {s.status !== 'aprovada' && (
                <Button size="sm" onClick={() => decidir(s.id, 'aprovar')}>
                  Aprovar
                </Button>
              )}
              {s.status === 'pendente' && (
                <Button variant="outline" size="sm" onClick={() => decidir(s.id, 'lista_espera')}>
                  Lista de espera
                </Button>
              )}
              {s.status !== 'rejeitada' && (
                <Button variant="destructive" size="sm" onClick={() => decidir(s.id, 'rejeitar')}>
                  Rejeitar
                </Button>
              )}
            </div>
          </div>
        ))}
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}
