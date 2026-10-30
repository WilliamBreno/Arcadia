import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { Link, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { api } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'

type Painel = {
  nome: string
  evento_titulo: string
  evento_slug: string
  inicio_em: string | null
  codigo: string
  comissao_tipo: 'nenhuma' | 'percentual' | 'fixo'
  comissao_bp: number
  comissao_fixa_centavos: number
  ingressos: number
  comissao_centavos: number
  pago_centavos: number
  a_pagar_centavos: number
  liberado: boolean
  pagamentos: { valor_centavos: number; observacao: string; pago_em: string }[]
}

function regra(p: Painel) {
  if (p.comissao_tipo === 'percentual') return `${(p.comissao_bp / 100).toLocaleString('pt-BR')}% do preço de cada ingresso`
  if (p.comissao_tipo === 'fixo') return `${formatarCentavos(p.comissao_fixa_centavos)} por ingresso`
  return 'sem comissão definida'
}

// Painel do divulgador (item 9.2): link secreto, sem login e sem dados dos compradores.
export default function PainelDivulgador() {
  const { token = '' } = useParams()
  const { data, isLoading, error } = useQuery({
    queryKey: ['painel-divulgador', token],
    queryFn: () => api<Painel>(`/divulgador/${token}`, { semAuth: true }),
    retry: false,
  })

  useEffect(() => {
    document.title = 'Minhas vendas | Evve'
    const meta = document.createElement('meta')
    meta.name = 'robots'
    meta.content = 'noindex, nofollow'
    document.head.appendChild(meta)
    return () => meta.remove()
  }, [])

  if (isLoading) return <Carregando />
  if (error || !data) {
    return (
      <main className="mx-auto max-w-lg px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-foreground">Link inválido ou trocado</h1>
        <p className="mt-2 text-muted-foreground">Peça o link novo ao organizador do evento.</p>
      </main>
    )
  }
  const link = `${window.location.origin}/e/${data.evento_slug}?ref=${data.codigo}`

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <p className="text-sm font-medium uppercase tracking-wide text-muted-foreground">Suas vendas como divulgador</p>
      <h1 className="mt-1 text-3xl font-bold text-foreground text-balance">{data.evento_titulo}</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        {data.nome} · regra atual: {regra(data)} (vendas antigas mantêm a regra da época)
      </p>

      <dl className="mt-6 grid grid-cols-2 gap-3">
        {[
          ['Ingressos vendidos', String(data.ingressos)],
          ['Comissão', formatarCentavos(data.comissao_centavos)],
          ['Já recebido', formatarCentavos(data.pago_centavos)],
          ['A receber', formatarCentavos(data.a_pagar_centavos)],
        ].map(([r, v]) => (
          <div key={r} className="rounded-xl border border-border p-4">
            <dt className="text-xs text-muted-foreground">{r}</dt>
            <dd className="mt-1 text-xl font-semibold tabular-nums text-foreground">{v}</dd>
          </div>
        ))}
      </dl>
      <p className="mt-3 text-sm text-muted-foreground text-pretty">
        {data.liberado
          ? 'O evento já terminou: o organizador paga a comissão diretamente a você.'
          : 'A comissão fica disponível depois do fim do evento. Ingresso cancelado ou reembolsado sai da conta.'}
      </p>

      <section aria-labelledby="t-link" className="mt-6 rounded-xl border border-border p-4">
        <h2 id="t-link" className="text-sm font-medium text-foreground">
          Seu link para divulgar
        </h2>
        <p className="mt-1 break-all font-mono text-sm text-foreground">{link}</p>
        <Link to={`/e/${data.evento_slug}`} className="mt-2 inline-block text-sm underline underline-offset-4">
          Ver a página do evento
        </Link>
      </section>

      {data.pagamentos.length > 0 && (
        <section aria-labelledby="t-pagamentos" className="mt-6">
          <h2 id="t-pagamentos" className="text-sm font-medium text-foreground">
            Pagamentos registrados pelo organizador
          </h2>
          <ul className="mt-2 flex flex-col divide-y divide-border rounded-xl border border-border">
            {data.pagamentos.map((p, i) => (
              <li key={i} className="flex items-baseline justify-between gap-3 px-4 py-2 text-sm">
                <span className="text-muted-foreground">
                  {new Date(p.pago_em).toLocaleDateString('pt-BR')}
                  {p.observacao && ` · ${p.observacao}`}
                </span>
                <span className="font-medium tabular-nums text-foreground">{formatarCentavos(p.valor_centavos)}</span>
              </li>
            ))}
          </ul>
        </section>
      )}
    </main>
  )
}
