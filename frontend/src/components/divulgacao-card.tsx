import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, ChevronDown, Copy } from 'lucide-react'
import { useEffect, useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Pixels = { meta: string; ga4: string; tiktok: string }

// Divulgação do evento (item 7.5), escrita para quem nunca usou pixel nem
// widget: cada parte diz para que serve, se a pessoa precisa e o passo a
// passo. Vai do mais simples (links) ao mais técnico (pixel).

type Plataforma = {
  chave: keyof Pixels
  nome: string
  exemplo: string
  formato: RegExp
  dicaFormato: string
  onde: { texto: string; link: string; passos: string[]; aviso?: string }
  eventos: string
}

const PLATAFORMAS: Plataforma[] = [
  {
    chave: 'meta',
    nome: 'Meta (Instagram e Facebook)',
    exemplo: '1234567890123456',
    formato: /^[0-9]{10,20}$/,
    dicaFormato: 'Só números (normalmente 15 ou 16).',
    onde: {
      texto: 'Gerenciador de Eventos da Meta',
      link: 'https://business.facebook.com/events_manager2',
      passos: [
        'Abra o Gerenciador de Eventos (link abaixo) com a conta que você usa para anunciar.',
        'Em "Fontes de dados", clique no seu pixel. Se ainda não tiver, use "Conectar dados" → "Web" para criar um.',
        'Copie o "ID do pixel" (uma sequência de números) e cole no campo acima.',
      ],
      aviso:
        'Em Configurações do pixel, deixe desligada a "Correspondência avançada automática". Assim nenhum dado digitado pelo comprador vai para a Meta.',
    },
    eventos: 'PageView, ViewContent, InitiateCheckout e Purchase',
  },
  {
    chave: 'ga4',
    nome: 'Google Analytics 4',
    exemplo: 'G-ABC123XYZ9',
    formato: /^G-[A-Z0-9]{4,15}$/i,
    dicaFormato: 'Começa com G- (ex.: G-ABC123XYZ9).',
    onde: {
      texto: 'Google Analytics',
      link: 'https://analytics.google.com/',
      passos: [
        'Entre no Google Analytics e clique na engrenagem "Administrador" (canto de baixo, à esquerda).',
        'Abra "Fluxos de dados" e escolha o fluxo da web. Se não tiver, crie um com o endereço do seu site ou do Evve.',
        'Copie o "ID da métrica", que começa com G-, e cole no campo acima.',
      ],
    },
    eventos: 'view_item, begin_checkout e purchase',
  },
  {
    chave: 'tiktok',
    nome: 'TikTok',
    exemplo: 'C4ABCDEFGHIJ12345678',
    formato: /^[A-Z0-9]{15,30}$/i,
    dicaFormato: 'Letras e números, sem espaços.',
    onde: {
      texto: 'Gerenciador de Eventos do TikTok',
      link: 'https://ads.tiktok.com/',
      passos: [
        'Entre no TikTok Ads Manager e vá em "Ferramentas" → "Eventos".',
        'Em "Eventos da web", abra o seu pixel (ou crie um em "Configurar fonte de dados da web").',
        'Copie o código do pixel (letras e números) e cole no campo acima.',
      ],
    },
    eventos: 'ViewContent, InitiateCheckout e CompletePayment',
  },
]

function BotaoCopiar({ texto, rotulo }: { texto: string; rotulo: string }) {
  const [ok, setOk] = useState(false)
  const copiar = async () => {
    try {
      await navigator.clipboard.writeText(texto)
      setOk(true)
      window.setTimeout(() => setOk(false), 2000)
    } catch {
      setOk(false)
    }
  }
  return (
    <Button type="button" variant="outline" size="sm" onClick={copiar} aria-live="polite">
      {ok ? <Check className="size-4" aria-hidden="true" /> : <Copy className="size-4" aria-hidden="true" />}
      {ok ? 'Copiado' : rotulo}
    </Button>
  )
}

// Passo a passo que abre e fecha (fica fechado para não poluir).
function ComoFazer({ titulo, children }: { titulo: string; children: React.ReactNode }) {
  return (
    <details className="group rounded-lg bg-muted px-3 py-2 text-sm text-foreground">
      <summary className="flex cursor-pointer list-none items-center justify-between gap-2 font-medium">
        {titulo}
        <ChevronDown className="size-4 shrink-0 transition-transform group-open:rotate-180" aria-hidden="true" />
      </summary>
      <div className="mt-2 flex flex-col gap-2">{children}</div>
    </details>
  )
}

function Secao({
  titulo,
  praQue,
  precisa,
  children,
}: {
  titulo: string
  praQue: string
  precisa: string
  children: React.ReactNode
}) {
  return (
    <section className="flex flex-col gap-3 border-t border-border pt-5 first:border-t-0 first:pt-0">
      <div>
        <h3 className="font-semibold text-foreground">{titulo}</h3>
        <p className="mt-1 text-sm text-foreground">{praQue}</p>
        <p className="mt-1 text-sm text-muted-foreground">{precisa}</p>
      </div>
      {children}
    </section>
  )
}

export function DivulgacaoCard({ eventoId, slug, ingressos = 1 }: { eventoId: number; slug: string; ingressos?: number }) {
  // Altura da vitrine: título (até 2 linhas no celular) e botão, mais uma linha por tipo de ingresso.
  const altura = 250 + 40 * Math.max(1, ingressos)
  const queryClient = useQueryClient()
  const { data: perfil } = useQuery({ queryKey: ['org-perfil'], queryFn: () => api<{ slug: string }>('/org/perfil') })
  const { data: salvos } = useQuery({
    queryKey: ['org-pixels', eventoId],
    queryFn: () => api<Pixels>(`/org/eventos/${eventoId}/pixels`),
  })
  const [p, setP] = useState<Pixels>({ meta: '', ga4: '', tiktok: '' })
  const [salvando, setSalvando] = useState(false)
  const [msg, setMsg] = useState<{ ok: boolean; texto: string } | null>(null)

  useEffect(() => {
    if (salvos) setP(salvos)
  }, [salvos])

  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null)
    setSalvando(true)
    try {
      const novo = await api<Pixels>(`/org/eventos/${eventoId}/pixels`, { method: 'PUT', body: p })
      setP(novo)
      queryClient.setQueryData(['org-pixels', eventoId], novo)
      const ligados = PLATAFORMAS.filter((x) => novo[x.chave]).map((x) => x.nome.split(' (')[0])
      setMsg({ ok: true, texto: ligados.length ? `Salvo. Ligado: ${ligados.join(', ')}.` : 'Salvo. Nenhum pixel ligado.' })
    } catch (err) {
      setMsg({ ok: false, texto: err instanceof ApiError ? err.message : 'Não foi possível salvar.' })
    } finally {
      setSalvando(false)
    }
  }

  const origem = window.location.origin
  const linkEvento = `${origem}/e/${slug}`
  const linkPagina = perfil?.slug ? `${origem}/o/${perfil.slug}` : ''
  const codigoWidget = `<iframe src="${origem}/widget/e/${slug}" title="Ingressos" style="width:100%;max-width:420px;height:${altura}px;border:0" loading="lazy"></iframe>`

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Divulgação</CardTitle>
        <p className="text-sm text-muted-foreground">
          Três formas de levar gente para o seu evento, da mais simples à mais avançada.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-5">
        <Secao
          titulo="1. Links para compartilhar"
          praQue="Endereços prontos para colar onde o seu público está: bio do Instagram, Stories, WhatsApp, e-mail."
          precisa="Todo mundo usa. Comece por aqui."
        >
          <div className="flex flex-col gap-1">
            <p className="text-sm font-medium text-foreground">Link deste evento</p>
            <div className="flex flex-wrap items-center gap-2">
              <code className="break-all rounded-md bg-muted px-2 py-1 text-xs text-foreground">{linkEvento}</code>
              <BotaoCopiar texto={linkEvento} rotulo="Copiar" />
            </div>
          </div>
          {linkPagina && (
            <div className="flex flex-col gap-1">
              <p className="text-sm font-medium text-foreground">Link da sua página (todos os seus eventos)</p>
              <div className="flex flex-wrap items-center gap-2">
                <code className="break-all rounded-md bg-muted px-2 py-1 text-xs text-foreground">{linkPagina}</code>
                <BotaoCopiar texto={linkPagina} rotulo="Copiar" />
              </div>
              <p className="text-xs text-muted-foreground">Ideal para a bio: muda sozinho quando você cria um evento novo.</p>
            </div>
          )}
          <ComoFazer titulo="Como colocar na bio e nos Stories">
            <ol className="list-decimal pl-5">
              <li>
                <strong>Bio do Instagram:</strong> "Editar perfil" → "Links" → "Adicionar link externo" e cole o link da sua
                página.
              </li>
              <li>
                <strong>Stories:</strong> crie o story, toque no ícone de figurinhas, escolha "Link" e cole o link do evento.
              </li>
              <li>
                <strong>TikTok:</strong> "Editar perfil" → "Site" (aparece para contas comerciais) e cole o link da sua página.
              </li>
              <li>
                <strong>WhatsApp:</strong> cole o link do evento na mensagem; a capa e o nome do evento aparecem sozinhos.
              </li>
            </ol>
          </ComoFazer>
        </Secao>

        <Secao
          titulo="2. Vitrine de ingressos no seu site"
          praQue="Um quadro com os ingressos e o preço final que você coloca no seu próprio site, como quem coloca um vídeo do YouTube. Quem clica em comprar vai para o Evve."
          precisa="Só se você tem site próprio (Wix, WordPress, Google Sites...). Os preços e o 'esgotado' se atualizam sozinhos."
        >
          <div className="flex flex-col gap-1">
            <p className="text-sm font-medium text-foreground">Assim ele aparece no seu site:</p>
            <iframe
              src={`/widget/e/${slug}`}
              title="Prévia da vitrine de ingressos"
              loading="lazy"
              className="w-full rounded-lg border border-dashed border-border"
              style={{ maxWidth: 420, height: altura }}
            />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <BotaoCopiar texto={codigoWidget} rotulo="Copiar código da vitrine" />
          </div>
          <ComoFazer titulo="Como colar no seu site (passo a passo)">
            <ol className="list-decimal pl-5">
              <li>Clique em "Copiar código da vitrine".</li>
              <li>
                No editor do seu site, adicione um bloco de código: no <strong>WordPress</strong> é "HTML personalizado"; no{' '}
                <strong>Wix</strong>, "Adicionar" → "Incorporar código" → "Incorporar HTML"; no <strong>Google Sites</strong>,
                "Incorporar" → "Código de incorporação".
              </li>
              <li>Cole o código, ajuste o tamanho se quiser e publique o site.</li>
            </ol>
            <p className="text-xs text-muted-foreground">O código é este (não precisa mexer nele):</p>
            <code className="block break-all rounded-md bg-background px-2 py-1 font-mono text-xs text-foreground">
              {codigoWidget}
            </code>
          </ComoFazer>
        </Secao>

        <Secao
          titulo="3. Medir seus anúncios (pixel)"
          praQue="Se você paga anúncio no Instagram, Facebook, Google ou TikTok, o pixel conta quantas pessoas viram o evento e compraram por causa do anúncio. Assim você sabe quanto gastou por ingresso vendido, e a plataforma aprende a mostrar o anúncio para quem tem mais chance de comprar."
          precisa="Só se você anuncia. Se não faz anúncio pago, pode pular esta parte."
        >
          <form onSubmit={salvar} className="flex flex-col gap-4">
            {PLATAFORMAS.map((pl) => {
              const valor = p[pl.chave].trim()
              const invalido = valor !== '' && !pl.formato.test(valor)
              const ligado = !!salvos?.[pl.chave]
              const id = `px-${pl.chave}`
              return (
                <div key={pl.chave} className="flex flex-col gap-1.5">
                  <div className="flex flex-wrap items-center gap-2">
                    <Label htmlFor={id}>{pl.nome}</Label>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs ${ligado ? 'bg-foreground text-background' : 'bg-muted text-muted-foreground'}`}
                    >
                      {ligado ? 'Ligado' : 'Desligado'}
                    </span>
                  </div>
                  <Input
                    id={id}
                    value={p[pl.chave]}
                    onChange={(e) => setP({ ...p, [pl.chave]: e.target.value })}
                    placeholder={`Ex.: ${pl.exemplo}`}
                    aria-invalid={invalido}
                    aria-describedby={`${id}-dica`}
                    className="max-w-sm"
                  />
                  <p id={`${id}-dica`} className={`text-xs ${invalido ? 'text-destructive' : 'text-muted-foreground'}`}>
                    {invalido ? `Formato diferente do esperado. ${pl.dicaFormato}` : pl.dicaFormato} Deixe vazio para desligar.
                  </p>
                  <ComoFazer
                    titulo={`Onde encontro o código ${pl.chave === 'ga4' ? 'do Google' : pl.chave === 'meta' ? 'da Meta' : 'do TikTok'}?`}
                  >
                    <ol className="list-decimal pl-5">
                      {pl.onde.passos.map((s) => (
                        <li key={s}>{s}</li>
                      ))}
                    </ol>
                    {pl.onde.aviso && (
                      <p className="rounded-md bg-background px-2 py-1 text-xs text-foreground">⚠️ {pl.onde.aviso}</p>
                    )}
                    <a
                      href={pl.onde.link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-sm underline underline-offset-4"
                    >
                      Abrir o {pl.onde.texto}
                    </a>
                    <p className="text-xs text-muted-foreground">
                      O caminho nos menus pode mudar um pouco quando a plataforma atualiza o site.
                    </p>
                  </ComoFazer>
                </div>
              )
            })}

            <div className="flex flex-wrap items-center gap-3">
              <Button type="submit" disabled={salvando}>
                {salvando ? 'Salvando…' : 'Salvar pixels'}
              </Button>
              {msg && (
                <p role="status" className={`text-sm ${msg.ok ? 'text-foreground' : 'text-destructive'}`}>
                  {msg.texto}
                </p>
              )}
            </div>
          </form>

          <ComoFazer titulo="O que é medido e como conferir se funcionou">
            <p>
              <strong>O que é medido:</strong> alguém viu a página do evento, começou a compra e concluiu a compra (com o valor e
              a quantidade). Na Meta esses passos aparecem como {PLATAFORMAS[0].eventos}; no Google, {PLATAFORMAS[1].eventos}; no
              TikTok, {PLATAFORMAS[2].eventos}.
            </p>
            <p>
              <strong>O que não é enviado:</strong> nome, e-mail, CPF e dados de pagamento do comprador. O pixel só funciona nas
              páginas deste evento.
            </p>
            <p>
              <strong>Por que os números ficam menores que as vendas:</strong> pela lei (LGPD), o pixel só liga para quem aceita
              os cookies no aviso do site. Quem recusa compra normalmente, mas não é contado.
            </p>
            <p>
              <strong>Como testar:</strong> salve o código, abra a página do evento numa janela anônima, clique em "Aceitar" no
              aviso de cookies e, depois de alguns minutos, veja o evento aparecer no painel da plataforma (na Meta: Gerenciador
              de Eventos → seu pixel → "Visão geral").
            </p>
          </ComoFazer>
        </Secao>
      </CardContent>
    </Card>
  )
}
