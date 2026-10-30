import { useQuery } from '@tanstack/react-query'
import { Check, Sparkles } from 'lucide-react'
import { useState } from 'react'

import { Carregando } from '@/components/carregando'
import { PagarCobranca } from '@/components/pagar-cobranca'
import { Button } from '@/components/ui/button'
import { VoltarLink } from '@/components/voltar-link'
import { api, ApiError, baixarArquivo } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import type { Cobranca, SituacaoPro } from '@/lib/plataforma'

const beneficios = [
  'E-mail do ingresso com o seu nome no lugar da marca Evve (assunto e cabeçalho)',
  'E-mail para quem segue você com cota 10 vezes maior: 50 mil por mês',
  'Planilha com todos os eventos (vendas, cortesias, receita, check-ins e cancelados)',
  'Selo "Pro" no seu perfil e na página dos eventos',
  '1 dia de destaque na vitrine incluído a cada 30 dias',
]

// Item 10.4: Plano Pro do organizador (30 dias ou 12 meses, sem renovação
// automática). A taxa por ingresso não muda; nada do grátis vira pago.
export default function Pro() {
  const { data, isLoading, refetch } = useQuery({ queryKey: ['org-pro'], queryFn: () => api<SituacaoPro>('/org/pro') })
  const [cobranca, setCobranca] = useState<Cobranca | null>(null)
  const [erro, setErro] = useState<string | null>(null)

  if (isLoading || !data) return <Carregando />

  const comprar = async (anual: boolean) => {
    setErro(null)
    try {
      setCobranca(await api<Cobranca>('/org/pro/cobrancas', { method: 'POST', body: { anual } }))
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível criar a cobrança.')
    }
  }
  const p = data.precos
  const economia = p.pro_mensal_centavos * 12 - p.pro_anual_centavos

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar ao painel</VoltarLink>
      <p className="flex items-center gap-2 text-sm font-medium text-primary">
        <Sparkles className="size-4" aria-hidden="true" />
        Evve Pro
      </p>
      <h1 className="mt-1 text-3xl font-semibold text-foreground">Ferramentas a mais para quem produz sempre</h1>
      <p className="mt-2 text-muted-foreground">A taxa por ingresso continua a mesma, e tudo o que é grátis continua grátis.</p>

      {data.pro && data.ate && (
        <p role="status" className="mt-6 rounded-xl border border-green-600 p-4 text-foreground">
          Você é Pro até <strong>{new Date(data.ate).toLocaleDateString('pt-BR')}</strong>. Renovar agora soma o novo período no fim deste.
        </p>
      )}

      <ul className="mt-8 flex flex-col gap-3">
        {beneficios.map((b) => (
          <li key={b} className="flex items-start gap-3 text-foreground">
            <Check className="mt-0.5 size-5 shrink-0 text-primary" aria-hidden="true" />
            {b}
          </li>
        ))}
      </ul>

      {cobranca ? (
        <div className="mt-8">
          <PagarCobranca cobranca={cobranca} aoPagar={() => refetch()} aoFechar={() => setCobranca(null)} />
        </div>
      ) : (
        <div className="mt-8 grid gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-3 rounded-xl border border-border p-4">
            <p className="text-sm text-muted-foreground">30 dias</p>
            <p className="text-3xl font-semibold tabular-nums text-foreground">{formatarCentavos(p.pro_mensal_centavos)}</p>
            <Button onClick={() => comprar(false)}>{data.pro ? 'Renovar 30 dias' : 'Assinar 30 dias'}</Button>
          </div>
          <div className="flex flex-col gap-3 rounded-xl border-2 border-primary p-4">
            <p className="text-sm text-muted-foreground">12 meses{economia > 0 ? ` · economia de ${formatarCentavos(economia)}` : ''}</p>
            <p className="text-3xl font-semibold tabular-nums text-foreground">{formatarCentavos(p.pro_anual_centavos)}</p>
            <Button onClick={() => comprar(true)}>{data.pro ? 'Renovar 12 meses' : 'Assinar 12 meses'}</Button>
          </div>
        </div>
      )}
      <p className="mt-3 text-xs text-muted-foreground">Pix ou cartão, sem renovação automática: avisamos por e-mail 5 dias antes de vencer.</p>
      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}

      {data.pro && (
        <section aria-labelledby="titulo-planilha" className="mt-10 rounded-xl border border-border p-4">
          <h2 id="titulo-planilha" className="font-semibold text-foreground">
            Planilha de todos os eventos
          </h2>
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            onClick={() => baixarArquivo('/org/relatorios.csv', 'relatorio-eventos.csv').catch((e) => setErro(e instanceof ApiError ? e.message : 'Erro ao baixar'))}
          >
            Baixar planilha (CSV)
          </Button>
        </section>
      )}
    </main>
  )
}
