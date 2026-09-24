import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

type Relatorios = {
  receita: { itens: number; taxa_centavos: number; garantia_centavos: number; total_centavos: number }
  reembolsos: { tipo: string; status: string; quantidade: number; valor_centavos: number }[]
  reembolsos_falhos: number
}

type Reembolso = {
  id: number
  evento_titulo: string
  codigo: string
  titular_nome: string
  valor_centavos: number
  tipo: string
  motivo: string
}

export default function AdminRelatorios() {
  const queryClient = useQueryClient()
  const [erro, setErro] = useState<string | null>(null)

  const { data: rel, error } = useQuery({ queryKey: ['admin-relatorios'], queryFn: () => api<Relatorios>('/admin/relatorios') })
  const { data: falhos } = useQuery({
    queryKey: ['admin-reembolsos-falhos'],
    queryFn: () => api<Reembolso[]>('/admin/reembolsos?status=falhou'),
  })

  const reprocessar = async (id: number) => {
    setErro(null)
    try {
      await api(`/admin/reembolsos/${id}/reprocessar`, { method: 'POST' })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao reprocessar')
    }
    queryClient.invalidateQueries({ queryKey: ['admin-reembolsos-falhos'] })
    queryClient.invalidateQueries({ queryKey: ['admin-relatorios'] })
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-destructive">Acesso restrito ao admin da plataforma.</p>
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <h1 className="mb-6 text-2xl font-semibold text-foreground">Relatórios</h1>

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Receita da plataforma (eventos já iniciados)</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-3 gap-4 text-sm">
          <div>
            <p className="text-xs text-muted-foreground">Taxas</p>
            <p className="text-lg font-semibold text-foreground">{formatarCentavos(rel?.receita.taxa_centavos ?? 0)}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Garantias</p>
            <p className="text-lg font-semibold text-foreground">{formatarCentavos(rel?.receita.garantia_centavos ?? 0)}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Total ({rel?.receita.itens ?? 0} itens)</p>
            <p className="text-lg font-semibold text-foreground">{formatarCentavos(rel?.receita.total_centavos ?? 0)}</p>
          </div>
        </CardContent>
      </Card>

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Reembolsos</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-1 text-sm">
          {rel?.reembolsos.length === 0 && <p className="text-muted-foreground">Nenhum reembolso.</p>}
          {rel?.reembolsos.map((r) => (
            <div key={`${r.tipo}-${r.status}`} className="flex justify-between">
              <span className="text-muted-foreground">
                {r.tipo} · {r.status}
              </span>
              <span className="text-foreground">
                {r.quantidade} · {formatarCentavos(r.valor_centavos)}
              </span>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Falhas de reembolso ({falhos?.length ?? 0})</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {erro && <p className="text-sm text-destructive">{erro}</p>}
          {falhos?.map((f) => (
            <div key={f.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
              <div>
                <p className="text-foreground">
                  {f.titular_nome} · {f.codigo}
                </p>
                <p className="text-xs text-muted-foreground">
                  {f.evento_titulo} · {f.tipo} · {formatarCentavos(f.valor_centavos)}
                </p>
              </div>
              <Button size="sm" onClick={() => reprocessar(f.id)}>
                Reprocessar
              </Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  )
}
