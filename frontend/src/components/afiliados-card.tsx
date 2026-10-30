import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type TipoComissao = 'nenhuma' | 'percentual' | 'fixo'

type Afiliado = {
  id: number
  nome: string
  codigo: string
  ativo: boolean
  pedidos: number
  itens: number
  receita_centavos: number
  // Item 9.2
  comissao_tipo: TipoComissao
  comissao_bp: number
  comissao_fixa_centavos: number
  painel_ativo: boolean
  comissao_centavos: number
  pago_centavos: number
  a_pagar_centavos: number
  liberado: boolean
}

type Regra = { tipo: TipoComissao; valor: string } // valor: % ou reais, conforme o tipo

const select = 'h-8 rounded-lg border border-border bg-background px-2.5 text-sm'

function paraCorpo(r: Regra) {
  const n = Number(String(r.valor).replace(',', '.')) || 0
  return {
    comissao_tipo: r.tipo,
    comissao_bp: r.tipo === 'percentual' ? Math.round(n * 100) : 0,
    comissao_fixa_centavos: r.tipo === 'fixo' ? Math.round(n * 100) : 0,
  }
}

function descricaoRegra(tipo: TipoComissao, bp: number, fixa: number) {
  if (tipo === 'percentual') return `${(bp / 100).toLocaleString('pt-BR')}% do preço`
  if (tipo === 'fixo') return `${formatarCentavos(fixa)} por ingresso`
  return 'sem comissão'
}

function CamposRegra({ id, regra, mudar }: { id: string; regra: Regra; mudar: (r: Regra) => void }) {
  return (
    <div className="flex flex-wrap items-end gap-2">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor={`${id}-tipo`}>Comissão</Label>
        <select id={`${id}-tipo`} className={select} value={regra.tipo} onChange={(e) => mudar({ tipo: e.target.value as TipoComissao, valor: '' })}>
          <option value="nenhuma">Sem comissão</option>
          <option value="percentual">% do preço</option>
          <option value="fixo">Valor por ingresso</option>
        </select>
      </div>
      {regra.tipo !== 'nenhuma' && (
        <div className="flex w-28 flex-col gap-1.5">
          <Label htmlFor={`${id}-valor`}>{regra.tipo === 'percentual' ? 'Percentual (%)' : 'Valor (R$)'}</Label>
          <Input
            id={`${id}-valor`}
            type="number"
            inputMode="decimal"
            min="0"
            max={regra.tipo === 'percentual' ? 50 : 1000}
            step="0.01"
            value={regra.valor}
            onChange={(e) => mudar({ ...regra, valor: e.target.value })}
          />
        </div>
      )}
    </div>
  )
}

export function AfiliadosCard({ eventoId, slug }: { eventoId: number; slug: string }) {
  const queryClient = useQueryClient()
  const [nome, setNome] = useState('')
  const [regra, setRegra] = useState<Regra>({ tipo: 'nenhuma', valor: '' })
  const [erro, setErro] = useState<string | null>(null)
  const { data } = useQuery({ queryKey: ['org-afiliados', eventoId], queryFn: () => api<Afiliado[]>(`/org/eventos/${eventoId}/afiliados`) })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-afiliados', eventoId] })

  const rodar = async (f: () => Promise<unknown>) => {
    setErro(null)
    try {
      await f()
      await invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir agora.')
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Divulgadores (links de afiliado)</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-xs text-muted-foreground text-pretty">
          Cada divulgador ganha um link; as vendas pagas por ele aparecem aqui. Se quiser, defina uma comissão (% do preço ou valor por ingresso): a
          Evve calcula, você paga o divulgador por fora e registra aqui. Ingresso cancelado sai da conta, e a comissão só fica a pagar depois do fim do
          evento. Mudar a regra vale para as próximas vendas.
        </p>
        {data?.map((a) => (
          <LinhaAfiliado key={a.id} a={a} eventoId={eventoId} slug={slug} rodar={rodar} />
        ))}
        <div className="flex flex-col gap-2 border-t border-border pt-3">
          <div className="flex flex-wrap items-end gap-2">
            <div className="flex min-w-48 flex-1 flex-col gap-1.5">
              <Label htmlFor="novo-divulgador">Nome do divulgador</Label>
              <Input id="novo-divulgador" value={nome} onChange={(e) => setNome(e.target.value)} />
            </div>
            <CamposRegra id="nova-regra" regra={regra} mudar={setRegra} />
            <Button
              disabled={!nome}
              onClick={() =>
                rodar(async () => {
                  await api(`/org/eventos/${eventoId}/afiliados`, { method: 'POST', body: { nome, ...paraCorpo(regra) } })
                  setNome('')
                  setRegra({ tipo: 'nenhuma', valor: '' })
                })
              }
            >
              Criar link
            </Button>
          </div>
        </div>
        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
      </CardContent>
    </Card>
  )
}

function LinhaAfiliado({
  a,
  eventoId,
  slug,
  rodar,
}: {
  a: Afiliado
  eventoId: number
  slug: string
  rodar: (f: () => Promise<unknown>) => Promise<void>
}) {
  const [editando, setEditando] = useState(false)
  const [regra, setRegra] = useState<Regra>({
    tipo: a.comissao_tipo,
    valor: a.comissao_tipo === 'percentual' ? String(a.comissao_bp / 100) : a.comissao_tipo === 'fixo' ? String(a.comissao_fixa_centavos / 100) : '',
  })
  const [painel, setPainel] = useState<string | null>(null)
  const base = `/org/eventos/${eventoId}/afiliados/${a.id}`

  const registrarPagamento = () => {
    const v = window.prompt(`Quanto você pagou a ${a.nome}? (a pagar: ${formatarCentavos(a.a_pagar_centavos)})`, (a.a_pagar_centavos / 100).toFixed(2))
    if (v === null) return
    const centavos = Math.round(Number(v.replace(',', '.')) * 100)
    if (!centavos) return
    const obs = window.prompt('Observação (opcional, ex.: "Pix em 30/10"):') ?? ''
    rodar(() => api(`${base}/pagamentos`, { method: 'POST', body: { valor_centavos: centavos, observacao: obs } }))
  }
  const gerarPainel = () =>
    rodar(async () => {
      const r = await api<{ token: string }>(`${base}/painel`, { method: 'POST' })
      setPainel(`${window.location.origin}/divulgador/${r.token}`)
    })

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border px-3 py-2 text-sm">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="font-medium text-foreground">
            {a.nome} {!a.ativo && <span className="text-xs text-muted-foreground">(desativado)</span>}
          </p>
          <p className="truncate font-mono text-xs text-muted-foreground">{`${window.location.origin}/e/${slug}?ref=${a.codigo}`}</p>
          <p className="text-xs text-muted-foreground">
            {a.pedidos} pedido(s) · {a.itens} ingresso(s) · {formatarCentavos(a.receita_centavos)} · {descricaoRegra(a.comissao_tipo, a.comissao_bp, a.comissao_fixa_centavos)}
          </p>
          {(a.comissao_centavos > 0 || a.pago_centavos > 0) && (
            <p className="text-xs text-foreground">
              Comissão {formatarCentavos(a.comissao_centavos)} · paga {formatarCentavos(a.pago_centavos)} ·{' '}
              <strong>
                a pagar {formatarCentavos(a.a_pagar_centavos)}
                {!a.liberado && ' (depois do fim do evento)'}
              </strong>
            </p>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {a.liberado && a.a_pagar_centavos > 0 && (
            <Button size="sm" onClick={registrarPagamento}>
              Registrar pagamento
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={() => setEditando(!editando)}>
            Comissão
          </Button>
          <Button size="sm" variant="outline" onClick={gerarPainel}>
            {a.painel_ativo ? 'Novo link do painel' : 'Link do painel'}
          </Button>
          {a.ativo && (
            <Button variant="ghost" size="sm" onClick={() => rodar(() => api(base, { method: 'DELETE' }))}>
              Desativar
            </Button>
          )}
        </div>
      </div>
      {editando && (
        <div className="flex flex-wrap items-end gap-2 border-t border-border pt-2">
          <CamposRegra id={`regra-${a.id}`} regra={regra} mudar={setRegra} />
          <Button
            size="sm"
            onClick={() =>
              rodar(async () => {
                await api(`${base}/comissao`, { method: 'PUT', body: paraCorpo(regra) })
                setEditando(false)
              })
            }
          >
            Salvar (vale para as próximas vendas)
          </Button>
        </div>
      )}
      {painel && (
        <div className="flex flex-col gap-1 border-t border-border pt-2">
          <p className="text-xs text-muted-foreground">
            Mande este link só para {a.nome}: ele mostra as vendas e a comissão dele, sem dados dos compradores. Gerar outro desliga este.
          </p>
          <div className="flex gap-2">
            <Input readOnly value={painel} aria-label={`Link do painel de ${a.nome}`} onFocus={(e) => e.currentTarget.select()} />
            <Button size="sm" variant="outline" onClick={() => navigator.clipboard?.writeText(painel)}>
              Copiar
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
