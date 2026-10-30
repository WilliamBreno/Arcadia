import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { Button } from '@/components/ui/button'

export type MinhaRevenda = { item_id: number; status: 'a_venda' | 'reservada' | 'vendida'; valor_devolver_centavos: number }

type Props = { itemId: number; precoCentavos?: number; revenda?: MinhaRevenda }

// Revenda oficial (item 7.4): só abre com o ingresso esgotado, pelo valor de
// face. Quem revende recebe de volta o preço do ingresso no mesmo Pix ou
// cartão da compra; a taxa de serviço não volta.
export function RevenderIngresso({ itemId, precoCentavos, revenda }: Props) {
  const queryClient = useQueryClient()
  const [confirmando, setConfirmando] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  const atualizar = () => {
    queryClient.invalidateQueries({ queryKey: ['minhas-revendas'] })
    queryClient.invalidateQueries({ queryKey: ['meus-ingressos'] })
  }

  const agir = async (metodo: 'POST' | 'DELETE') => {
    setErro(null)
    setEnviando(true)
    try {
      await api(`/itens/${itemId}/revenda`, { method: metodo })
      setConfirmando(false)
      atualizar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível agora. Tente de novo.')
    } finally {
      setEnviando(false)
    }
  }

  if (revenda?.status === 'reservada') {
    return (
      <p className="rounded-lg bg-muted px-3 py-2 text-xs text-foreground" role="status">
        Alguém está comprando este ingresso na revenda agora. Se o pagamento for aprovado, você recebe{' '}
        {formatarCentavos(revenda.valor_devolver_centavos)} de volta.
      </p>
    )
  }

  if (revenda?.status === 'a_venda') {
    return (
      <div className="flex w-full flex-col gap-2">
        <p className="rounded-lg bg-muted px-3 py-2 text-xs text-foreground" role="status">
          À venda na revenda oficial. Se vender, você recebe {formatarCentavos(revenda.valor_devolver_centavos)}. Até lá, o ingresso
          continua seu.
        </p>
        <Button variant="outline" size="sm" disabled={enviando} onClick={() => agir('DELETE')}>
          {enviando ? 'Retirando…' : 'Retirar da revenda'}
        </Button>
        {erro && <p className="text-xs text-destructive">{erro}</p>}
      </div>
    )
  }

  if (!confirmando) {
    return (
      <div className="flex w-full flex-col gap-1">
        <Button variant="outline" size="sm" onClick={() => setConfirmando(true)}>
          Revender (revenda oficial)
        </Button>
        {erro && <p className="text-xs text-destructive">{erro}</p>}
      </div>
    )
  }

  return (
    <div className="flex w-full flex-col gap-2 rounded-lg border border-border p-3 text-left text-xs text-foreground">
      <p className="font-medium">Como funciona a revenda</p>
      <ul className="list-disc pl-4 text-muted-foreground">
        <li>O ingresso vai à venda pelo preço de tabela. Se houver lista de espera, a primeira pessoa tem a vez.</li>
        <li>
          Se alguém comprar, você recebe de volta{precoCentavos ? ` ${formatarCentavos(precoCentavos)}` : ' o preço do ingresso'} no mesmo Pix
          ou cartão da compra. A taxa de serviço não volta.
        </li>
        <li>Até vender, o ingresso continua seu, e você pode retirar da revenda. Se não vender até o início do evento, ele continua válido.</li>
      </ul>
      <div className="flex gap-2">
        <Button size="sm" disabled={enviando} onClick={() => agir('POST')}>
          {enviando ? 'Anunciando…' : 'Pôr à venda'}
        </Button>
        <Button size="sm" variant="ghost" onClick={() => setConfirmando(false)}>
          Agora não
        </Button>
      </div>
      {erro && <p className="text-destructive">{erro}</p>}
    </div>
  )
}
