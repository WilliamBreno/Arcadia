import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import { formatarCentavos, type TipoIngresso } from '@/lib/evento'
import type { Adicional, Extras, Pacote } from '@/lib/extras'

const reais = (texto: string) => {
  const n = Number(texto.replace(/\./g, '').replace(',', '.'))
  return Number.isNaN(n) ? -1 : Math.round(n * 100)
}
const paraTexto = (c: number) => (c / 100).toFixed(2).replace('.', ',')

type FormAdicional = { id: number; nome: string; descricao: string; preco: string; quantidade: string; max: string; ativo: boolean }
type FormPacote = { id: number; nome: string; descricao: string; preco: string; quantidade: string; max: string; ativo: boolean; itens: Record<number, string> }

const adicionalVazio: FormAdicional = { id: 0, nome: '', descricao: '', preco: '', quantidade: '100', max: '4', ativo: true }
const pacoteVazio: FormPacote = { id: 0, nome: '', descricao: '', preco: '', quantidade: '50', max: '4', ativo: true, itens: {} }

// Item 10.3: adicionais (camiseta, estacionamento...) e pacotes com preço fechado.
export function ExtrasCard({ eventoId, tipos }: { eventoId: number; tipos: TipoIngresso[] }) {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['org-extras', eventoId], queryFn: () => api<Extras>(`/org/eventos/${eventoId}/extras`) })
  const [fa, setFa] = useState<FormAdicional | null>(null)
  const [fp, setFp] = useState<FormPacote | null>(null)
  const [erro, setErro] = useState<string | null>(null)

  const executar = async (acao: () => Promise<unknown>) => {
    setErro(null)
    try {
      await acao()
      queryClient.invalidateQueries({ queryKey: ['org-extras', eventoId] })
      return true
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
      return false
    }
  }

  const salvarAdicional = async () => {
    if (!fa) return
    const corpo = { nome: fa.nome, descricao: fa.descricao, preco_centavos: reais(fa.preco), quantidade: Number(fa.quantidade), max_por_pedido: Number(fa.max), ativo: fa.ativo }
    const ok = await executar(() =>
      api(fa.id ? `/org/eventos/${eventoId}/adicionais/${fa.id}` : `/org/eventos/${eventoId}/adicionais`, { method: fa.id ? 'PUT' : 'POST', body: corpo }),
    )
    if (ok) setFa(null)
  }

  const salvarPacote = async () => {
    if (!fp) return
    const itens = Object.entries(fp.itens)
      .map(([id, q]) => ({ tipo_ingresso_id: Number(id), quantidade: Number(q) }))
      .filter((it) => it.quantidade > 0)
    const corpo = { nome: fp.nome, descricao: fp.descricao, preco_centavos: reais(fp.preco), quantidade: Number(fp.quantidade), max_por_pedido: Number(fp.max), ativo: fp.ativo, itens }
    const ok = await executar(() =>
      api(fp.id ? `/org/eventos/${eventoId}/pacotes/${fp.id}` : `/org/eventos/${eventoId}/pacotes`, { method: fp.id ? 'PUT' : 'POST', body: corpo }),
    )
    if (ok) setFp(null)
  }

  const editarAdicional = (a: Adicional) =>
    setFa({ id: a.id, nome: a.nome, descricao: a.descricao, preco: paraTexto(a.preco_centavos), quantidade: String(a.quantidade), max: String(a.max_por_pedido), ativo: a.ativo })
  const editarPacote = (p: Pacote) =>
    setFp({
      id: p.id, nome: p.nome, descricao: p.descricao, preco: paraTexto(p.preco_centavos), quantidade: String(p.quantidade), max: String(p.max_por_pedido), ativo: p.ativo,
      itens: Object.fromEntries(p.itens.map((it) => [it.tipo_ingresso_id, String(it.quantidade)])),
    })

  const escolhiveis = [...tipos.map((t) => ({ id: t.id, nome: t.nome, preco: t.preco_centavos })), ...(data?.adicionais ?? []).map((a) => ({ id: a.id, nome: `${a.nome} (adicional)`, preco: a.preco_centavos }))]
  const tabelaPacote = fp ? escolhiveis.reduce((s, t) => s + t.preco * (Number(fp.itens[t.id]) || 0), 0) : 0

  return (
    <Card>
      <CardHeader>
        <CardTitle>Adicionais e pacotes</CardTitle>
        <p className="text-sm text-muted-foreground">
          Adicionais (camiseta, estacionamento, kit, fichas) são vendidos junto com o ingresso, com a mesma taxa por item e as mesmas regras de reembolso, e
          entregues no evento pelo QR de cada um. Pacotes juntam ingressos e adicionais por um preço fechado.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <section aria-labelledby="titulo-org-adicionais" className="flex flex-col gap-3">
          <h3 id="titulo-org-adicionais" className="text-sm font-semibold text-foreground">
            Adicionais
          </h3>
          {data?.adicionais.map((a) => (
            <div key={a.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2 text-sm">
              <span className="min-w-0">
                <span className="font-medium text-foreground">{a.nome}</span>{' '}
                <span className="text-muted-foreground">
                  · {formatarCentavos(a.preco_centavos)} · {a.vendidos}/{a.quantidade} vendidos{a.ativo ? '' : ' · inativo'}
                </span>
              </span>
              <span className="flex gap-1">
                <Button size="sm" variant="outline" onClick={() => editarAdicional(a)}>
                  Editar
                </Button>
                {a.vendidos === 0 && (
                  <Button size="sm" variant="ghost" onClick={() => window.confirm(`Apagar ${a.nome}?`) && executar(() => api(`/org/eventos/${eventoId}/adicionais/${a.id}`, { method: 'DELETE' }))}>
                    Apagar
                  </Button>
                )}
              </span>
            </div>
          ))}
          {fa ? (
            <div className="flex flex-col gap-3 rounded-lg border border-dashed border-border p-3">
              <div className="grid gap-3 sm:grid-cols-2">
                <Campo id="ad-nome" rotulo="Nome" valor={fa.nome} aoMudar={(v) => setFa({ ...fa, nome: v })} />
                <Campo id="ad-desc" rotulo="Descrição (opcional)" valor={fa.descricao} aoMudar={(v) => setFa({ ...fa, descricao: v })} />
                <Campo id="ad-preco" rotulo="Preço (R$)" valor={fa.preco} aoMudar={(v) => setFa({ ...fa, preco: v })} decimal />
                <Campo id="ad-qtd" rotulo="Quantidade à venda" valor={fa.quantidade} aoMudar={(v) => setFa({ ...fa, quantidade: v })} numero />
                <Campo id="ad-max" rotulo="Máximo por pedido" valor={fa.max} aoMudar={(v) => setFa({ ...fa, max: v })} numero />
              </div>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input type="checkbox" checked={fa.ativo} onChange={(e) => setFa({ ...fa, ativo: e.target.checked })} /> À venda
              </label>
              <div className="flex gap-2">
                <Button size="sm" onClick={salvarAdicional} disabled={!fa.nome.trim() || reais(fa.preco) < 0}>
                  Salvar adicional
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setFa(null)}>
                  Cancelar
                </Button>
              </div>
            </div>
          ) : (
            <Button size="sm" variant="outline" className="self-start" onClick={() => setFa(adicionalVazio)}>
              Novo adicional
            </Button>
          )}
        </section>

        <section aria-labelledby="titulo-org-pacotes" className="flex flex-col gap-3">
          <h3 id="titulo-org-pacotes" className="text-sm font-semibold text-foreground">
            Pacotes
          </h3>
          {data?.pacotes.map((p) => (
            <div key={p.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2 text-sm">
              <span className="min-w-0">
                <span className="font-medium text-foreground">{p.nome}</span>{' '}
                <span className="text-muted-foreground">
                  · {p.itens.map((it) => `${it.quantidade}× ${it.nome}`).join(' + ')} · {formatarCentavos(p.preco_centavos)} (separado {formatarCentavos(p.tabela_centavos)}) · {p.vendidos}/
                  {p.quantidade} vendidos{p.ativo ? '' : ' · inativo'}
                </span>
              </span>
              <span className="flex gap-1">
                <Button size="sm" variant="outline" onClick={() => editarPacote(p)}>
                  Editar
                </Button>
                {p.vendidos === 0 && (
                  <Button size="sm" variant="ghost" onClick={() => window.confirm(`Apagar ${p.nome}?`) && executar(() => api(`/org/eventos/${eventoId}/pacotes/${p.id}`, { method: 'DELETE' }))}>
                    Apagar
                  </Button>
                )}
              </span>
            </div>
          ))}
          {fp ? (
            <div className="flex flex-col gap-3 rounded-lg border border-dashed border-border p-3">
              <div className="grid gap-3 sm:grid-cols-2">
                <Campo id="pc-nome" rotulo="Nome" valor={fp.nome} aoMudar={(v) => setFp({ ...fp, nome: v })} />
                <Campo id="pc-desc" rotulo="Descrição (opcional)" valor={fp.descricao} aoMudar={(v) => setFp({ ...fp, descricao: v })} />
              </div>
              <fieldset className="flex flex-col gap-2">
                <legend className="mb-1 text-sm font-medium text-foreground">O que vem no pacote</legend>
                {escolhiveis.map((t) => (
                  <div key={t.id} className="flex items-center justify-between gap-3 text-sm">
                    <label htmlFor={`pc-item-${t.id}`} className="text-foreground">
                      {t.nome} · {formatarCentavos(t.preco)}
                    </label>
                    <Input
                      id={`pc-item-${t.id}`}
                      className="w-20"
                      type="number"
                      min={0}
                      max={20}
                      value={fp.itens[t.id] ?? ''}
                      placeholder="0"
                      onChange={(e) => setFp({ ...fp, itens: { ...fp.itens, [t.id]: e.target.value } })}
                    />
                  </div>
                ))}
                <p className="text-xs text-muted-foreground">Separado, sai por {formatarCentavos(tabelaPacote)}. O pacote precisa ter pelo menos um ingresso.</p>
              </fieldset>
              <div className="grid gap-3 sm:grid-cols-3">
                <Campo id="pc-preco" rotulo="Preço do pacote (R$)" valor={fp.preco} aoMudar={(v) => setFp({ ...fp, preco: v })} decimal />
                <Campo id="pc-qtd" rotulo="Pacotes à venda" valor={fp.quantidade} aoMudar={(v) => setFp({ ...fp, quantidade: v })} numero />
                <Campo id="pc-max" rotulo="Máximo por pedido" valor={fp.max} aoMudar={(v) => setFp({ ...fp, max: v })} numero />
              </div>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input type="checkbox" checked={fp.ativo} onChange={(e) => setFp({ ...fp, ativo: e.target.checked })} /> À venda
              </label>
              <p className="text-xs text-muted-foreground">
                O preço é dividido entre os itens na proporção do preço de cada um; a taxa e o reembolso continuam por item. Com vendas, o preço e os
                itens não mudam.
              </p>
              <div className="flex gap-2">
                <Button size="sm" onClick={salvarPacote} disabled={!fp.nome.trim() || reais(fp.preco) <= 0}>
                  Salvar pacote
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setFp(null)}>
                  Cancelar
                </Button>
              </div>
            </div>
          ) : (
            <Button size="sm" variant="outline" className="self-start" onClick={() => setFp(pacoteVazio)}>
              Novo pacote
            </Button>
          )}
        </section>
        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

function Campo({ id, rotulo, valor, aoMudar, numero, decimal }: { id: string; rotulo: string; valor: string; aoMudar: (v: string) => void; numero?: boolean; decimal?: boolean }) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{rotulo}</Label>
      <Input id={id} inputMode={decimal ? 'decimal' : numero ? 'numeric' : undefined} type={numero ? 'number' : 'text'} min={numero ? 1 : undefined} value={valor} onChange={(e) => aoMudar(e.target.value)} />
    </div>
  )
}
