import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export type ItemCronograma = { id: number; titulo: string; descricao: string; local: string; inicio_em: string; fim_em: string | null }

export function ListaCronograma({ itens }: { itens: ItemCronograma[] }) {
  return (
    <div className="flex flex-col gap-2">
      {itens.map((i) => (
        <div key={i.id} className="flex gap-3 border-b border-border pb-2 text-sm">
          <span className="w-28 shrink-0 text-muted-foreground">
            {new Date(i.inicio_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })}
          </span>
          <div>
            <p className="font-medium text-foreground">{i.titulo}</p>
            {(i.local || i.descricao) && <p className="text-xs text-muted-foreground">{[i.local, i.descricao].filter(Boolean).join(' · ')}</p>}
          </div>
        </div>
      ))}
    </div>
  )
}

export function CronogramaCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({ titulo: '', local: '', inicio: '', fim: '' })
  const [erro, setErro] = useState<string | null>(null)
  const { data } = useQuery({ queryKey: ['org-cronograma', eventoId], queryFn: () => api<ItemCronograma[]>(`/org/eventos/${eventoId}/cronograma`) })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-cronograma', eventoId] })

  const criar = async () => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/cronograma`, {
        method: 'POST',
        body: {
          titulo: form.titulo,
          local: form.local,
          inicio_em: new Date(form.inicio).toISOString(),
          fim_em: form.fim ? new Date(form.fim).toISOString() : null,
        },
      })
      setForm({ titulo: '', local: '', inicio: '', fim: '' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar item')
    }
  }

  const excluir = async (id: number) => {
    await api(`/org/eventos/${eventoId}/cronograma/${id}`, { method: 'DELETE' })
    invalidar()
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Cronograma (público)</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {data?.length === 0 && <p className="text-sm text-muted-foreground">Nenhuma atividade cadastrada.</p>}
        {data?.map((i) => (
          <div key={i.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
            <span className="text-foreground">
              {new Date(i.inicio_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })} — {i.titulo}
              {i.local ? ` (${i.local})` : ''}
            </span>
            <Button variant="destructive" size="sm" onClick={() => excluir(i.id)}>
              Excluir
            </Button>
          </div>
        ))}
        <div className="grid grid-cols-[1.5fr_1fr_1fr_1fr_auto] items-end gap-2 border-t border-border pt-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cg-titulo">Atividade</Label>
            <Input id="cg-titulo" value={form.titulo} onChange={(e) => setForm({ ...form, titulo: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cg-local">Local/palco</Label>
            <Input id="cg-local" value={form.local} onChange={(e) => setForm({ ...form, local: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cg-inicio">Início</Label>
            <Input id="cg-inicio" type="datetime-local" value={form.inicio} onChange={(e) => setForm({ ...form, inicio: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cg-fim">Fim</Label>
            <Input id="cg-fim" type="datetime-local" value={form.fim} onChange={(e) => setForm({ ...form, fim: e.target.value })} />
          </div>
          <Button type="button" onClick={criar} disabled={!form.titulo || !form.inicio}>
            Adicionar
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}

type FichaOrdem = { id: number; nome: string; nome_artistico: string; papel: string; status: string; ordem_apresentacao: number | null; tipo_apresentacao: string | null }

// Ordem de apresentação dos participantes aprovados (visível só para jurados e organizador).
export function OrdemApresentacaoCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [erro, setErro] = useState<string | null>(null)
  const { data } = useQuery({
    queryKey: ['org-evento-participantes', eventoId],
    queryFn: () => api<FichaOrdem[]>(`/org/eventos/${eventoId}/participantes`),
  })

  const aprovados = (data ?? [])
    .filter((f) => f.papel === 'participante' && f.status === 'aprovado')
    .sort((a, b) => (a.ordem_apresentacao ?? 9999) - (b.ordem_apresentacao ?? 9999) || a.id - b.id)

  const mover = async (indice: number, delta: number) => {
    const novo = [...aprovados]
    const alvo = indice + delta
    if (alvo < 0 || alvo >= novo.length) return
    ;[novo[indice], novo[alvo]] = [novo[alvo], novo[indice]]
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/ordem-apresentacao`, { method: 'PUT', body: { ficha_ids: novo.map((f) => f.id) } })
      queryClient.invalidateQueries({ queryKey: ['org-evento-participantes', eventoId] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao reordenar')
    }
  }

  if (aprovados.length === 0) return null
  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Ordem de apresentação</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <p className="text-xs text-muted-foreground">Os jurados veem os participantes nesta ordem.</p>
        {aprovados.map((f, i) => (
          <div key={f.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
            <span className="text-foreground">
              {i + 1}. {f.nome}
              {f.nome_artistico ? ` (${f.nome_artistico})` : ''} <span className="text-xs text-muted-foreground">{f.tipo_apresentacao}</span>
            </span>
            <span className="flex gap-1">
              <Button variant="outline" size="sm" disabled={i === 0} onClick={() => mover(i, -1)}>
                ▲
              </Button>
              <Button variant="outline" size="sm" disabled={i === aprovados.length - 1} onClick={() => mover(i, 1)}>
                ▼
              </Button>
            </span>
          </div>
        ))}
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}
