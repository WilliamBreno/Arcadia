import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import type { ItemPedido, Pedido as PedidoTipo } from '@/lib/checkout'
import { formatarCentavos } from '@/lib/evento'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

type DecisaoCancelamento = { pode: boolean; motivo: string; valor_reembolso_centavos: number }

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
              <ItemLinha key={item.id} item={item} pedidoId={pedido.id} />
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

const rotuloStatusItem: Record<ItemPedido['status'], string> = {
  reservado: 'Reservado',
  pago: 'Pago',
  utilizado: 'Utilizado',
  cancelado: 'Cancelado',
  reembolsado: 'Reembolsado',
  expirado: 'Expirado',
}

function ItemLinha({ item, pedidoId }: { item: ItemPedido; pedidoId: number }) {
  const queryClient = useQueryClient()
  const [verificando, setVerificando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  const cancelar = async () => {
    setErro(null)
    setVerificando(true)
    try {
      const decisao = await api<DecisaoCancelamento>(`/itens/${item.id}/cancelamento`)
      if (!decisao.pode) {
        alert(decisao.motivo)
        return
      }
      const confirmado = confirm(
        `${decisao.motivo}\n\nVocê recebe de volta ${formatarCentavos(decisao.valor_reembolso_centavos)}. Confirmar cancelamento?`,
      )
      if (!confirmado) return

      await api(`/itens/${item.id}/cancelar`, { method: 'POST' })
      queryClient.invalidateQueries({ queryKey: ['pedido', String(pedidoId)] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao verificar cancelamento')
    } finally {
      setVerificando(false)
    }
  }

  return (
    <div className="flex flex-col gap-1 rounded-lg border border-border px-3 py-2 text-sm">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-foreground">
            {item.titular_nome} <span className="text-xs text-muted-foreground">({rotuloStatusItem[item.status]})</span>
          </p>
          <p className="text-xs text-muted-foreground">
            {formatarCentavos(item.preco_centavos)} + {formatarCentavos(item.taxa_plataforma_centavos)} de taxa
            {item.garantia_contratada && <> + {formatarCentavos(item.garantia_centavos)} de garantia de vaga</>}
            {item.desconto_centavos > 0 && <> (cupom: −{formatarCentavos(item.desconto_centavos)})</>}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span className="font-medium text-foreground">{formatarCentavos(item.total_centavos)}</span>
          {item.status === 'pago' && (
            <Button variant="destructive" size="sm" onClick={cancelar} disabled={verificando}>
              Cancelar
            </Button>
          )}
        </div>
      </div>
      {erro && <p className="text-xs text-destructive">{erro}</p>}
    </div>
  )
}
