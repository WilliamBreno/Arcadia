import { useQuery } from '@tanstack/react-query'

import { VoltarLink } from '@/components/voltar-link'
import { formatarCentavos } from '@/lib/evento'
import { obterExtrato } from '@/lib/financeiro'
import { Card, CardContent } from '@/components/ui/card'

const rotulo = { calculado: 'Calculado', pendente: 'A receber', pago: 'Pago', cancelado: 'Cancelado' }

export default function Repasses() {
  const { data } = useQuery({ queryKey: ['org-repasses'], queryFn: obterExtrato })

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar ao painel</VoltarLink>
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Repasses</h1>
      <p className="mb-6 text-sm text-muted-foreground">
        A receber: <strong className="text-foreground">{formatarCentavos(data?.a_receber_centavos ?? 0)}</strong>
        {(data?.reserva_retida_centavos ?? 0) > 0 && (
          <> · Reserva retida: <strong className="text-foreground">{formatarCentavos(data?.reserva_retida_centavos ?? 0)}</strong></>
        )}
      </p>
      {data?.repasses.length === 0 && <p className="text-muted-foreground">Nenhum repasse gerado ainda.</p>}
      <div className="flex flex-col gap-3">
        {data?.repasses.map((r) => (
          <Card key={r.id}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <p className="font-medium text-foreground">{r.evento_titulo}</p>
                <p className="text-xs text-muted-foreground">
                  Bruto {formatarCentavos(r.valor_bruto_centavos)} − taxa do processador {formatarCentavos(r.taxa_processador_centavos)}
                  {r.taxa_absorvida_centavos > 0 && <> − taxa da plataforma paga por você {formatarCentavos(r.taxa_absorvida_centavos)}</>}
                </p>
                {r.reserva_centavos > 0 && (
                  <p className="text-xs text-muted-foreground">
                    Reserva contra contestações: {formatarCentavos(r.reserva_centavos)}
                    {r.reserva_status === 'retida' &&
                      r.reserva_liberar_em &&
                      ` · liberada a partir de ${new Date(r.reserva_liberar_em).toLocaleDateString('pt-BR')}`}
                    {r.reserva_status === 'paga' && ' · paga'}
                    {r.reserva_status === 'usada' && ` · usada para cobrir contestação${r.reserva_observacao ? ` (${r.reserva_observacao})` : ''}`}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">
                  {r.pago_em ? `Pago em ${new Date(r.pago_em).toLocaleDateString('pt-BR')}` : `Liberado em ${new Date(r.liberar_em).toLocaleDateString('pt-BR')}`}
                </p>
              </div>
              <div className="text-right">
                <p className="font-semibold text-foreground">{formatarCentavos(r.valor_a_pagar_agora_centavos)}</p>
                <span className="text-xs text-muted-foreground">{rotulo[r.status]}</span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
