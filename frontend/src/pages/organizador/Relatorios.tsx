import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

type Relatorio = {
  totais: { vendidos: number; receita_centavos: number; checkins: number }
  por_evento: {
    evento_id: number
    titulo: string
    status: string
    vendidos: number
    cortesias: number
    receita_centavos: number
    checkins: number
    cancelados: number
    comparecimento: number
  }[]
  vendas_por_dia: { dia: string; quantidade: number; receita_centavos: number }[]
}

export default function Relatorios() {
  const [de, setDe] = useState('')
  const [ate, setAte] = useState('')
  const query = new URLSearchParams()
  if (de) query.set('de', de)
  if (ate) query.set('ate', ate)

  const { data, error } = useQuery({
    queryKey: ['org-relatorios', de, ate],
    queryFn: () => api<Relatorio>(`/org/relatorios?${query.toString()}`),
  })

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-muted-foreground">Só o dono do perfil de organizador vê os relatórios.</p>
  }

  const maxDia = Math.max(1, ...(data?.vendas_por_dia.map((d) => d.quantidade) ?? [1]))

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <h1 className="mb-6 text-2xl font-semibold text-foreground">Relatórios</h1>

      <div className="mb-6 grid grid-cols-3 gap-4">
        {[
          ['Ingressos vendidos', String(data?.totais.vendidos ?? 0)],
          ['Receita bruta', formatarCentavos(data?.totais.receita_centavos ?? 0)],
          ['Check-ins', String(data?.totais.checkins ?? 0)],
        ].map(([rotulo, valor]) => (
          <Card key={rotulo}>
            <CardContent className="py-4">
              <p className="text-xs text-muted-foreground">{rotulo}</p>
              <p className="text-xl font-semibold text-foreground">{valor}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Por evento</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-muted-foreground">
              <tr>
                <th className="py-1">Evento</th>
                <th>Vendidos</th>
                <th>Cortesias</th>
                <th>Receita</th>
                <th>Check-ins</th>
                <th>Cancel.</th>
              </tr>
            </thead>
            <tbody>
              {data?.por_evento.map((e) => (
                <tr key={e.evento_id} className="border-t border-border">
                  <td className="py-1.5 text-foreground">
                    {e.titulo} <span className="text-xs text-muted-foreground">({e.status})</span>
                  </td>
                  <td>{e.vendidos}</td>
                  <td>{e.cortesias}</td>
                  <td>{formatarCentavos(e.receita_centavos)}</td>
                  <td>
                    {e.checkins} <span className="text-xs text-muted-foreground">({Math.round(e.comparecimento * 100)}%)</span>
                  </td>
                  <td>{e.cancelados}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Vendas por dia</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <div className="flex gap-2">
            <Input type="date" value={de} onChange={(e) => setDe(e.target.value)} />
            <Input type="date" value={ate} onChange={(e) => setAte(e.target.value)} />
          </div>
          {data?.vendas_por_dia.length === 0 && <p className="text-sm text-muted-foreground">Sem vendas no período.</p>}
          {data?.vendas_por_dia.map((d) => (
            <div key={d.dia} className="grid grid-cols-[6rem_1fr_9rem] items-center gap-2 text-sm">
              <span className="text-muted-foreground">{new Date(d.dia + 'T12:00:00').toLocaleDateString('pt-BR')}</span>
              <div className="h-3 rounded bg-muted">
                <div className="h-3 rounded bg-primary" style={{ width: `${(d.quantidade / maxDia) * 100}%` }} />
              </div>
              <span className="text-right text-foreground">
                {d.quantidade} · {formatarCentavos(d.receita_centavos)}
              </span>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  )
}
