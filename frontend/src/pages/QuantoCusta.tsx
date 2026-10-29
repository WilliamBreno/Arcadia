import { useQuery } from '@tanstack/react-query'
import { ShieldCheck } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { Surgir } from '@/components/surgir'

import { api } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { garantiaDoPreco, taxaPorFaixa, type RegraTaxas } from '@/lib/taxas'

// Página "Quanto custa" (item 6.3). A página é o próprio ingresso: o
// simulador imprime, linha por linha, preço, taxa Evve e total, com a taxa
// típica do mercado ao lado. As taxas vêm da API (GET /taxas), nunca fixas
// aqui. Concorrentes não são citados pelo nome.

type TabelaTaxas = Omit<RegraTaxas, 'limite_bp' | 'taxa_paga_por'>

const SUGESTOES = [2000, 5000, 15000, 40000]

// Taxa típica do mercado brasileiro: 10% do ingresso, mínimo R$ 3,99.
function taxaMercado(preco: number) {
  if (preco <= 0) return 0
  return Math.max(Math.round(preco * 0.1), 399)
}

function paraCentavos(texto: string): number {
  const limpo = texto.replace(/[^\d,.]/g, '').replace(/\./g, '').replace(',', '.')
  const n = Number(limpo)
  return Number.isFinite(n) ? Math.min(Math.round(n * 100), 10_000_000) : 0
}

function reaisSemSimbolo(c: number) {
  return (c / 100).toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

const regraDe = (t: TabelaTaxas): RegraTaxas => ({ ...t, limite_bp: null, taxa_paga_por: 'comprador' })

// Recorte do picote: círculos na cor do fundo "mordendo" as bordas.
function Recorte({ className, style }: { className: string; style?: React.CSSProperties }) {
  return <span aria-hidden="true" className={`absolute size-7 rounded-full bg-muted ${className}`} style={style} />
}

function Ingresso({ tabela }: { tabela: TabelaTaxas }) {
  const navigate = useNavigate()
  const [texto, setTexto] = useState('50,00')
  const [garantia, setGarantia] = useState(false)
  const preco = paraCentavos(texto)
  const regra = regraDe(tabela)
  const taxa = taxaPorFaixa(regra, preco)
  const valorGarantia = garantia ? garantiaDoPreco(regra, preco) : 0
  const total = preco + taxa + valorGarantia
  const mercado = taxaMercado(preco)
  const economia = mercado - taxa

  return (
    <div className="relative mx-auto flex w-full max-w-5xl flex-col bg-background md:flex-row" style={{ borderRadius: 12 }}>
      {/* parte principal */}
      <div className="flex flex-col gap-6 p-6 md:p-10" style={{ flex: '7 1 0%' }}>
        <div>
          <h1 className="text-4xl font-bold leading-none text-foreground text-balance sm:text-5xl lg:text-6xl" style={{ letterSpacing: '-0.03em' }}>
            Taxa fixa. Sem porcentagem.
          </h1>
          <p className="mt-4 max-w-md text-base text-foreground text-pretty">
            O comprador paga a partir de {formatarCentavos(tabela.faixas[0]?.taxa_centavos ?? 0)} por ingresso. Evento gratuito não paga nada.
            Digite o preço do seu ingresso e veja a conta.
          </p>
        </div>

        <div>
          <label htmlFor="preco" className="text-sm font-medium text-foreground">
            Preço do ingresso
          </label>
          <div className="mt-1.5 flex items-center gap-2">
            <span className="text-2xl font-semibold text-foreground">R$</span>
            <input
              id="preco"
              inputMode="decimal"
              autoComplete="off"
              value={texto}
              onChange={(e) => setTexto(e.target.value)}
              onBlur={() => setTexto(reaisSemSimbolo(preco))}
              className="h-12 w-40 rounded-lg bg-muted px-3 text-2xl font-semibold tabular-nums text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
          </div>
          <div className="mt-3 flex flex-wrap gap-2">
            {SUGESTOES.map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => setTexto(reaisSemSimbolo(s))}
                aria-pressed={preco === s}
                className={`rounded-full px-3 py-1 text-sm transition-colors ${preco === s ? 'bg-primary text-primary-foreground' : 'bg-muted text-foreground hover:bg-foreground/10'}`}
              >
                {formatarCentavos(s)}
              </button>
            ))}
          </div>
          <label className="mt-4 flex items-center gap-2 text-sm text-foreground">
            <input type="checkbox" checked={garantia} onChange={(e) => setGarantia(e.target.checked)} />
            Comprador quer a garantia de vaga (opcional)
          </label>
        </div>

        <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-base text-foreground" aria-live="polite">
          <div>
            <dt className="text-sm text-muted-foreground">Ingresso</dt>
            <dd className="font-semibold tabular-nums">{formatarCentavos(preco)}</dd>
          </div>
          <div>
            <dt className="text-sm text-muted-foreground">Taxa Evve</dt>
            <dd className="font-semibold tabular-nums">{formatarCentavos(taxa)}</dd>
          </div>
          {garantia && (
            <div>
              <dt className="text-sm text-muted-foreground">Garantia de vaga</dt>
              <dd className="font-semibold tabular-nums">{formatarCentavos(valorGarantia)}</dd>
            </div>
          )}
          <div>
            <dt className="text-sm text-muted-foreground">Taxa típica do mercado (10%, mín. R$ 3,99)</dt>
            <dd className="font-semibold tabular-nums">{formatarCentavos(mercado)}</dd>
          </div>
        </dl>
      </div>

      {/* picote: coluna de pontos na cor do fundo (vertical no computador, horizontal no celular) */}
      <div aria-hidden="true" className="relative hidden md:block" style={{ width: 0, borderLeft: '6px dotted var(--muted)' }}>
        <Recorte className="-top-3.5 -left-4" />
        <Recorte className="-bottom-3.5 -left-4" />
      </div>
      <div aria-hidden="true" className="relative md:hidden" style={{ height: 0, borderTop: '6px dotted var(--muted)' }}>
        <Recorte className="-left-3.5 -top-4" />
        <Recorte className="-right-3.5 -top-4" />
      </div>

      {/* canhoto */}
      <div className="flex flex-col justify-between gap-6 p-6 md:p-8" style={{ flex: '3 1 0%' }}>
        <div>
          <p key={total} className="animar-valor text-5xl font-bold leading-none tabular-nums text-foreground" style={{ letterSpacing: '-0.03em' }}>
            {formatarCentavos(total)}
          </p>
          <p className="mt-2 text-base text-foreground">é o que o comprador paga.</p>
          <p className="mt-3 text-base text-foreground text-pretty">
            {preco === 0
              ? 'Gratuito: sem taxa e sem passar pelo pagamento.'
              : economia > 0
                ? `${formatarCentavos(economia)} a menos de taxa que o típico do mercado.`
                : 'A mesma taxa fixa, em qualquer cidade.'}
          </p>
        </div>
        <button
          type="button"
          onClick={() => navigate('/organizador/eventos/novo')}
          className="h-14 w-full rounded-lg bg-primary text-base font-semibold text-primary-foreground transition-transform active:scale-95"
        >
          Criar meu evento
        </button>
      </div>
    </div>
  )
}

function Tabela({ tabela }: { tabela: TabelaTaxas }) {
  const faixas = tabela.faixas.map((f, i) => {
    const anterior = i > 0 ? tabela.faixas[i - 1].ate_centavos : 0
    const nome =
      f.ate_centavos < 0
        ? `Acima de ${formatarCentavos(anterior)}`
        : i === 0
          ? `Até ${formatarCentavos(f.ate_centavos)}`
          : `De ${formatarCentavos(anterior + 1)} a ${formatarCentavos(f.ate_centavos)}`
    return { nome, valor: formatarCentavos(f.taxa_centavos) }
  })
  const grupos: { titulo: string; itens: { nome: string; valor: string; nota?: string }[] }[] = [
    {
      titulo: 'Taxa por ingresso pago',
      itens: [{ nome: 'Ingresso gratuito', valor: 'R$ 0,00', nota: 'Inscrição grátis não paga taxa nem passa pelo pagamento.' }, ...faixas],
    },
    {
      titulo: 'Garantia de vaga (opcional, o comprador escolhe)',
      itens: [
        {
          nome: `${tabela.garantia_percentual_bp / 100}% do ingresso`,
          valor: `${formatarCentavos(tabela.garantia_min_centavos)} a ${formatarCentavos(tabela.garantia_max_centavos)}`,
          nota: 'Cancela até o início do evento e recebe tudo de volta.',
        },
      ],
    },
    {
      titulo: 'Não cobramos',
      itens: [
        { nome: 'Mensalidade ou adesão', valor: 'R$ 0,00' },
        { nome: 'Criar e publicar eventos', valor: 'R$ 0,00' },
        { nome: 'Cortesias', valor: 'R$ 0,00' },
        { nome: 'Check-in e app da portaria', valor: 'R$ 0,00' },
      ],
    },
    {
      titulo: 'Pago à parte pelo organizador',
      itens: [
        {
          nome: 'Processamento do pagamento',
          valor: 'no repasse',
          nota: 'Cobrado pelo Mercado Pago e descontado do repasse; no Pix é bem menor que no cartão.',
        },
      ],
    },
  ]

  return (
    <section aria-labelledby="titulo-tabela" className="mx-auto w-full max-w-xl bg-background px-5 py-10 sm:px-12" style={{ borderRadius: 12 }}>
      <h2 id="titulo-tabela" className="text-center text-4xl font-bold text-foreground" style={{ letterSpacing: '-0.02em' }}>
        Tabela de taxas
      </h2>
      <div className="mt-10 flex flex-col gap-11">
        {grupos.map((g) => (
          <Surgir key={g.titulo}>
            <h3 className="text-lg font-bold text-foreground">{g.titulo}</h3>
            <ul className="mt-4 flex flex-col gap-4">
              {g.itens.map((it) => (
                <li key={it.nome}>
                  <div className="flex items-baseline gap-2 text-base text-foreground">
                    <span className="shrink-0">{it.nome}</span>
                    <span aria-hidden="true" className="flex-1" style={{ borderBottom: '4px dotted var(--border)', transform: 'translateY(-4px)' }} />
                    <span className="shrink-0 font-semibold tabular-nums">{it.valor}</span>
                  </div>
                  {it.nota && <p className="mt-1 max-w-xs text-sm text-muted-foreground">{it.nota}</p>}
                </li>
              ))}
            </ul>
          </Surgir>
        ))}
      </div>
      <p className="mt-10 text-sm text-muted-foreground">
        Em estados com limite em lei para a taxa (como Alagoas, 10% do ingresso), o valor é ajustado sozinho para nunca passar do limite.
      </p>
    </section>
  )
}

const ABAS = {
  comprador: {
    rotulo: 'Para quem compra',
    titulo: 'Seu ingresso protegido',
    texto:
      'Se o evento for cancelado, você recebe tudo de volta automaticamente, inclusive a taxa. Com a garantia de vaga, pode desistir até o início do evento. Sem ela, vale o arrependimento de 7 dias da compra, até 48 horas antes do evento. O ingresso sai no seu nome, com QR Code único.',
    acao: { rotulo: 'Ver eventos', para: '/' },
  },
  organizador: {
    rotulo: 'Para quem organiza',
    titulo: 'Seu dinheiro chega',
    texto:
      'O repasse é liberado por Pix a partir de 3 dias úteis depois do evento, com extrato de cada venda e da taxa do pagamento. Organizadores verificados ganham o selo na página do evento. Contestações de cartão são acompanhadas pela plataforma, e nada é descontado de você automaticamente.',
    acao: { rotulo: 'Criar meu evento', para: '/organizador/eventos/novo' },
  },
} as const

function Garante() {
  const [aba, setAba] = useState<keyof typeof ABAS>('comprador')
  const atual = ABAS[aba]
  return (
    <section id="garante" aria-labelledby="titulo-garante" className="mx-auto grid w-full max-w-5xl grid-cols-1 gap-8 md:grid-cols-2 md:gap-12">
      <div data-ui className="flex aspect-square w-full flex-col items-center justify-center gap-4 bg-primary p-8 text-primary-foreground" style={{ borderRadius: 12 }}>
        <ShieldCheck className="size-24" strokeWidth={1.5} aria-hidden="true" />
        <p className="text-4xl font-bold" style={{ letterSpacing: '-0.03em' }}>
          Evve Garante
        </p>
        <p className="max-w-xs text-center text-base opacity-90">Aparece na página de cada evento com as proteções que valem para ele.</p>
      </div>
      <div className="flex flex-col justify-between gap-8">
        <div role="tablist" aria-label="Proteções" className="flex gap-6">
          {(Object.keys(ABAS) as (keyof typeof ABAS)[]).map((k) => (
            <button
              key={k}
              role="tab"
              aria-selected={aba === k}
              onClick={() => setAba(k)}
              className={`text-lg transition-colors ${aba === k ? 'font-semibold text-foreground' : 'text-muted-foreground hover:text-foreground'}`}
            >
              {ABAS[k].rotulo}
            </button>
          ))}
        </div>
        <div role="tabpanel">
          <h2 id="titulo-garante" className="text-3xl font-bold text-foreground text-balance sm:text-4xl" style={{ letterSpacing: '-0.02em' }}>
            {atual.titulo}
          </h2>
          <p className="mt-6 max-w-md text-base text-foreground text-pretty">{atual.texto}</p>
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Link to={atual.acao.para} className="flex h-16 items-center justify-center rounded-lg bg-primary text-base font-semibold text-primary-foreground">
            {atual.acao.rotulo}
          </Link>
          <Link to="/mapa" className="flex h-16 items-center justify-center rounded-lg bg-muted text-base font-semibold text-foreground">
            Eventos perto de você
          </Link>
        </div>
      </div>
    </section>
  )
}

const PERGUNTAS: [string, string][] = [
  ['Quem paga a taxa?', 'O comprador, somada ao preço e mostrada antes do pagamento. Se preferir, você assume a taxa e o comprador vê só o preço do ingresso; ela sai do seu repasse.'],
  ['Evento gratuito paga alguma coisa?', 'Não. Inscrição gratuita não tem taxa e não passa pelo pagamento: a confirmação sai na hora.'],
  ['Quando eu recebo?', 'O repasse é liberado a partir de 3 dias úteis depois do fim do evento e enviado por Pix para a chave do seu perfil. Você acompanha tudo no extrato.'],
  ['E a taxa do cartão e do Pix?', 'É cobrada pelo Mercado Pago e descontada do repasse. No painel do evento você vê quanto foi e quanto economizou com as vendas no Pix.'],
  ['Tem mensalidade ou fidelidade?', 'Não. Criar, publicar e fazer check-in não custa nada. Você só paga quando vende.'],
  ['Dá para parcelar?', 'Por enquanto o pagamento é à vista, no Pix ou no cartão.'],
  ['E se o evento for cancelado?', 'Todos os compradores recebem tudo de volta automaticamente, preço e taxa, sem precisar pedir.'],
  ['O que é a garantia de vaga?', 'Uma opção que o comprador marca na compra: por 5% do ingresso (mínimo R$ 1,99, máximo R$ 14,90), pode cancelar até o início do evento e receber tudo de volta.'],
  ['Por que parte do meu repasse fica retida?', 'Nos seus 3 primeiros repasses, ou até a verificação do perfil, 10% das vendas no cartão ficam retidos 60 dias contra contestações. Pix não entra nessa conta.'],
  ['A taxa muda de estado para estado?', 'É a mesma em todo o Brasil. Onde a lei limita a taxa, como em Alagoas, ela é reduzida sozinha até o limite.'],
]

function Perguntas() {
  return (
    <section aria-labelledby="titulo-perguntas" className="mx-auto grid w-full max-w-5xl grid-cols-1 gap-8 md:grid-cols-12">
      <h2 id="titulo-perguntas" className="text-2xl font-bold text-foreground md:col-span-3 md:text-3xl">
        Perguntas
      </h2>
      <div className="md:col-span-9" style={{ columnGap: 56 }}>
        <div className="sm:columns-2" style={{ columnGap: 56 }}>
          {PERGUNTAS.map(([p, r]) => (
            <p key={p} className="mb-5 text-base text-foreground" style={{ breakInside: 'avoid', lineHeight: 1.55 }}>
              <span className="font-semibold">{p}</span> {r}
            </p>
          ))}
        </div>
      </div>
    </section>
  )
}

function Fechamento() {
  const navigate = useNavigate()
  const [titulo, setTitulo] = useState('Concurso de Cosplay de Maceió')
  const [tipo, setTipo] = useState<'ingresso' | 'cadastro'>('ingresso')
  return (
    <section aria-labelledby="titulo-fechamento" className="bg-primary px-4 py-16 text-primary-foreground sm:py-24">
      <div className="mx-auto grid max-w-5xl grid-cols-1 items-end gap-10 md:grid-cols-2">
        <div>
          <h2 id="titulo-fechamento" className="text-5xl font-bold leading-none text-balance sm:text-7xl" style={{ letterSpacing: '-0.04em' }}>
            Comece grátis.
          </h2>
          <p className="mt-5 max-w-sm text-base opacity-90">Dê um nome e continue: em poucos minutos o evento está no ar, com link para divulgar.</p>
        </div>
        <form
          data-ui
          onSubmit={(e) => {
            e.preventDefault()
            navigate(`/organizador/eventos/novo?titulo=${encodeURIComponent(titulo.trim())}&tipo=${tipo}`)
          }}
          className="w-full max-w-md justify-self-end bg-background p-6 text-foreground"
          style={{ borderRadius: 12 }}
        >
          <p className="text-lg font-semibold">Criar evento</p>
          <label htmlFor="nome-evento" className="mt-4 block text-sm font-medium">
            Nome do evento
          </label>
          <input
            id="nome-evento"
            value={titulo}
            maxLength={200}
            onChange={(e) => setTitulo(e.target.value)}
            className="mt-1.5 h-11 w-full rounded-lg border border-border bg-background px-3 text-base outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
          <div role="radiogroup" aria-label="Tipo de acesso" className="mt-4 grid grid-cols-2 gap-1 rounded-lg bg-muted p-1 text-sm">
            {(['ingresso', 'cadastro'] as const).map((v) => (
              <button
                key={v}
                type="button"
                role="radio"
                aria-checked={tipo === v}
                onClick={() => setTipo(v)}
                className={`h-9 rounded-md transition-colors ${tipo === v ? 'bg-background font-semibold text-foreground' : 'text-foreground hover:bg-background/60'}`}
              >
                {v === 'ingresso' ? 'Vende ingresso' : 'Gratuito'}
              </button>
            ))}
          </div>
          <button type="submit" disabled={titulo.trim().length < 2} className="mt-5 h-12 w-full rounded-lg bg-primary text-base font-semibold text-primary-foreground transition-transform active:scale-95 disabled:opacity-50">
            Continuar
          </button>
        </form>
      </div>
    </section>
  )
}

export default function QuantoCusta() {
  const { data } = useQuery({ queryKey: ['taxas'], queryFn: () => api<TabelaTaxas>('/taxas'), staleTime: 600_000 })
  useEffect(() => {
    document.title = 'Quanto custa | Evve'
  }, [])
  useEffect(() => {
    if (data && window.location.hash) document.querySelector(window.location.hash)?.scrollIntoView()
  }, [data])
  const tabela = useMemo(() => data, [data])

  if (!tabela) return <main className="min-h-svh bg-muted" />

  return (
    <main className="bg-muted">
      <div className="px-3 py-10 sm:px-6 sm:py-16">
        <Surgir>
          <Ingresso tabela={tabela} />
        </Surgir>
      </div>
      <div className="px-3 pb-16 sm:px-6 sm:pb-24">
        <Surgir>
          <Tabela tabela={tabela} />
        </Surgir>
      </div>
      <div className="bg-background px-4 py-20 sm:py-28">
        <Surgir>
          <Garante />
        </Surgir>
      </div>
      <div className="bg-background px-4 pb-24 sm:pb-36">
        <Surgir>
          <Perguntas />
        </Surgir>
      </div>
      <Surgir>
        <Fechamento />
      </Surgir>
    </main>
  )
}
