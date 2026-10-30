// Pixels de anúncio do organizador (item 7.5): Meta, Google Analytics 4 e
// TikTok. Regras:
//  - só carregam depois do "Aceitar" no aviso de cookies;
//  - só nas páginas daquele evento (página do evento e pedido); ao sair, o
//    site recarrega e os scripts deixam de existir (RotaDosPixels);
//  - cada evento vai só para o pixel daquele organizador (trackSingle,
//    send_to, instance), nunca para pixels de outro evento já carregados;
//  - nenhum dado pessoal: só tipo de evento, valor, moeda e quantidade.
import { useEffect, useMemo, useState } from 'react'

import { aceitouMarketing, aoMudarEscolha } from '@/lib/consentimento'

export type Pixels = { meta: string; ga4: string; tiktok: string }

type W = Window & {
  fbq?: (...a: unknown[]) => void
  _fbq?: unknown
  dataLayer?: unknown[]
  gtag?: (...a: unknown[]) => void
  ttq?: { load: (id: string) => void; instance: (id: string) => { track: (e: string, p?: object) => void; page: () => void } }
  TiktokAnalyticsObject?: string
  __evvePixels?: boolean
  __evvePixelsSlug?: string
}
const w = window as unknown as W
const carregados = new Set<string>()

// O formato vem validado do servidor; conferimos de novo antes de pôr no script.
const valido = { meta: /^[0-9]{10,20}$/, ga4: /^G-[A-Z0-9]{4,15}$/, tiktok: /^[A-Z0-9]{15,30}$/ }

function script(src: string) {
  const s = document.createElement('script')
  s.async = true
  s.src = src
  document.head.appendChild(s)
}

function carregarMeta(id: string) {
  if (!w.fbq) {
    const n: {
      (...a: unknown[]): void
      callMethod?: (...a: unknown[]) => void
      queue: unknown[]
      push?: unknown
      loaded?: boolean
      version?: string
    } = Object.assign(
      (...a: unknown[]) => {
        if (n.callMethod) n.callMethod(...a)
        else n.queue.push(a)
      },
      { queue: [] as unknown[] },
    )
    n.push = n
    n.loaded = true
    n.version = '2.0'
    w.fbq = n
    w._fbq = n
    script('https://connect.facebook.net/en_US/fbevents.js')
  }
  // Sem "eventos automáticos" (cliques em botões de outras páginas).
  w.fbq('set', 'autoConfig', false, id)
  w.fbq('init', id)
}

function carregarGA4(id: string) {
  if (!w.gtag) {
    w.dataLayer = w.dataLayer || []
    w.gtag = function gtag() {
      // eslint-disable-next-line prefer-rest-params
      w.dataLayer!.push(arguments)
    }
    w.gtag('js', new Date())
  }
  script(`https://www.googletagmanager.com/gtag/js?id=${id}`)
  w.gtag('config', id, { send_page_view: false })
}

function carregarTikTok(id: string) {
  if (!w.ttq) {
    w.TiktokAnalyticsObject = 'ttq'
    const fila: unknown[] & Record<string, unknown> = Object.assign([], {}) as never
    const metodos = [
      'page',
      'track',
      'identify',
      'instances',
      'debug',
      'on',
      'off',
      'once',
      'ready',
      'alias',
      'group',
      'enableCookie',
      'disableCookie',
    ]
    const adiar = (alvo: Record<string, unknown> & unknown[], m: string) => {
      alvo[m] = (...a: unknown[]) => alvo.push([m, ...a])
    }
    metodos.forEach((m) => adiar(fila, m))
    fila.instance = (i: string) => {
      const inst = ((fila._i as Record<string, unknown[]>)[i] || []) as unknown[] & Record<string, unknown>
      metodos.forEach((m) => adiar(inst, m))
      return inst
    }
    fila.load = (i: string) => {
      const src = 'https://analytics.tiktok.com/i18n/pixel/events.js'
      fila._i = (fila._i as object) || {}
      ;(fila._i as Record<string, unknown>)[i] = Object.assign([], { _u: src })
      fila._t = (fila._t as object) || {}
      ;(fila._t as Record<string, number>)[i] = +new Date()
      fila._o = (fila._o as object) || {}
      ;(fila._o as Record<string, object>)[i] = {}
      script(`${src}?sdkid=${i}&lib=ttq`)
    }
    w.ttq = fila as unknown as W['ttq']
  }
  w.ttq!.load(id)
}

function carregar(p: Pixels) {
  if (p.meta && valido.meta.test(p.meta) && !carregados.has('m' + p.meta)) {
    carregarMeta(p.meta)
    carregados.add('m' + p.meta)
  }
  if (p.ga4 && valido.ga4.test(p.ga4) && !carregados.has('g' + p.ga4)) {
    carregarGA4(p.ga4)
    carregados.add('g' + p.ga4)
  }
  if (p.tiktok && valido.tiktok.test(p.tiktok) && !carregados.has('t' + p.tiktok)) {
    carregarTikTok(p.tiktok)
    carregados.add('t' + p.tiktok)
  }
  if (carregados.size > 0 && !w.__evvePixels) {
    w.__evvePixels = true
    // Evento em que os pixels foram carregados (ou nenhum, se foi no pedido).
    w.__evvePixelsSlug = window.location.pathname.match(/^\/e\/([^/]+)/)?.[1] ?? ''
  }
}

export type EventoPixel =
  | { tipo: 'ver' }
  | { tipo: 'iniciar'; valor: number; itens: number }
  | { tipo: 'comprar'; valor: number; itens: number; pedido: string }

function disparar(p: Pixels, e: EventoPixel) {
  const valor = 'valor' in e ? Math.round(e.valor) / 100 : undefined
  const base = valor !== undefined ? { value: valor, currency: 'BRL' } : {}
  if (p.meta && carregados.has('m' + p.meta) && w.fbq) {
    if (e.tipo === 'ver') {
      w.fbq('trackSingle', p.meta, 'PageView')
      w.fbq('trackSingle', p.meta, 'ViewContent')
    } else if (e.tipo === 'iniciar') w.fbq('trackSingle', p.meta, 'InitiateCheckout', { ...base, num_items: e.itens })
    else w.fbq('trackSingle', p.meta, 'Purchase', { ...base, num_items: e.itens }, { eventID: `evve-${e.pedido}` })
  }
  if (p.ga4 && carregados.has('g' + p.ga4) && w.gtag) {
    if (e.tipo === 'ver') w.gtag('event', 'view_item', { send_to: p.ga4 })
    else if (e.tipo === 'iniciar') w.gtag('event', 'begin_checkout', { send_to: p.ga4, ...base })
    else w.gtag('event', 'purchase', { send_to: p.ga4, transaction_id: `evve-${e.pedido}`, ...base })
  }
  if (p.tiktok && carregados.has('t' + p.tiktok) && w.ttq) {
    const t = w.ttq.instance(p.tiktok)
    if (e.tipo === 'ver') {
      t.page()
      t.track('ViewContent')
    } else if (e.tipo === 'iniciar') t.track('InitiateCheckout', { ...base, quantity: e.itens })
    else t.track('CompletePayment', { ...base, quantity: e.itens })
  }
}

const temAlgum = (p?: Pixels | null) => !!p && !!(p.meta || p.ga4 || p.tiktok)

// usePixels carrega os pixels do evento quando há consentimento (inclusive
// se a pessoa aceitar com a página aberta) e devolve como disparar eventos.
export function usePixels(p?: Pixels | null) {
  const [consentiu, setConsentiu] = useState(aceitouMarketing)
  useEffect(() => aoMudarEscolha((e) => setConsentiu(e.marketing)), [])
  const ativo = consentiu && temAlgum(p)
  useEffect(() => {
    if (ativo && p) carregar(p)
  }, [ativo, p])
  return useMemo(() => (e: EventoPixel) => ativo && p && disparar(p, e), [ativo, p])
}

// podeFicar: com os pixels de um evento carregados, só a página daquele
// evento e as páginas de pedido. Em qualquer outra, o site recarrega.
export function podeFicar(caminho: string, slugOrigem: string) {
  if (/^\/pedidos\/\d+\/?$/.test(caminho)) return true
  const slug = caminho.match(/^\/e\/([^/]+)\/?$/)?.[1]
  return !!slug && slug === slugOrigem
}
export const pixelsCarregados = () => (w.__evvePixels ? { slug: w.__evvePixelsSlug ?? '' } : null)
