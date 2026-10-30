import { useQuery, useQueryClient } from '@tanstack/react-query'
import { X } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import {
  aprovarExpositor,
  atualizarTipoEspaco,
  criarMesas,
  criarTipoEspaco,
  excluirMesa,
  excluirTipoEspaco,
  painelExpositores,
  rejeitarExpositor,
  rotuloStatusExpositor,
  type ExpositorPainel,
  type PainelExpositores,
} from '@/lib/expositor'

const select = 'h-8 rounded-lg border border-border bg-background px-2.5 text-sm'

// Cartão do painel do evento (item 8.6): artist alley e expositores.
export function ExpositoresCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const chave = ['org-expositores', eventoId]
  const { data } = useQuery({ queryKey: chave, queryFn: () => painelExpositores(eventoId) })
  const [erro, setErro] = useState<string | null>(null)
  const [novoTipo, setNovoTipo] = useState({ nome: 'Mesa de artista', preco: '150', credenciais: '2' })
  const [lote, setLote] = useState({ tipo: '', setor: 'A', de: '1', ate: '10' })

  const rodar = async (f: () => Promise<unknown>) => {
    setErro(null)
    try {
      await f()
      await queryClient.invalidateQueries({ queryKey: chave })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir agora.')
    }
  }
  if (!data) return null

  const tipoDe = (id: number) => data.tipos.find((t) => t.id === id)
  const setores = [...new Set(data.mesas.map((m) => m.setor))]
  const ocupantes = new Map<number, ExpositorPainel>()
  data.expositores.forEach((x) => {
    if (x.espaco_id && (x.status === 'aprovado' || x.status === 'aguardando_pagamento')) ocupantes.set(x.espaco_id, x)
  })
  const pendentes = data.expositores.filter((x) => x.status === 'pendente')
  const demais = data.expositores.filter((x) => x.status !== 'pendente')
  const tipoLote = Number(lote.tipo) || data.tipos[0]?.id || 0

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Expositores e artist alley</CardTitle>
        <p className="text-sm text-muted-foreground text-pretty">
          Cadastre os tipos de espaço e as mesas. Expositores se candidatam pela página do evento; você aprova escolhendo a mesa. Espaço pago: a pessoa tem 48
          horas para pagar (a taxa Evve é somada para ela); o valor entra no seu repasse.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <section aria-labelledby="t-tipos" className="flex flex-col gap-2">
          <h3 id="t-tipos" className="text-sm font-medium text-foreground">
            Tipos de espaço
          </h3>
          {data.tipos.map((t) => (
            <div key={t.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2 text-sm">
              <span className="text-foreground">
                {t.nome} <span className="text-muted-foreground">· {t.preco_centavos ? formatarCentavos(t.preco_centavos) : 'gratuito'} · {t.credenciais} credencial(is)</span>
              </span>
              <span className="flex items-center gap-2">
                <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <input type="checkbox" checked={t.ativo} onChange={(e) => rodar(() => atualizarTipoEspaco(eventoId, { ...t, ativo: e.target.checked }))} />
                  recebendo candidaturas
                </label>
                <Button variant="ghost" size="sm" onClick={() => rodar(() => excluirTipoEspaco(eventoId, t.id))}>
                  Excluir
                </Button>
              </span>
            </div>
          ))}
          <div className="flex flex-wrap items-end gap-2">
            <div className="flex min-w-40 flex-1 flex-col gap-1.5">
              <Label htmlFor="te-nome">Nome</Label>
              <Input id="te-nome" value={novoTipo.nome} onChange={(e) => setNovoTipo({ ...novoTipo, nome: e.target.value })} />
            </div>
            <div className="flex w-28 flex-col gap-1.5">
              <Label htmlFor="te-preco">Preço (R$)</Label>
              <Input id="te-preco" type="number" min="0" step="0.01" inputMode="decimal" value={novoTipo.preco} onChange={(e) => setNovoTipo({ ...novoTipo, preco: e.target.value })} />
            </div>
            <div className="flex w-28 flex-col gap-1.5">
              <Label htmlFor="te-cred">Credenciais</Label>
              <Input id="te-cred" type="number" min="1" max="10" value={novoTipo.credenciais} onChange={(e) => setNovoTipo({ ...novoTipo, credenciais: e.target.value })} />
            </div>
            <Button
              onClick={() =>
                rodar(() =>
                  criarTipoEspaco(eventoId, {
                    nome: novoTipo.nome,
                    descricao: '',
                    preco_centavos: Math.round(Number(novoTipo.preco || 0) * 100),
                    credenciais: Number(novoTipo.credenciais),
                    ativo: true,
                  }),
                )
              }
            >
              Adicionar tipo
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">Preço 0 = gratuito (aprovado já confirma). Credenciais: quantas pessoas da equipe do expositor entram.</p>
        </section>

        {data.tipos.length > 0 && (
          <section aria-labelledby="t-mesas" className="flex flex-col gap-2">
            <h3 id="t-mesas" className="text-sm font-medium text-foreground">
              Mesas ({data.mesas.length})
            </h3>
            {setores.map((s) => (
              <div key={s}>
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Setor {s}</p>
                <ul className="mt-1 flex flex-wrap gap-1.5">
                  {data.mesas
                    .filter((m) => m.setor === s)
                    .map((m) => {
                      const x = ocupantes.get(m.id)
                      return (
                        <li
                          key={m.id}
                          className={`inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs ${x ? 'border-foreground bg-foreground text-background' : 'border-border text-foreground'}`}
                          title={`${tipoDe(m.tipo_espaco_id)?.nome ?? ''}${x ? ` · ${x.marca}` : ''}`}
                        >
                          {m.codigo}
                          {!x && (
                            <button type="button" aria-label={`Excluir mesa ${m.codigo}`} className="opacity-60 hover:opacity-100" onClick={() => rodar(() => excluirMesa(eventoId, m.id))}>
                              <X className="size-3" aria-hidden="true" />
                            </button>
                          )}
                        </li>
                      )
                    })}
                </ul>
              </div>
            ))}
            <div className="flex flex-wrap items-end gap-2">
              <div className="flex min-w-40 flex-1 flex-col gap-1.5">
                <Label htmlFor="lote-tipo">Tipo</Label>
                <select id="lote-tipo" className={select} value={tipoLote} onChange={(e) => setLote({ ...lote, tipo: e.target.value })}>
                  {data.tipos.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.nome}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex w-20 flex-col gap-1.5">
                <Label htmlFor="lote-setor">Setor</Label>
                <Input id="lote-setor" maxLength={10} value={lote.setor} onChange={(e) => setLote({ ...lote, setor: e.target.value })} />
              </div>
              <div className="flex w-20 flex-col gap-1.5">
                <Label htmlFor="lote-de">De</Label>
                <Input id="lote-de" type="number" min="1" value={lote.de} onChange={(e) => setLote({ ...lote, de: e.target.value })} />
              </div>
              <div className="flex w-20 flex-col gap-1.5">
                <Label htmlFor="lote-ate">Até</Label>
                <Input id="lote-ate" type="number" min="1" value={lote.ate} onChange={(e) => setLote({ ...lote, ate: e.target.value })} />
              </div>
              <Button
                variant="outline"
                onClick={() => rodar(() => criarMesas(eventoId, { tipo_espaco_id: tipoLote, setor: lote.setor, de: Number(lote.de), ate: Number(lote.ate) }))}
              >
                Criar mesas
              </Button>
            </div>
          </section>
        )}

        <section aria-labelledby="t-cand" className="flex flex-col gap-2">
          <h3 id="t-cand" className="text-sm font-medium text-foreground">
            Candidaturas ({pendentes.length} aguardando)
          </h3>
          {data.expositores.length === 0 && <p className="text-sm text-muted-foreground">Ninguém se candidatou ainda.</p>}
          {pendentes.map((x) => (
            <Candidatura key={x.id} x={x} painel={data} ocupantes={ocupantes} rodar={rodar} eventoId={eventoId} />
          ))}
          {demais.map((x) => (
            <Candidatura key={x.id} x={x} painel={data} ocupantes={ocupantes} rodar={rodar} eventoId={eventoId} />
          ))}
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

function Candidatura({
  x,
  painel,
  ocupantes,
  rodar,
  eventoId,
}: {
  x: ExpositorPainel
  painel: PainelExpositores
  ocupantes: Map<number, ExpositorPainel>
  rodar: (f: () => Promise<unknown>) => Promise<void>
  eventoId: number
}) {
  const [mesa, setMesa] = useState('')
  const tipo = painel.tipos.find((t) => t.id === x.tipo_espaco_id)
  const atual = painel.mesas.find((m) => m.id === x.espaco_id)
  const livres = painel.mesas.filter((m) => !ocupantes.has(m.id) || m.id === x.espaco_id)
  const ordenadas = [...livres.filter((m) => m.tipo_espaco_id === x.tipo_espaco_id), ...livres.filter((m) => m.tipo_espaco_id !== x.tipo_espaco_id)]
  const ativo = x.status === 'aprovado' || x.status === 'aguardando_pagamento'
  const podeAprovar = x.status !== 'desistiu'
  const escolhida = Number(mesa) || (x.espaco_id ?? ordenadas[0]?.id ?? 0)

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border px-3 py-2 text-sm">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="font-medium text-foreground">
          {x.marca} <span className="font-normal text-muted-foreground">· {tipo?.nome}</span>
        </p>
        <span className="rounded-full border border-border px-2.5 py-0.5 text-xs text-muted-foreground">
          {rotuloStatusExpositor[x.status]}
          {atual ? ` · ${atual.codigo}` : ''}
        </span>
      </div>
      <p className="text-xs text-muted-foreground">
        {x.responsavel_nome}
        {x.telefone && ` · ${x.telefone}`}
        {x.categoria && ` · ${x.categoria}`}
        {x.instagram && ` · @${x.instagram}`}
        {x.status === 'aguardando_pagamento' && x.prazo_pagamento_em && ` · paga até ${new Date(x.prazo_pagamento_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })}`}
        {x.status === 'rejeitado' && x.motivo_rejeicao && ` · ${x.motivo_rejeicao}`}
      </p>
      {x.descricao && <p className="text-foreground text-pretty">{x.descricao}</p>}
      {x.portfolio_url && (
        <a href={x.portfolio_url} target="_blank" rel="noopener noreferrer nofollow" className="text-xs underline underline-offset-4">
          Ver portfólio
        </a>
      )}
      {podeAprovar && (
        <div className="flex flex-wrap items-center gap-2">
          <label htmlFor={`mesa-${x.id}`} className="sr-only">
            Mesa para {x.marca}
          </label>
          <select id={`mesa-${x.id}`} className={select} value={escolhida} onChange={(e) => setMesa(e.target.value)}>
            {ordenadas.length === 0 && <option value={0}>Sem mesa livre</option>}
            {ordenadas.map((m) => (
              <option key={m.id} value={m.id}>
                {m.codigo} · {painel.tipos.find((t) => t.id === m.tipo_espaco_id)?.nome}
              </option>
            ))}
          </select>
          <Button size="sm" disabled={!escolhida || (ativo && escolhida === x.espaco_id)} onClick={() => rodar(() => aprovarExpositor(eventoId, x.id, escolhida))}>
            {x.status === 'pendente' ? 'Aprovar nesta mesa' : ativo ? 'Trocar de mesa' : 'Aprovar de novo'}
          </Button>
          {(x.status === 'pendente' || ativo) && (
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                const motivo = window.prompt(ativo ? 'Motivo da remoção (opcional). Se já pagou, recebe tudo de volta:' : 'Motivo da recusa (opcional):')
                if (motivo === null) return
                rodar(() => rejeitarExpositor(eventoId, x.id, motivo))
              }}
            >
              {ativo ? 'Remover' : 'Recusar'}
            </Button>
          )}
        </div>
      )}
    </div>
  )
}
