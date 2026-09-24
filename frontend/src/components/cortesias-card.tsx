import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError, baixarArquivo } from '@/lib/api'
import type { TipoIngresso } from '@/lib/evento'
import type { VendaItem } from '@/lib/vendas'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export function CortesiasCard({ eventoId, tipos }: { eventoId: number; tipos: TipoIngresso[] }) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({ tipo: '', nome: '', email: '', quantidade: '1' })
  const [erro, setErro] = useState<string | null>(null)

  const { data: cortesias } = useQuery({
    queryKey: ['org-evento-cortesias', eventoId],
    queryFn: () => api<VendaItem[]>(`/org/eventos/${eventoId}/cortesias`),
  })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-evento-cortesias', eventoId] })

  const emitir = async () => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/cortesias`, {
        method: 'POST',
        body: {
          tipo_ingresso_id: Number(form.tipo || tipos[0]?.id),
          nome: form.nome,
          email: form.email,
          quantidade: Number(form.quantidade),
        },
      })
      setForm({ ...form, nome: '', email: '', quantidade: '1' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao emitir cortesia')
    }
  }

  const revogar = async (itemId: number) => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/cortesias/${itemId}`, { method: 'DELETE' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao revogar')
    }
  }

  const exportar = async (tipo: 'compradores' | 'participantes') => {
    setErro(null)
    try {
      await baixarArquivo(`/org/eventos/${eventoId}/exportar.csv?tipo=${tipo}`, `${tipo}-evento-${eventoId}.csv`)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao exportar')
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Cortesias e exportação</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="text-xs text-muted-foreground">
          Cortesia é um ingresso gratuito, sem taxa da plataforma, enviado por e-mail com link do QR. Ocupa vaga do estoque.
        </p>

        {cortesias?.map((c) => (
          <div key={c.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <div>
              <p className="text-sm font-medium text-foreground">{c.titular_nome}</p>
              <p className="text-xs text-muted-foreground">
                {c.tipo_ingresso_nome} · {c.codigo} · {c.status}
              </p>
            </div>
            {c.status === 'pago' && (
              <Button variant="destructive" size="sm" onClick={() => revogar(c.id)}>
                Revogar
              </Button>
            )}
          </div>
        ))}

        <div className="grid grid-cols-[1fr_1.2fr_1.2fr_0.6fr_auto] items-end gap-2 border-t border-border pt-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cort-tipo">Ingresso</Label>
            <select
              id="cort-tipo"
              className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
              value={form.tipo}
              onChange={(e) => setForm({ ...form, tipo: e.target.value })}
            >
              {tipos.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.nome}
                </option>
              ))}
            </select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cort-nome">Nome</Label>
            <Input id="cort-nome" value={form.nome} onChange={(e) => setForm({ ...form, nome: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cort-email">E-mail</Label>
            <Input id="cort-email" type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cort-qtd">Qtd</Label>
            <Input id="cort-qtd" type="number" min="1" max="50" value={form.quantidade} onChange={(e) => setForm({ ...form, quantidade: e.target.value })} />
          </div>
          <Button type="button" onClick={emitir} disabled={!form.nome || !form.email || tipos.length === 0}>
            Emitir
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}

        <div className="flex gap-2 border-t border-border pt-4">
          <Button variant="outline" size="sm" onClick={() => exportar('compradores')}>
            Exportar compradores (CSV)
          </Button>
          <Button variant="outline" size="sm" onClick={() => exportar('participantes')}>
            Exportar participantes (CSV)
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
