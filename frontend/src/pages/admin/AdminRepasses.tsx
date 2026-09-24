import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { listarRepassesAdmin, pagarRepasse } from '@/lib/financeiro'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export default function AdminRepasses() {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState('pendente')
  const [erro, setErro] = useState<string | null>(null)

  const { data, error } = useQuery({
    queryKey: ['admin-repasses', status],
    queryFn: () => listarRepassesAdmin(status),
  })

  const pagar = async (id: number) => {
    const comprovante = window.prompt('Link do comprovante do Pix (opcional):') ?? ''
    if (!confirm('Confirmar que o Pix foi enviado ao organizador?')) return
    setErro(null)
    try {
      await pagarRepasse(id, comprovante, '')
      queryClient.invalidateQueries({ queryKey: ['admin-repasses'] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao marcar como pago')
    }
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-destructive">Acesso restrito ao admin da plataforma.</p>
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Repasses</h1>
        <select
          className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="pendente">A pagar</option>
          <option value="pago">Pagos</option>
          <option value="">Todos</option>
        </select>
      </div>
      {erro && <p className="mb-3 text-sm text-destructive">{erro}</p>}
      {data?.length === 0 && <p className="text-muted-foreground">Nada por aqui.</p>}
      <div className="flex flex-col gap-3">
        {data?.map((r) => (
          <Card key={r.id}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <p className="font-medium text-foreground">{r.evento_titulo}</p>
                <p className="text-xs text-muted-foreground">
                  {r.organizador_nome} · Pix ({r.tipo_chave_pix}): <span className="font-mono">{r.chave_pix || 'não cadastrada'}</span>
                </p>
                <p className="text-xs text-muted-foreground">
                  Bruto {formatarCentavos(r.valor_bruto_centavos)} − taxa {formatarCentavos(r.taxa_processador_centavos)} · {r.status}
                </p>
              </div>
              <div className="flex items-center gap-3">
                <span className="font-semibold text-foreground">{formatarCentavos(r.valor_liquido_centavos)}</span>
                {r.status === 'pendente' && (
                  <Button size="sm" onClick={() => pagar(r.id)}>
                    Marcar pago
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
