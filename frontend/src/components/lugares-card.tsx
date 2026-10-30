import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { MapaLugares } from '@/components/mapa-lugares'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import { formatarCentavos, type TipoIngresso } from '@/lib/evento'
import type { SetorMapa } from '@/lib/lugares'

type Form = {
  nome: string
  formato: 'fileiras' | 'mesas'
  tipos: number[]
  fileiras: string
  lugares: string
  mesas: string
  primeira: string
  cadeiras: string
  inteira: boolean
}

const vazio: Form = { nome: '', formato: 'fileiras', tipos: [], fileiras: '10', lugares: '12', mesas: '10', primeira: '1', cadeiras: '6', inteira: false }

// Item 9.5: o organizador monta setores de fileiras ou de mesas e liga
// cada setor aos tipos de ingresso que dão o preço (inteira, meia, lotes).
export function LugaresCard({ eventoId, tipos }: { eventoId: number; tipos: TipoIngresso[] }) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<Form>(vazio)
  const [erro, setErro] = useState<string | null>(null)
  const [salvando, setSalvando] = useState(false)
  const [editando, setEditando] = useState<{ id: number; nome: string; tipos: number[] } | null>(null)

  const { data } = useQuery({
    queryKey: ['org-lugares', eventoId],
    queryFn: () => api<{ setores: SetorMapa[] }>(`/org/eventos/${eventoId}/lugares`),
  })
  const setores = data?.setores ?? []
  const invalidar = () => {
    queryClient.invalidateQueries({ queryKey: ['org-lugares', eventoId] })
    queryClient.invalidateQueries({ queryKey: ['lugares'] })
  }
  const executar = async (acao: () => Promise<unknown>) => {
    setErro(null)
    setSalvando(true)
    try {
      await acao()
      invalidar()
      return true
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar o mapa.')
      return false
    } finally {
      setSalvando(false)
    }
  }

  const totalNovo =
    form.formato === 'fileiras' ? Number(form.fileiras) * Number(form.lugares) || 0 : Number(form.mesas) * Number(form.cadeiras) || 0

  const criar = async () => {
    const ok = await executar(() =>
      api(`/org/eventos/${eventoId}/setores`, {
        method: 'POST',
        body: {
          nome: form.nome,
          formato: form.formato,
          tipo_ingresso_ids: form.tipos,
          fileiras: Number(form.fileiras),
          lugares_por_fileira: Number(form.lugares),
          mesas: Number(form.mesas),
          primeira_mesa: Number(form.primeira),
          cadeiras_por_mesa: Number(form.cadeiras),
          mesa_inteira: form.inteira,
        },
      }),
    )
    if (ok) setForm({ ...vazio, formato: form.formato })
  }

  const marcar = (lista: number[], id: number) => (lista.includes(id) ? lista.filter((x) => x !== id) : [...lista, id])
  const nomeTipo = (id: number) => tipos.find((t) => t.id === id)
  const tiposDePlateia = tipos

  return (
    <Card>
      <CardHeader>
        <CardTitle>Mapa de lugares</CardTitle>
        <p className="text-sm text-muted-foreground">
          Para teatro, auditório ou jantar: o público escolhe a cadeira no mapa. Cada setor usa o preço dos tipos de ingresso que você ligar
          a ele (por exemplo, inteira e meia da plateia). Sem setores, a venda segue sem lugar marcado.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        {setores.length > 0 && (
          <div className="flex flex-col gap-4">
            <MapaLugares setores={setores} />
            <ul className="flex flex-col gap-3">
              {setores.map((st) => (
                <li key={st.id} className="flex flex-col gap-2 rounded-lg border border-border p-3">
                  {editando?.id === st.id ? (
                    <div className="flex flex-col gap-3">
                      <div className="flex flex-col gap-1.5">
                        <Label htmlFor={`setor-nome-${st.id}`}>Nome do setor</Label>
                        <Input id={`setor-nome-${st.id}`} value={editando.nome} onChange={(e) => setEditando({ ...editando, nome: e.target.value })} />
                      </div>
                      <TiposSetor tipos={tiposDePlateia} marcados={editando.tipos} aoMarcar={(id) => setEditando({ ...editando, tipos: marcar(editando.tipos, id) })} />
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          disabled={salvando}
                          onClick={async () => {
                            const ok = await executar(() =>
                              api(`/org/eventos/${eventoId}/setores/${st.id}`, { method: 'PUT', body: { nome: editando.nome, tipo_ingresso_ids: editando.tipos } }),
                            )
                            if (ok) setEditando(null)
                          }}
                        >
                          Salvar setor
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => setEditando(null)}>
                          Cancelar
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <div className="flex flex-wrap items-baseline justify-between gap-2">
                        <span className="font-medium text-foreground">{st.nome}</span>
                        <span className="text-xs text-muted-foreground tabular-nums">
                          {st.total - st.livres} de {st.total} ocupados
                        </span>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {st.tipo_ingresso_ids
                          .map((id) => nomeTipo(id))
                          .filter(Boolean)
                          .map((t) => `${t!.nome} (${formatarCentavos(t!.preco_centavos)})`)
                          .join(', ') || 'Nenhum tipo ligado'}
                      </p>
                      {st.formato === 'mesas' && (
                        <div className="flex flex-wrap gap-1.5" aria-label={`Venda das mesas de ${st.nome}`}>
                          {st.grupos.map((g) => (
                            <Button
                              key={g.grupo}
                              size="sm"
                              variant={g.mesa_inteira ? 'default' : 'outline'}
                              disabled={salvando}
                              aria-pressed={g.mesa_inteira}
                              onClick={() =>
                                executar(() =>
                                  api(`/org/eventos/${eventoId}/setores/${st.id}/mesas/${encodeURIComponent(g.grupo)}`, {
                                    method: 'PUT',
                                    body: { mesa_inteira: !g.mesa_inteira },
                                  }),
                                )
                              }
                            >
                              Mesa {g.grupo}: {g.mesa_inteira ? 'inteira' : 'avulsa'}
                            </Button>
                          ))}
                        </div>
                      )}
                      <div className="flex gap-2">
                        <Button size="sm" variant="outline" onClick={() => setEditando({ id: st.id, nome: st.nome, tipos: st.tipo_ingresso_ids })}>
                          Editar
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={salvando}
                          onClick={() => {
                            if (window.confirm(`Apagar o setor ${st.nome} e as cadeiras dele?`)) {
                              executar(() => api(`/org/eventos/${eventoId}/setores/${st.id}`, { method: 'DELETE' }))
                            }
                          }}
                        >
                          Apagar
                        </Button>
                      </div>
                    </>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        <form
          className="flex flex-col gap-4 rounded-lg border border-dashed border-border p-4"
          onSubmit={(e) => {
            e.preventDefault()
            criar()
          }}
        >
          <h3 className="text-sm font-semibold text-foreground">Novo setor</h3>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="setor-nome">Nome</Label>
            <Input id="setor-nome" placeholder="Plateia, Camarote, Salão…" value={form.nome} onChange={(e) => setForm({ ...form, nome: e.target.value })} />
          </div>
          <fieldset className="flex flex-col gap-2">
            <legend className="mb-1 text-sm font-medium text-foreground">Formato</legend>
            <div className="flex flex-wrap gap-2">
              {(['fileiras', 'mesas'] as const).map((f) => (
                <Button key={f} type="button" size="sm" variant={form.formato === f ? 'default' : 'outline'} aria-pressed={form.formato === f} onClick={() => setForm({ ...form, formato: f })}>
                  {f === 'fileiras' ? 'Fileiras (teatro)' : 'Mesas (jantar, show)'}
                </Button>
              ))}
            </div>
          </fieldset>
          {form.formato === 'fileiras' ? (
            <div className="grid grid-cols-2 gap-3">
              <Campo id="setor-fileiras" rotulo="Fileiras (A, B, C…)" valor={form.fileiras} aoMudar={(v) => setForm({ ...form, fileiras: v })} max={52} />
              <Campo id="setor-lugares" rotulo="Lugares por fileira" valor={form.lugares} aoMudar={(v) => setForm({ ...form, lugares: v })} max={60} />
            </div>
          ) : (
            <>
              <div className="grid grid-cols-3 gap-3">
                <Campo id="setor-mesas" rotulo="Mesas" valor={form.mesas} aoMudar={(v) => setForm({ ...form, mesas: v })} max={300} />
                <Campo id="setor-primeira" rotulo="Começa na mesa" valor={form.primeira} aoMudar={(v) => setForm({ ...form, primeira: v })} max={9999} />
                <Campo id="setor-cadeiras" rotulo="Cadeiras por mesa" valor={form.cadeiras} aoMudar={(v) => setForm({ ...form, cadeiras: v })} max={20} />
              </div>
              <label className="flex items-start gap-2 text-sm text-foreground">
                <input type="checkbox" className="mt-0.5" checked={form.inteira} onChange={(e) => setForm({ ...form, inteira: e.target.checked })} />
                <span>
                  Vender cada mesa inteira (quem compra leva todas as cadeiras, com um QR por cadeira). Depois dá para mudar mesa por mesa.
                </span>
              </label>
            </>
          )}
          <TiposSetor tipos={tiposDePlateia} marcados={form.tipos} aoMarcar={(id) => setForm({ ...form, tipos: marcar(form.tipos, id) })} />
          <p className="text-xs text-muted-foreground">
            Serão criados {totalNovo} lugares. A quantidade de cada tipo de ingresso continua valendo como limite: deixe-a igual ou maior que
            as cadeiras do setor.
          </p>
          {erro && (
            <p role="alert" className="text-sm text-destructive">
              {erro}
            </p>
          )}
          <Button type="submit" disabled={salvando || !form.nome.trim() || form.tipos.length === 0} className="self-start">
            Criar setor
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

function TiposSetor({ tipos, marcados, aoMarcar }: { tipos: TipoIngresso[]; marcados: number[]; aoMarcar: (id: number) => void }) {
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-1 text-sm font-medium text-foreground">Tipos de ingresso deste setor</legend>
      {tipos.length === 0 && <p className="text-sm text-muted-foreground">Crie os tipos de ingresso primeiro.</p>}
      {tipos.map((t) => (
        <label key={t.id} className="flex items-center gap-2 text-sm text-foreground">
          <input type="checkbox" checked={marcados.includes(t.id)} onChange={() => aoMarcar(t.id)} />
          {t.nome} · {formatarCentavos(t.preco_centavos)}
        </label>
      ))}
    </fieldset>
  )
}

function Campo({ id, rotulo, valor, aoMudar, max }: { id: string; rotulo: string; valor: string; aoMudar: (v: string) => void; max: number }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{rotulo}</Label>
      <Input id={id} type="number" inputMode="numeric" min={1} max={max} value={valor} onChange={(e) => aoMudar(e.target.value)} />
    </div>
  )
}
