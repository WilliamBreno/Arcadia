import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import type { Criterio } from '@/components/avaliar-ficha'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Ranking = Record<string, { posicao: number; ficha_id?: number; nome: string; nome_artistico?: string; nota_final: number; jurados?: number }[]>

const rotuloTipo: Record<string, string> = { cosplay: 'Cosplay', danca: 'Dança', canto: 'Canto', atuacao: 'Atuação', '': 'Sem categoria' }

export function ListaRanking({ ranking }: { ranking: Ranking }) {
  const tipos = Object.keys(ranking)
  if (tipos.length === 0) return <p className="text-sm text-muted-foreground">Ainda não há avaliações finalizadas.</p>
  return (
    <div className="flex flex-col gap-3">
      {tipos.map((tipo) => (
        <div key={tipo}>
          <p className="mb-1 text-sm font-medium text-foreground">{rotuloTipo[tipo] ?? tipo}</p>
          {ranking[tipo].map((l, i) => (
            <div key={`${tipo}-${l.ficha_id ?? i}`} className="flex items-center justify-between border-b border-border py-1 text-sm">
              <span className="text-foreground">
                {l.posicao}º · {l.nome}
                {l.nome_artistico ? ` (${l.nome_artistico})` : ''}
              </span>
              <span className="text-muted-foreground">
                {l.nota_final.toFixed(2)}
                {l.jurados ? ` · ${l.jurados} jurado(s)` : ''}
              </span>
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}

export function CriteriosCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({ nome: '', peso: '1', min: '0', max: '10', passo: '1', tipo: '' })
  const [erro, setErro] = useState<string | null>(null)
  const { data } = useQuery({ queryKey: ['org-criterios', eventoId], queryFn: () => api<Criterio[]>(`/org/eventos/${eventoId}/criterios`) })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-criterios', eventoId] })

  const criar = async () => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/criterios`, {
        method: 'POST',
        body: {
          nome: form.nome,
          peso: Number(form.peso),
          nota_min: Number(form.min),
          nota_max: Number(form.max),
          passo: Number(form.passo),
          tipo_apresentacao: form.tipo || null,
          ordem: (data?.length ?? 0) + 1,
        },
      })
      setForm({ ...form, nome: '' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar critério')
    }
  }

  const excluir = async (id: number) => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/criterios/${id}`, { method: 'DELETE' })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao excluir')
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Critérios de avaliação</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-xs text-muted-foreground">
          Cada nota é normalizada para 0–10 e ponderada pelo peso. Desempate: maior média no critério de maior peso. Critério com
          notas lançadas não pode mais ser alterado.
        </p>
        {data?.map((c) => (
          <div key={c.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
            <span className="text-foreground">
              {c.nome} · peso {c.peso} · {c.nota_min}–{c.nota_max} (passo {c.passo}){c.tipo_apresentacao ? ` · só ${rotuloTipo[c.tipo_apresentacao]}` : ''}
            </span>
            <Button variant="destructive" size="sm" onClick={() => excluir(c.id)}>
              Excluir
            </Button>
          </div>
        ))}
        <div className="grid grid-cols-[1.5fr_0.6fr_0.6fr_0.6fr_0.6fr_1fr_auto] items-end gap-2 border-t border-border pt-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cr-nome">Nome</Label>
            <Input id="cr-nome" value={form.nome} onChange={(e) => setForm({ ...form, nome: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cr-peso">Peso</Label>
            <Input id="cr-peso" type="number" min="0.1" step="0.1" value={form.peso} onChange={(e) => setForm({ ...form, peso: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cr-min">Mín.</Label>
            <Input id="cr-min" type="number" value={form.min} onChange={(e) => setForm({ ...form, min: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cr-max">Máx.</Label>
            <Input id="cr-max" type="number" value={form.max} onChange={(e) => setForm({ ...form, max: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cr-passo">Passo</Label>
            <Input id="cr-passo" type="number" min="0.1" step="0.1" value={form.passo} onChange={(e) => setForm({ ...form, passo: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cr-tipo">Categoria</Label>
            <select
              id="cr-tipo"
              className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
              value={form.tipo}
              onChange={(e) => setForm({ ...form, tipo: e.target.value })}
            >
              <option value="">Todas</option>
              <option value="cosplay">Cosplay</option>
              <option value="danca">Dança</option>
              <option value="canto">Canto</option>
              <option value="atuacao">Atuação</option>
            </select>
          </div>
          <Button type="button" onClick={criar} disabled={!form.nome}>
            Criar
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}

export function RankingCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const { data } = useQuery({
    queryKey: ['org-ranking', eventoId],
    queryFn: () => api<{ ranking: Ranking; resultado_liberado_em: string | null }>(`/org/eventos/${eventoId}/ranking`),
  })

  const definir = async (liberar: boolean) => {
    if (liberar && !confirm('Liberar o resultado publicamente na página do evento?')) return
    await api(`/org/eventos/${eventoId}/resultado`, { method: 'POST', body: { liberar } })
    queryClient.invalidateQueries({ queryKey: ['org-ranking', eventoId] })
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Ranking e resultado</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-xs text-muted-foreground">
          Só notas finalizadas entram. Só você vê este ranking até liberar; o público vê nome artístico (ou primeiro nome + inicial) e a nota.
        </p>
        {data && <ListaRanking ranking={data.ranking} />}
        {data?.resultado_liberado_em ? (
          <Button variant="outline" size="sm" className="w-fit" onClick={() => definir(false)}>
            Ocultar resultado (liberado em {new Date(data.resultado_liberado_em).toLocaleString('pt-BR')})
          </Button>
        ) : (
          <Button size="sm" className="w-fit" onClick={() => definir(true)}>
            Liberar resultado ao público
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
