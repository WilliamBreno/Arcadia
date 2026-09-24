import { useQuery } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { useParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { Card, CardContent } from '@/components/ui/card'

type Ingresso = {
  codigo: string
  qr_token: string
  titular_nome: string
  tipo_ingresso_nome: string
  status: string
  evento_titulo: string
  evento_inicio_em: string | null
}

// Página aberta pelo link do e-mail de cortesia — não exige login; o
// segredo é o próprio qr_token na URL.
export default function IngressoPublico() {
  const { codigo, token } = useParams()
  const { data, error, isLoading } = useQuery({
    queryKey: ['ingresso-publico', codigo],
    queryFn: () => api<Ingresso>(`/ingressos/${codigo}/${token}`, { semAuth: true }),
    retry: false,
  })

  if (isLoading) return <p className="p-8 text-center text-muted-foreground">Carregando…</p>
  if (error instanceof ApiError || !data) {
    return <p className="p-8 text-center text-muted-foreground">Ingresso não encontrado.</p>
  }

  const ativo = data.status === 'pago'
  return (
    <main className="mx-auto max-w-sm px-4 py-12">
      <Card>
        <CardContent className="flex flex-col items-center gap-3 py-6 text-center">
          <p className="text-lg font-semibold text-foreground">{data.evento_titulo}</p>
          {data.evento_inicio_em && (
            <p className="text-sm text-muted-foreground">{new Date(data.evento_inicio_em).toLocaleString('pt-BR')}</p>
          )}
          {ativo ? (
            <QRCodeSVG value={`${data.codigo}:${data.qr_token}`} size={220} />
          ) : (
            <p className="text-sm text-destructive">Este ingresso não está mais válido ({data.status}).</p>
          )}
          <p className="font-mono text-sm text-foreground">{data.codigo}</p>
          <p className="text-sm text-muted-foreground">
            {data.titular_nome} · {data.tipo_ingresso_nome}
          </p>
        </CardContent>
      </Card>
    </main>
  )
}
