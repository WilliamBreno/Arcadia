import { CardPayment, initMercadoPago, StatusScreen } from '@mercadopago/sdk-react'
import { useEffect, useRef, useState } from 'react'

import { ApiError } from '@/lib/api'
import { pagarCartao, verificarPagamento, type AlvoPagamento, type ResultadoCartao } from '@/lib/pagamento'

let chaveIniciada = ''

type Props = {
  publicKey: string
  pedidoId: AlvoPagamento
  totalCentavos: number
  // Item 7.6: até 12x, parcela mínima de R$ 20 (o servidor calcula).
  maxParcelas?: number
  email?: string
  aoConfirmar: () => void
  aoFalhar: () => void
}

// Formulário de cartão do Mercado Pago dentro da página (item 7.1). Os
// campos do cartão ficam num iframe do MP: o número nunca passa pelo Evve.
// Se o banco pedir confirmação (3DS), mostra a tela do banco aqui mesmo.
// Carregado sob demanda: o SDK só baixa quando a pessoa escolhe cartão.
export default function CartaoMercadoPago({ publicKey, pedidoId, totalCentavos, maxParcelas = 1, email, aoConfirmar, aoFalhar }: Props) {
  if (chaveIniciada !== publicKey) {
    initMercadoPago(publicKey, { locale: 'pt-BR' })
    chaveIniciada = publicKey
  }
  const [mensagem, setMensagem] = useState<string | null>(null)
  const [desafio, setDesafio] = useState<ResultadoCartao | null>(null)
  const [formKey, setFormKey] = useState(0)
  const temporizador = useRef<number | null>(null)

  // Durante a confirmação do banco, pergunta ao servidor a cada 3 s.
  useEffect(() => {
    if (!desafio?.pagamento_id) return
    let tentativas = 0
    const verificar = async () => {
      tentativas++
      try {
        const p = (await verificarPagamento(pedidoId, desafio.pagamento_id)) as { status: string }
        if (p.status === 'pago') {
          aoConfirmar()
          return
        }
      } catch {
        // segue tentando; a notificação do MP também confirma
      }
      if (tentativas < 100) temporizador.current = window.setTimeout(verificar, 3000)
    }
    temporizador.current = window.setTimeout(verificar, 3000)
    return () => {
      if (temporizador.current) window.clearTimeout(temporizador.current)
    }
  }, [desafio, pedidoId, aoConfirmar])

  if (desafio?.pagamento_id && desafio.desafio_url && desafio.desafio_creq) {
    return (
      <div className="flex flex-col gap-3">
        <p className="text-sm text-foreground">Seu banco pediu uma confirmação. Siga as instruções abaixo.</p>
        <StatusScreen
          locale="pt-BR"
          initialization={{
            paymentId: desafio.pagamento_id,
            additionalInfo: { externalResourceURL: desafio.desafio_url, creq: desafio.desafio_creq },
          }}
        />
        <button
          type="button"
          className="self-start text-sm text-muted-foreground underline underline-offset-4"
          onClick={() => {
            setDesafio(null)
            setFormKey((k) => k + 1)
          }}
        >
          Tentar com outro cartão
        </button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {mensagem && (
        <p role="alert" className="rounded-lg border border-destructive p-3 text-sm text-destructive">
          {mensagem}
        </p>
      )}
      <CardPayment
        key={formKey}
        locale="pt-BR"
        initialization={{ amount: totalCentavos / 100, payer: email ? { email } : undefined }}
        customization={{ paymentMethods: { maxInstallments: Math.max(1, maxParcelas) } }}
        onError={() => aoFalhar()}
        onSubmit={async (dados) => {
          setMensagem(null)
          try {
            const r = await pagarCartao(pedidoId, {
              token: dados.token,
              payment_method_id: dados.payment_method_id,
              installments: dados.installments || 1,
              issuer_id: dados.issuer_id ? String(dados.issuer_id) : '',
              documento_tipo: dados.payer?.identification?.type,
              documento_numero: dados.payer?.identification?.number,
            })
            if (r.situacao === 'aprovado') aoConfirmar()
            else if (r.situacao === 'desafio') setDesafio(r)
            else {
              setMensagem(r.mensagem ?? 'O pagamento não foi concluído.')
              // Um token de cartão vale uma vez: recria o formulário para tentar de novo.
              if (r.situacao === 'recusado') setFormKey((k) => k + 1)
            }
          } catch (e) {
            setMensagem(e instanceof ApiError ? e.message : 'Não foi possível processar o cartão. Tente de novo.')
            setFormKey((k) => k + 1)
          }
        }}
      />
    </div>
  )
}
