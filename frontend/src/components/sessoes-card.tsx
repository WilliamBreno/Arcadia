import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export type Sessao = { id: number; titulo: string; inicio_em: string; fim_em: string | null; status: 'ativa' | 'cancelada' }

export function rotuloSessao(s: Sessao): string {
  const data = new Date(s.inicio_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
  return s.titulo ? `${s.titulo} (${data})` : data
}

export function SessoesCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({ titulo: '', inicio: '', fim: '' })
  const [erro, setErro] = useState<string | null>(null)
  const { data } = useQuery({ queryKey: ['org-sessoes', eventoId], queryFn: () => api<Sessao[]>(`/org/eventos/${eventoId}/sessoes`) })
  const invalidar = () => {
    queryClient.invalidateQueries({ queryKey: ['org-sessoes', eventoId] })
    queryClient.invalidateQueries({ queryKey: ['org-evento'] })
    queryClient.invalidateQueries({ queryKey: ['org-evento-ingressos'] })
  }

  const criar = async () => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/sessoes`, {
        method: 'POST',
        body: {
          titulo: form.titulo,
          inicio_em: new Date(form.inicio).toISOString(),
          fim_em: form.fim ? new Date(form.fim).toISOString() : null,
        },
      })
      setForm({ titulo: '', inicio: '', fim: '' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar sessão')
    }
  }

  const excluir = async (id: number) => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/sessoes/${id}`, { method: 'DELETE' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao excluir')
    }
  }

  const cancelar = async (s: Sessao) => {
    const motivo = window.prompt(
      `Cancelar a sessão "${rotuloSessao(s)}"?\n\nSerão reembolsados integralmente os ingressos válidos SÓ para esta sessão. Ingressos do evento todo continuam valendo.\n\nMotivo:`,
    )
    if (!motivo || motivo.trim().length < 3) return
    if (!confirm('Confirmar cancelamento da sessão? Não pode ser desfeito.')) return
    setErro(null)
    try {
      const r = await api<{ reembolsos_concluidos: number; falhas: number[] }>(`/org/eventos/${eventoId}/sessoes/${s.id}/cancelar`, {
        method: 'POST',
        body: { motivo },
      })
      if (r.falhas.length > 0) alert(`Sessão cancelada. ${r.reembolsos_concluidos} reembolso(s) feito(s); ${r.falhas.length} falharam — veja o painel admin.`)
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao cancelar sessão')
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Sessões (várias datas)</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-xs text-muted-foreground">
          Com sessões, o início e o fim do evento passam a ser calculados por elas; o repasse sai depois da última. O estoque é único; um ingresso
          vale para todas as sessões (uma entrada em cada), a menos que o tipo de ingresso seja restrito a uma sessão.
        </p>
        {data?.length === 0 && <p className="text-sm text-muted-foreground">Sem sessões — evento de data única.</p>}
        {data?.map((s) => (
          <div key={s.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
            <span className={s.status === 'cancelada' ? 'text-muted-foreground line-through' : 'text-foreground'}>
              {rotuloSessao(s)}
              {s.status === 'cancelada' && ' — cancelada'}
            </span>
            {s.status === 'ativa' && (
              <span className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => excluir(s.id)}>
                  Excluir
                </Button>
                <Button variant="destructive" size="sm" onClick={() => cancelar(s)}>
                  Cancelar sessão
                </Button>
              </span>
            )}
          </div>
        ))}
        <div className="grid grid-cols-[1fr_1fr_1fr_auto] items-end gap-2 border-t border-border pt-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ses-titulo">Nome (opcional)</Label>
            <Input id="ses-titulo" placeholder="Dia 1" value={form.titulo} onChange={(e) => setForm({ ...form, titulo: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ses-inicio">Início</Label>
            <Input id="ses-inicio" type="datetime-local" value={form.inicio} onChange={(e) => setForm({ ...form, inicio: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ses-fim">Fim</Label>
            <Input id="ses-fim" type="datetime-local" value={form.fim} onChange={(e) => setForm({ ...form, fim: e.target.value })} />
          </div>
          <Button onClick={criar} disabled={!form.inicio}>
            Adicionar
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}
