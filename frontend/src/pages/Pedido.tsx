import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import type { Pedido as PedidoTipo } from '@/lib/checkout'
import { formatarCentavos } from '@/lib/evento'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

const rotuloStatus: Record<PedidoTipo['status'], string> = {
  aberto: 'Aberto',
  aguardando_pagamento: 'Aguardando pagamento',
  pago: 'Pago',
  expirado: 'Expirado',
  cancelado: 'Cancelado',
  reembolsado_parcial: 'Reembolsado parcialmente',
  reembolsado: 'Reembolsado',
}

const bannerRetorno: Record<string, { texto: string; classe: string }> = {
  sucesso: { texto: 'Pagamento em processamento — a confirmação chega em instantes.', classe: 'border-primary text-primary' },
  falha: { texto: 'O pagamento não foi concluído. Você pode tentar novamente.', classe: 'border-destructive text-destructive' },
  pendente: { texto: 'Pagamento pendente (ex.: Pix aguardando compensação).', classe: 'border-border text-muted-foreground' },
}

export default function Pedido() {
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const statusRetorno = searchParams.get('status')
  const [erro, setErro] = useState<string | null>(null)
  const [pagando, setPagando] = useState(false)

  const { data: pedido, isLoading, refetch } = useQuery({
    queryKey: ['pedido', id],
    queryFn: () => api<PedidoTipo>(`/pedidos/${id}`),
    refetchInterval: (query) => (query.state.data?.status === 'aguardando_pagamento' ? 5000 : false),
  })

  const pagar = async () => {
    setErro(null)
    setPagando(true)
    try {
      const resp = await api<{ checkout_url: string }>(`/pedidos/${id}/pagar`, { method: 'POST' })
      window.location.href = resp.checkout_url
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao iniciar pagamento')
      setPagando(false)
    }
  }

  if (isLoading) return <p className="p-8 text-center text-muted-foreground">Carregando…</p>
  if (!pedido) return null

  const banner = statusRetorno ? bannerRetorno[statusRetorno] : null

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      {banner && (
        <div className={`mb-6 rounded-lg border p-3 text-sm ${banner.classe}`}>
          {banner.texto}{' '}
          <button type="button" onClick={() => refetch()} className="underline underline-offset-4">
            Atualizar status
          </button>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>
            Pedido #{pedido.id} <span className="text-sm font-normal text-muted-foreground">({rotuloStatus[pedido.status]})</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            {pedido.itens.map((item) => (
              <div key={item.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
                <div>
                  <p className="text-foreground">{item.titular_nome}</p>
                  <p className="text-xs text-muted-foreground">
                    {formatarCentavos(item.preco_centavos)} + {formatarCentavos(item.taxa_plataforma_centavos)} de taxa
                  </p>
                </div>
                <span className="font-medium text-foreground">{formatarCentavos(item.total_centavos)}</span>
              </div>
            ))}
          </div>

          <div className="flex items-center justify-between border-t border-border pt-4">
            <span className="font-medium text-foreground">Total</span>
            <span className="text-lg font-semibold text-foreground">{formatarCentavos(pedido.total_centavos)}</span>
          </div>

          {pedido.status === 'aguardando_pagamento' && (
            <>
              {pedido.expira_em && (
                <p className="text-xs text-muted-foreground">
                  Reserva válida até {new Date(pedido.expira_em).toLocaleTimeString('pt-BR')}.
                </p>
              )}
              {erro && <p className="text-sm text-destructive">{erro}</p>}
              <Button onClick={pagar} disabled={pagando}>
                {pagando ? 'Redirecionando…' : 'Pagar com Mercado Pago'}
              </Button>
            </>
          )}

          {pedido.status === 'pago' && (
            <p className="text-sm text-muted-foreground">
              Pagamento confirmado! Os códigos dos ingressos aparecem em "Meus ingressos".
            </p>
          )}
          {pedido.status === 'expirado' && (
            <p className="text-sm text-muted-foreground">Essa reserva expirou. Volte ao evento para comprar de novo.</p>
          )}
        </CardContent>
      </Card>
    </main>
  )
}
