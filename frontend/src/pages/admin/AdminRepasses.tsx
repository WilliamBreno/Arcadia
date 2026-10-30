import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { enviarPixRepasse, listarRepassesAdmin, pagarRepasse, pagarReserva, usarReserva } from '@/lib/financeiro'
import { AdminNav } from '@/components/admin-nav'
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

  // Item 9.1: envia o Pix pelo Mercado Pago (mesmo acima do limite automático).
  const enviarPix = async (id: number, valor: string) => {
    if (!confirm(`Enviar ${valor} por Pix agora, pelo Mercado Pago?`)) return
    setErro(null)
    try {
      await enviarPixRepasse(id)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao enviar o Pix')
    }
    queryClient.invalidateQueries({ queryKey: ['admin-repasses'] })
  }

  const acaoReserva = async (id: number, acao: 'pagar' | 'usar') => {
    setErro(null)
    try {
      if (acao === 'pagar') {
        const comprovante = window.prompt('Link do comprovante do Pix da reserva (opcional):') ?? ''
        if (!confirm('Confirmar que o Pix da reserva foi enviado ao organizador?')) return
        await pagarReserva(id, comprovante)
      } else {
        const motivo = window.prompt('Qual contestação esta reserva vai cobrir? (obrigatório)')
        if (!motivo) return
        await usarReserva(id, motivo)
      }
      queryClient.invalidateQueries({ queryKey: ['admin-repasses'] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao atualizar a reserva')
    }
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-destructive">Acesso restrito ao admin da plataforma.</p>
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <AdminNav />
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Repasses</h1>
        <select
          className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="pendente">A pagar</option>
          <option value="pago">Pagos</option>
          <option value="reservas">Reservas retidas</option>
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
                  Bruto {formatarCentavos(r.valor_bruto_centavos)} − taxa {formatarCentavos(r.taxa_processador_centavos)}
                  {r.taxa_absorvida_centavos > 0 && <> − taxa absorvida {formatarCentavos(r.taxa_absorvida_centavos)}</>} · {r.status}
                </p>
                {r.envio_status !== 'manual' && (
                  <p className={`text-xs ${r.envio_status === 'falhou' || r.envio_status === 'aguardando_aprovacao' ? 'font-medium text-foreground' : 'text-muted-foreground'}`}>
                    Pix automático: {r.envio_rotulo}
                    {r.envio_motivo && ` · ${r.envio_motivo}`}
                  </p>
                )}
                {r.reserva_centavos > 0 && (
                  <p className="text-xs text-muted-foreground">
                    Reserva {formatarCentavos(r.reserva_centavos)} ({r.reserva_status})
                    {r.reserva_status === 'retida' && r.reserva_liberar_em && ` · libera em ${new Date(r.reserva_liberar_em).toLocaleDateString('pt-BR')}`}
                  </p>
                )}
              </div>
              <div className="flex items-center gap-3">
                <span className="font-semibold text-foreground">
                  {formatarCentavos(r.status === 'pendente' ? r.valor_a_pagar_agora_centavos : r.valor_liquido_centavos)}
                </span>
                {r.status === 'pendente' && !['enviando', 'processando', 'enviado'].includes(r.envio_status) && (
                  <>
                    {r.tipo_chave_pix !== 'aleatoria' && (
                      <Button size="sm" onClick={() => enviarPix(r.id, formatarCentavos(r.valor_a_pagar_agora_centavos))}>
                        {r.envio_status === 'falhou' ? 'Tentar Pix de novo' : 'Enviar Pix'}
                      </Button>
                    )}
                    <Button size="sm" variant="outline" onClick={() => pagar(r.id)}>
                      Marcar pago (Pix manual)
                    </Button>
                  </>
                )}
                {r.status === 'pago' && r.reserva_status === 'retida' && (
                  <>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => acaoReserva(r.id, 'pagar')}
                      disabled={!!r.reserva_liberar_em && new Date(r.reserva_liberar_em) > new Date()}
                    >
                      Pagar reserva
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => acaoReserva(r.id, 'usar')}>
                      Usar reserva
                    </Button>
                  </>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
