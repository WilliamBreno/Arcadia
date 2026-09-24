import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Cupom = {
  id: number
  codigo: string
  tipo: 'percentual' | 'valor'
  valor: number
  max_usos: number | null
  usos: number
  ativo: boolean
}

export function CuponsCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({ codigo: '', tipo: 'percentual', valor: '10', maxUsos: '' })
  const [erro, setErro] = useState<string | null>(null)

  const { data: cupons } = useQuery({
    queryKey: ['org-evento-cupons', eventoId],
    queryFn: () => api<Cupom[]>(`/org/eventos/${eventoId}/cupons`),
  })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-evento-cupons', eventoId] })

  const criar = async () => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/cupons`, {
        method: 'POST',
        body: {
          codigo: form.codigo,
          tipo: form.tipo,
          valor: form.tipo === 'valor' ? Math.round(Number(form.valor) * 100) : Number(form.valor),
          max_usos: form.maxUsos ? Number(form.maxUsos) : null,
        },
      })
      setForm({ ...form, codigo: '' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar cupom')
    }
  }

  const desativar = async (id: number) => {
    await api(`/org/eventos/${eventoId}/cupons/${id}`, { method: 'DELETE' })
    invalidar()
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Cupons de desconto</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="text-xs text-muted-foreground">O desconto vale só sobre o preço do ingresso (a taxa e a garantia não mudam) e sai do seu repasse.</p>
        {cupons?.length === 0 && <p className="text-sm text-muted-foreground">Nenhum cupom criado.</p>}
        {cupons?.map((c) => (
          <div key={c.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <div>
              <p className="font-mono text-sm font-medium text-foreground">
                {c.codigo} {!c.ativo && <span className="text-xs text-muted-foreground">(desativado)</span>}
              </p>
              <p className="text-xs text-muted-foreground">
                {c.tipo === 'percentual' ? `${c.valor}%` : formatarCentavos(c.valor)} · {c.usos}
                {c.max_usos ? `/${c.max_usos}` : ''} uso(s)
              </p>
            </div>
            {c.ativo && (
              <Button variant="destructive" size="sm" onClick={() => desativar(c.id)}>
                Desativar
              </Button>
            )}
          </div>
        ))}

        <div className="grid grid-cols-[1.5fr_1fr_1fr_1fr_auto] items-end gap-2 border-t border-border pt-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cupom-codigo">Código</Label>
            <Input id="cupom-codigo" value={form.codigo} onChange={(e) => setForm({ ...form, codigo: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cupom-tipo">Tipo</Label>
            <select
              id="cupom-tipo"
              className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
              value={form.tipo}
              onChange={(e) => setForm({ ...form, tipo: e.target.value })}
            >
              <option value="percentual">%</option>
              <option value="valor">R$</option>
            </select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cupom-valor">Valor</Label>
            <Input id="cupom-valor" type="number" min="1" value={form.valor} onChange={(e) => setForm({ ...form, valor: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cupom-max">Máx. usos</Label>
            <Input id="cupom-max" type="number" min="1" value={form.maxUsos} onChange={(e) => setForm({ ...form, maxUsos: e.target.value })} />
          </div>
          <Button type="button" onClick={criar} disabled={!form.codigo}>
            Criar
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}
