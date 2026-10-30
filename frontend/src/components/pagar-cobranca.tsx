import { useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2 } from 'lucide-react'
import { useEffect, useRef } from 'react'

import { PagamentoNaPagina } from '@/components/pagamento-na-pagina'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import type { Cobranca } from '@/lib/plataforma'

// Item 10.4: paga uma cobrança da Evve (Pro ou destaque) na própria página.
// Quem diz se pagou é o servidor: a cobrança é consultada a cada 3 s.
export function PagarCobranca({ cobranca, aoPagar, aoFechar }: { cobranca: Cobranca; aoPagar: () => void; aoFechar: () => void }) {
  const queryClient = useQueryClient()
  const atual = useQuery({
    queryKey: ['org-cobranca', cobranca.id],
    initialData: cobranca,
    queryFn: () => api<Cobranca>(`/org/cobrancas/${cobranca.id}`),
    refetchInterval: (q) => (q.state.data?.status === 'pendente' ? 3000 : false),
  })
  const pago = atual.data?.status === 'paga'
  const avisado = useRef(false)
  useEffect(() => {
    if (pago && !avisado.current) {
      avisado.current = true
      queryClient.invalidateQueries({ queryKey: ['org-pro'] })
      queryClient.invalidateQueries({ queryKey: ['org-destaques'] })
      aoPagar()
    }
  }, [pago, aoPagar, queryClient])

  if (pago) {
    return (
      <div role="status" className="flex flex-col gap-3 rounded-xl border border-green-600 p-4">
        <p className="flex items-center gap-2 font-medium text-foreground">
          <CheckCircle2 className="size-5 text-green-600" aria-hidden="true" />
          Pagamento confirmado: {cobranca.descricao}
        </p>
        <Button size="sm" variant="outline" className="self-start" onClick={aoFechar}>
          Fechar
        </Button>
      </div>
    )
  }
  if (atual.data?.status === 'expirada') {
    return (
      <div className="flex flex-col gap-3 rounded-xl border border-border p-4">
        <p className="text-sm text-foreground">Esta cobrança expirou. Faça uma nova.</p>
        <Button size="sm" variant="outline" className="self-start" onClick={aoFechar}>
          Voltar
        </Button>
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-border p-4">
      <div className="flex items-baseline justify-between gap-3">
        <p className="text-sm text-foreground">{cobranca.descricao}</p>
        <p className="font-semibold tabular-nums text-foreground">{formatarCentavos(cobranca.valor_centavos)}</p>
      </div>
      <PagamentoNaPagina pedidoId={`/org/cobrancas/${cobranca.id}`} totalCentavos={cobranca.valor_centavos} aoConfirmar={() => atual.refetch()} />
      <Button size="sm" variant="ghost" className="self-start" onClick={aoFechar}>
        Cancelar
      </Button>
    </div>
  )
}
