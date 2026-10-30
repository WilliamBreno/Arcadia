import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError, baixarArquivo } from '@/lib/api'
import type { Pergunta } from '@/lib/perguntas'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type Rascunho = {
  texto: string
  tipo: 'texto' | 'opcao'
  opcoes: string
  obrigatoria: boolean
  finalidade: string
  tipos: number[]
}
const vazio: Rascunho = { texto: '', tipo: 'texto', opcoes: '', obrigatoria: false, finalidade: '', tipos: [] }

// Perguntas personalizadas do checkout (item 7.6). Cada ingresso comprado
// responde; o organizador baixa a planilha. Respostas somem 90 dias depois
// do fim do evento.
export function PerguntasCard({ eventoId, tipos }: { eventoId: number; tipos: { id: number; nome: string }[] }) {
  const queryClient = useQueryClient()
  const chave = ['org-perguntas', eventoId]
  const { data: perguntas } = useQuery({ queryKey: chave, queryFn: () => api<Pergunta[]>(`/org/eventos/${eventoId}/perguntas`) })
  const [r, setR] = useState<Rascunho>(vazio)
  const [erro, setErro] = useState<string | null>(null)
  const [salvando, setSalvando] = useState(false)

  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setSalvando(true)
    try {
      await api(`/org/eventos/${eventoId}/perguntas`, {
        method: 'POST',
        body: {
          texto: r.texto,
          tipo: r.tipo,
          opcoes: r.tipo === 'opcao' ? r.opcoes.split('\n') : [],
          obrigatoria: r.obrigatoria,
          finalidade: r.finalidade,
          tipo_ingresso_ids: r.tipos,
          ordem: perguntas?.length ?? 0,
        },
      })
      setR(vazio)
      queryClient.invalidateQueries({ queryKey: chave })
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Não foi possível salvar.')
    } finally {
      setSalvando(false)
    }
  }

  const remover = async (id: number) => {
    await api(`/org/eventos/${eventoId}/perguntas/${id}`, { method: 'DELETE' }).catch(() => undefined)
    queryClient.invalidateQueries({ queryKey: chave })
  }

  const nomeTipos = (ids: number[]) =>
    ids.length === 0 ? 'Todos os ingressos' : ids.map((id) => tipos.find((t) => t.id === id)?.nome ?? '?').join(', ')

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Perguntas na compra</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Peça só o que você vai usar (ex.: tamanho de camiseta, como soube do evento). Evite dados sensíveis, como saúde,
          religião ou etnia; se precisar mesmo (ex.: restrição alimentar), explique o motivo. O comprador vê para que você pede, e
          as respostas são apagadas 90 dias depois do evento.
        </p>

        {(perguntas?.length ?? 0) > 0 && (
          <ul className="flex flex-col divide-y divide-border">
            {perguntas!.map((p) => (
              <li key={p.id} className="flex flex-wrap items-start justify-between gap-2 py-2 text-sm">
                <div className="min-w-0">
                  <p className="font-medium text-foreground">
                    {p.texto} {p.obrigatoria && <span className="text-xs font-normal text-muted-foreground">(obrigatória)</span>}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {p.tipo === 'opcao' ? `Opções: ${p.opcoes.join(', ')}` : 'Texto curto'} · {nomeTipos(p.tipo_ingresso_ids)}
                  </p>
                  <p className="text-xs text-muted-foreground">Para quê: {p.finalidade}</p>
                </div>
                <Button type="button" variant="ghost" size="sm" onClick={() => remover(p.id)}>
                  Remover
                </Button>
              </li>
            ))}
          </ul>
        )}

        <form onSubmit={salvar} className="flex flex-col gap-3 rounded-lg border border-border p-3">
          <p className="text-sm font-medium text-foreground">Nova pergunta</p>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pq-texto">Pergunta</Label>
            <Input
              id="pq-texto"
              value={r.texto}
              maxLength={200}
              onChange={(e) => setR({ ...r, texto: e.target.value })}
              placeholder="Tamanho da camiseta"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pq-finalidade">Para que você pede (o comprador vê)</Label>
            <Input
              id="pq-finalidade"
              value={r.finalidade}
              maxLength={200}
              onChange={(e) => setR({ ...r, finalidade: e.target.value })}
              placeholder="Para separar o kit de boas-vindas"
            />
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <Label htmlFor="pq-tipo">Resposta</Label>
              <select
                id="pq-tipo"
                value={r.tipo}
                onChange={(e) => setR({ ...r, tipo: e.target.value as Rascunho['tipo'] })}
                className="h-8 rounded-lg border border-border bg-background px-2 text-sm text-foreground"
              >
                <option value="texto">Texto curto</option>
                <option value="opcao">Escolher entre opções</option>
              </select>
            </div>
            <label className="flex items-center gap-2 text-sm text-foreground">
              <input type="checkbox" checked={r.obrigatoria} onChange={(e) => setR({ ...r, obrigatoria: e.target.checked })} />
              Obrigatória
            </label>
          </div>
          {r.tipo === 'opcao' && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="pq-opcoes">Opções (uma por linha)</Label>
              <Textarea
                id="pq-opcoes"
                rows={4}
                value={r.opcoes}
                onChange={(e) => setR({ ...r, opcoes: e.target.value })}
                placeholder={'P\nM\nG'}
              />
            </div>
          )}
          {tipos.length > 1 && (
            <fieldset className="flex flex-col gap-1">
              <legend className="text-sm text-foreground">Vale para (nenhum marcado = todos)</legend>
              <div className="flex flex-wrap gap-3">
                {tipos.map((t) => (
                  <label key={t.id} className="flex items-center gap-2 text-sm text-foreground">
                    <input
                      type="checkbox"
                      checked={r.tipos.includes(t.id)}
                      onChange={(e) =>
                        setR({ ...r, tipos: e.target.checked ? [...r.tipos, t.id] : r.tipos.filter((x) => x !== t.id) })
                      }
                    />
                    {t.nome}
                  </label>
                ))}
              </div>
            </fieldset>
          )}
          {erro && (
            <p role="alert" className="text-sm text-destructive">
              {erro}
            </p>
          )}
          <div>
            <Button type="submit" disabled={salvando}>
              {salvando ? 'Salvando…' : 'Adicionar pergunta'}
            </Button>
          </div>
        </form>

        <div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => baixarArquivo(`/org/eventos/${eventoId}/respostas.csv`, `respostas-evento-${eventoId}.csv`)}
          >
            Baixar respostas (planilha)
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
