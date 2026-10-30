import { useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Minus, Plus } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'

import { VoltarBotao } from '@/components/voltar-link'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'

type Tipo = { id: number; nome: string; preco_centavos: number; total_pix_centavos: number; disponivel: number }
type Venda = { id: number; status: string; metodo: 'pix' | 'dinheiro'; total_centavos: number; itens: { codigo: string; status: string }[] }
type Resumo = {
  organizador: boolean
  linhas: { vendedor: string; ingressos: number; pix_centavos: number; dinheiro_centavos: number; taxa_dinheiro_centavos: number }[]
}

// Venda na porta (item 9.4): organizador e equipe de check-in vendem pelo
// celular. Dinheiro: o ingresso sai na hora. Pix: o QR aparece na tela e a
// confirmação chega sozinha. O ingresso já nasce usado (a pessoa entrou).
export default function VendaPorta() {
  const { eventoId = '' } = useParams()
  const base = `/checkin/eventos/${eventoId}/porta`
  const queryClient = useQueryClient()
  const { data: tipos, error } = useQuery({ queryKey: ['porta-tipos', eventoId], queryFn: () => api<Tipo[]>(`${base}/tipos`), retry: false })
  const { data: resumo } = useQuery({ queryKey: ['porta-resumo', eventoId], queryFn: () => api<Resumo>(`${base}/resumo`) })
  const [qtd, setQtd] = useState<Record<number, number>>({})
  const [nome, setNome] = useState('')
  const [email, setEmail] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const [ocupado, setOcupado] = useState(false)
  const [venda, setVenda] = useState<Venda | null>(null)
  const [pix, setPix] = useState<string | null>(null)

  useEffect(() => {
    document.title = 'Venda na porta | Evve'
  }, [])

  // Pix: confere a cada 3 s até pagar.
  useEffect(() => {
    if (!venda || venda.metodo !== 'pix' || venda.status !== 'aguardando_pagamento') return
    const t = window.setInterval(async () => {
      try {
        const v = await api<Venda>(`${base}/vendas/${venda.id}`)
        if (v.status !== 'aguardando_pagamento') setVenda(v)
      } catch {
        // tenta de novo no próximo ciclo
      }
    }, 3000)
    return () => window.clearInterval(t)
  }, [venda, base])

  const recarregar = () =>
    Promise.all([queryClient.invalidateQueries({ queryKey: ['porta-tipos', eventoId] }), queryClient.invalidateQueries({ queryKey: ['porta-resumo', eventoId] })])

  if (error instanceof ApiError && error.status === 403) {
    return (
      <main className="mx-auto max-w-md px-4 py-12 text-center">
        <p className="text-muted-foreground">Só o organizador e a equipe de check-in deste evento vendem na porta.</p>
      </main>
    )
  }

  const itens = (tipos ?? []).filter((t) => (qtd[t.id] ?? 0) > 0)
  const totalDinheiro = itens.reduce((s, t) => s + t.preco_centavos * (qtd[t.id] ?? 0), 0)
  const totalPix = itens.reduce((s, t) => s + t.total_pix_centavos * (qtd[t.id] ?? 0), 0)
  const unidades = itens.reduce((s, t) => s + (qtd[t.id] ?? 0), 0)

  const vender = async (metodo: 'pix' | 'dinheiro') => {
    if (metodo === 'dinheiro' && !confirm(`Recebeu ${formatarCentavos(totalDinheiro)} em dinheiro?`)) return
    setErro(null)
    setOcupado(true)
    try {
      const v = await api<Venda>(`${base}/vendas`, {
        method: 'POST',
        body: { itens: itens.map((t) => ({ tipo_ingresso_id: t.id, quantidade: qtd[t.id] })), metodo, nome, email },
      })
      setVenda(v)
      if (metodo === 'pix') {
        const p = await api<{ copia_cola: string }>(`${base}/vendas/${v.id}/pix`, { method: 'POST' })
        setPix(p.copia_cola)
      }
      recarregar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível vender agora.')
    } finally {
      setOcupado(false)
    }
  }
  const novaVenda = () => {
    setVenda(null)
    setPix(null)
    setQtd({})
    setNome('')
    setEmail('')
    setErro(null)
    recarregar()
  }

  if (venda && venda.status !== 'aguardando_pagamento') {
    const ok = venda.status === 'pago'
    return (
      <main className={`flex min-h-svh flex-col items-center justify-center gap-4 px-6 text-center ${ok ? 'bg-green-600 text-white' : 'bg-background'}`}>
        {ok && <CheckCircle2 className="size-20" aria-hidden="true" />}
        <h1 className="text-3xl font-bold">{ok ? 'Pago! Pode entrar.' : 'O Pix não foi concluído.'}</h1>
        {ok && (
          <p className="text-lg">
            {venda.itens.length} ingresso(s) · {formatarCentavos(venda.total_centavos)} {venda.metodo === 'dinheiro' ? 'em dinheiro' : 'no Pix'}
          </p>
        )}
        <Button size="lg" variant={ok ? 'secondary' : 'default'} onClick={novaVenda}>
          Nova venda
        </Button>
      </main>
    )
  }

  if (venda && pix) {
    return (
      <main className="mx-auto flex max-w-md flex-col items-center gap-4 px-4 py-8 text-center">
        <h1 className="text-2xl font-semibold text-foreground">Pague com Pix</h1>
        <p className="text-3xl font-bold tabular-nums text-foreground">{formatarCentavos(venda.total_centavos)}</p>
        <div className="rounded-xl bg-white p-4">
          <QRCodeSVG value={pix} size={260} level="M" aria-label="QR do Pix" />
        </div>
        <p className="text-sm text-muted-foreground" aria-live="polite">
          Esperando o pagamento… a tela muda sozinha quando o Pix cair.
        </p>
        <Button variant="outline" onClick={() => navigator.clipboard?.writeText(pix)}>
          Copiar código Pix
        </Button>
        <Button variant="ghost" onClick={novaVenda}>
          Cancelar e voltar
        </Button>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-md px-4 py-8">
      <VoltarBotao fallback={`/checkin/${eventoId}`} />
      <h1 className="text-2xl font-semibold text-foreground">Venda na porta</h1>
      <p className="mt-1 text-sm text-muted-foreground">O ingresso já sai validado: a pessoa pode entrar.</p>

      <ul className="mt-5 flex flex-col gap-2">
        {tipos?.map((t) => {
          const n = qtd[t.id] ?? 0
          return (
            <li key={t.id} className="flex items-center justify-between gap-3 rounded-xl border border-border p-3">
              <div className="min-w-0">
                <p className="font-medium text-foreground">{t.nome}</p>
                <p className="text-xs text-muted-foreground">
                  {formatarCentavos(t.preco_centavos)} em dinheiro · {formatarCentavos(t.total_pix_centavos)} no Pix · restam {t.disponivel}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button size="icon" variant="outline" aria-label={`Menos ${t.nome}`} disabled={n === 0} onClick={() => setQtd({ ...qtd, [t.id]: n - 1 })}>
                  <Minus className="size-4" aria-hidden="true" />
                </Button>
                <span className="w-6 text-center text-lg font-semibold tabular-nums" aria-live="polite">
                  {n}
                </span>
                <Button size="icon" variant="outline" aria-label={`Mais ${t.nome}`} disabled={n >= t.disponivel} onClick={() => setQtd({ ...qtd, [t.id]: n + 1 })}>
                  <Plus className="size-4" aria-hidden="true" />
                </Button>
              </div>
            </li>
          )
        })}
        {tipos?.length === 0 && <p className="text-sm text-muted-foreground">Nenhum ingresso ativo neste evento.</p>}
      </ul>

      <div className="mt-4 grid grid-cols-1 gap-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="porta-nome">Nome (opcional)</Label>
          <Input id="porta-nome" value={nome} onChange={(e) => setNome(e.target.value)} autoComplete="off" />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="porta-email">E-mail para receber o ingresso (opcional)</Label>
          <Input id="porta-email" type="email" inputMode="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" />
        </div>
      </div>

      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}

      <div className="mt-5 grid grid-cols-2 gap-3">
        <Button size="lg" className="h-auto flex-col py-3" disabled={!unidades || ocupado} onClick={() => vender('dinheiro')}>
          <span>Dinheiro</span>
          <span className="text-base font-bold tabular-nums">{formatarCentavos(totalDinheiro)}</span>
        </Button>
        <Button size="lg" variant="outline" className="h-auto flex-col py-3" disabled={!unidades || ocupado} onClick={() => vender('pix')}>
          <span>Pix</span>
          <span className="text-base font-bold tabular-nums">{formatarCentavos(totalPix)}</span>
        </Button>
      </div>
      <p className="mt-2 text-xs text-muted-foreground text-pretty">
        No Pix a taxa da Evve vai junto, como na compra online. No dinheiro a pessoa paga só o preço e a taxa sai do repasse do organizador.
      </p>

      {resumo && resumo.linhas.length > 0 && (
        <section aria-labelledby="t-resumo" className="mt-8">
          <h2 id="t-resumo" className="text-lg font-semibold text-foreground">
            {resumo.organizador ? 'Vendas na porta (equipe)' : 'Minhas vendas'}
          </h2>
          <ul className="mt-2 flex flex-col divide-y divide-border rounded-xl border border-border">
            {resumo.linhas.map((l) => (
              <li key={l.vendedor} className="px-4 py-2 text-sm">
                <p className="font-medium text-foreground">{l.vendedor}</p>
                <p className="text-xs text-muted-foreground">
                  {l.ingressos} ingresso(s) · Pix {formatarCentavos(l.pix_centavos)} · dinheiro em mãos {formatarCentavos(l.dinheiro_centavos)}
                </p>
              </li>
            ))}
          </ul>
        </section>
      )}
    </main>
  )
}
