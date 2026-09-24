import { useQuery } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { api } from '@/lib/api'
import type { MeuIngresso } from '@/lib/conta'
import { TransferirIngresso } from '@/components/transferir-ingresso'
import { Card, CardContent } from '@/components/ui/card'

const rotuloStatus: Record<string, string> = {
  pago: 'Válido',
  utilizado: 'Já utilizado',
  cancelado: 'Cancelado',
  reembolsado: 'Reembolsado',
}

export default function MeusIngressos() {
  const { data: ingressos, isLoading } = useQuery({
    queryKey: ['meus-ingressos'],
    queryFn: () => api<MeuIngresso[]>('/me/ingressos'),
  })
  const [selecionado, setSelecionado] = useState<MeuIngresso | null>(null)

  if (isLoading) return <p className="p-8 text-center text-muted-foreground">Carregando…</p>

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <h1 className="mb-6 text-2xl font-semibold text-foreground">Meus ingressos</h1>

      {ingressos?.length === 0 && <p className="text-muted-foreground">Você ainda não tem ingressos.</p>}

      <div className="flex flex-col gap-3">
        {ingressos?.map((i) => (
          <Card key={i.id} className="cursor-pointer" onClick={() => setSelecionado(i)}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <p className="font-medium text-foreground">{i.evento_titulo}</p>
                <p className="text-sm text-muted-foreground">
                  {i.evento_inicio_em && new Date(i.evento_inicio_em).toLocaleDateString('pt-BR')} · {i.titular_nome}
                </p>
                <p className="text-xs text-muted-foreground">{rotuloStatus[i.status] ?? i.status}</p>
              </div>
              <span className="font-mono text-xs text-muted-foreground">{i.codigo}</span>
            </CardContent>
          </Card>
        ))}
      </div>

      {selecionado && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => setSelecionado(null)}
        >
          <Card className="max-w-xs" onClick={(e) => e.stopPropagation()}>
            <CardContent className="flex flex-col items-center gap-4 py-6 text-center">
              <p className="font-medium text-foreground">{selecionado.evento_titulo}</p>
              <div className="rounded-lg bg-white p-4">
                <QRCodeSVG value={`${selecionado.codigo}:${selecionado.qr_token}`} size={200} />
              </div>
              <p className="font-mono text-sm text-foreground">{selecionado.codigo}</p>
              <p className="text-sm text-muted-foreground">{selecionado.titular_nome}</p>
              <p className="text-xs text-muted-foreground">{rotuloStatus[selecionado.status] ?? selecionado.status}</p>
              {selecionado.status === 'pago' && (
                <TransferirIngresso itemId={selecionado.id} onConcluido={() => setSelecionado(null)} />
              )}
              <Link
                to={`/e/${selecionado.evento_slug}`}
                className="text-xs text-primary underline-offset-4 hover:underline"
              >
                Ver evento
              </Link>
            </CardContent>
          </Card>
        </div>
      )}
    </main>
  )
}
