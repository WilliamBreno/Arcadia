// SEO das páginas públicas (item 6.1). A SPA entrega um HTML vazio; o
// Google até executa JavaScript, mas indexa melhor (e mais rápido) quando
// recebe o conteúdo pronto. O middleware da Vercel usa estas funções para
// devolver aos robôs de busca e de redes sociais o mesmo conteúdo que a
// pessoa vê: título, descrição, link canônico, Open Graph, dados
// estruturados schema.org (que fazem o evento aparecer na busca de eventos
// do Google) e o texto principal da página.
//
// Funções puras, testadas em render.test.ts (node --test).

import { calcularItem, type RegraTaxas } from '../src/lib/taxas.ts'

export type EventoSEO = {
  evento: {
    titulo: string
    slug: string
    descricao: string
    categoria: string
    capa_url: string
    inicio_em: string | null
    fim_em: string | null
    timezone: string
    visibilidade: string
  }
  local: { nome: string; logradouro: string; numero: string; bairro: string; cidade: string; uf: string; latitude: number | null; longitude: number | null } | null
  organizador: { nome_publico: string; slug: string } | null
  ingressos: { nome: string; preco_centavos: number; ativo: boolean; vendas_inicio?: string | null }[]
  taxas: RegraTaxas
  vendas_encerradas?: boolean
}

export type OrganizadorSEO = {
  organizador: { nome_publico: string; slug: string; descricao: string; logo_url: string; instagram: string; site: string; verificado?: boolean }
  eventos: { slug: string; titulo: string; inicio_em: string | null; cidade?: string; uf?: string }[]
}

export function escapeHtml(v: string): string {
  return v.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}

// JSON dentro de <script>: impede que "</script>" no texto do organizador
// feche a tag (injeção de HTML).
function jsonSeguro(obj: unknown): string {
  return JSON.stringify(obj).replace(/</g, '\\u003c').replace(/>/g, '\\u003e').replace(/&/g, '\\u0026')
}

function resumo(texto: string, max = 160): string {
  const limpo = texto.replace(/\s+/g, ' ').trim()
  return limpo.length <= max ? limpo : limpo.slice(0, max - 1).trimEnd() + '…'
}

function dataPorExtenso(iso: string | null, tz: string): string {
  if (!iso) return ''
  try {
    return new Intl.DateTimeFormat('pt-BR', { dateStyle: 'full', timeStyle: 'short', timeZone: tz || 'America/Maceio' }).format(new Date(iso))
  } catch {
    return new Date(iso).toISOString()
  }
}

function reais(centavos: number): string {
  return (centavos / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
}

// Troca <title>, a descrição e as tags OG/Twitter genéricas do index.html,
// e põe o conteúdo dentro de #root (o React substitui ao carregar).
function aplicar(html: string, head: string[], corpo: string): string {
  const semGenericas = html
    .replace(/<title>[\s\S]*?<\/title>\s*/i, '')
    .replace(/<meta (?:name="description"|property="og:[^"]*"|name="twitter:[^"]*")[^>]*>\s*/gi, '')
    .replace(/<link rel="canonical"[^>]*>\s*/gi, '')
  return semGenericas
    .replace('</head>', `    ${head.join('\n    ')}\n  </head>`)
    // A tela de carregamento (#splash) dá lugar ao conteúdo.
    .replace(/<div id="splash">[\s\S]*?<\/div>/, '')
    .replace('<div id="root">', `<div id="root">${corpo}`)
}

export function jsonLdEvento(d: EventoSEO, base: string): Record<string, unknown> {
  const url = `${base}/e/${d.evento.slug}`
  const ld: Record<string, unknown> = {
    '@context': 'https://schema.org',
    '@type': 'Event',
    name: d.evento.titulo,
    url,
    eventStatus: 'https://schema.org/EventScheduled',
    eventAttendanceMode: d.local ? 'https://schema.org/OfflineEventAttendanceMode' : 'https://schema.org/OnlineEventAttendanceMode',
  }
  if (d.evento.descricao) ld.description = resumo(d.evento.descricao, 5000)
  if (d.evento.inicio_em) ld.startDate = d.evento.inicio_em
  if (d.evento.fim_em) ld.endDate = d.evento.fim_em
  if (d.evento.capa_url) ld.image = [d.evento.capa_url]
  if (d.local) {
    const rua = [d.local.logradouro, d.local.numero].filter(Boolean).join(', ')
    const lugar: Record<string, unknown> = {
      '@type': 'Place',
      name: d.local.nome,
      address: {
        '@type': 'PostalAddress',
        ...(rua ? { streetAddress: rua } : {}),
        addressLocality: d.local.cidade,
        addressRegion: d.local.uf,
        addressCountry: 'BR',
      },
    }
    if (d.local.latitude != null && d.local.longitude != null) {
      lugar.geo = { '@type': 'GeoCoordinates', latitude: d.local.latitude, longitude: d.local.longitude }
    }
    ld.location = lugar
  } else {
    ld.location = { '@type': 'VirtualLocation', url }
  }
  if (d.organizador) {
    ld.organizer = { '@type': 'Organization', name: d.organizador.nome_publico, url: `${base}/o/${d.organizador.slug}` }
  }
  // Preço que o comprador paga de fato (preço + taxa da faixa), em reais.
  const ofertas = d.ingressos
    .filter((i) => i.ativo)
    .map((i) => {
      const oferta: Record<string, unknown> = {
        '@type': 'Offer',
        name: i.nome,
        price: (calcularItem(d.taxas, i.preco_centavos, false).total / 100).toFixed(2),
        priceCurrency: 'BRL',
        url,
      }
      if (!d.vendas_encerradas) oferta.availability = 'https://schema.org/InStock'
      if (i.vendas_inicio) oferta.validFrom = i.vendas_inicio
      return oferta
    })
  if (ofertas.length > 0) ld.offers = ofertas
  return ld
}

export function renderizarEvento(html: string, d: EventoSEO, base: string): string {
  const e = d.evento
  const url = `${base}/e/${e.slug}`
  const onde = d.local ? `${d.local.nome} · ${d.local.cidade}/${d.local.uf}` : 'Online'
  const quando = dataPorExtenso(e.inicio_em, e.timezone)
  const titulo = `${e.titulo} — ingressos | Evve`
  const descricao = resumo(e.descricao || `Ingressos para ${e.titulo}${quando ? ` em ${quando}` : ''} · ${onde}`)
  const imagem = e.capa_url || `${base}/brand/og.png`

  const head = [
    `<title>${escapeHtml(titulo)}</title>`,
    `<meta name="description" content="${escapeHtml(descricao)}" />`,
    `<link rel="canonical" href="${escapeHtml(url)}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:site_name" content="Evve" />`,
    `<meta property="og:title" content="${escapeHtml(e.titulo)}" />`,
    `<meta property="og:description" content="${escapeHtml(descricao)}" />`,
    `<meta property="og:url" content="${escapeHtml(url)}" />`,
    `<meta property="og:image" content="${escapeHtml(imagem)}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<script type="application/ld+json">${jsonSeguro(jsonLdEvento(d, base))}</script>`,
  ]
  // Evento "não listado" só é acessível por quem tem o link: fora do Google.
  if (e.visibilidade !== 'publico') head.push(`<meta name="robots" content="noindex" />`)

  const ingressos = d.ingressos
    .filter((i) => i.ativo)
    .map((i) => {
      const v = calcularItem(d.taxas, i.preco_centavos, false)
      const preco = i.preco_centavos === 0 ? 'Gratuito' : v.taxa > 0 ? `${reais(i.preco_centavos)} + ${reais(v.taxa)} de taxa` : reais(i.preco_centavos)
      return `<li>${escapeHtml(i.nome)}: ${escapeHtml(preco)}</li>`
    })
    .join('')

  const corpo = [
    '<main>',
    `<h1>${escapeHtml(e.titulo)}</h1>`,
    quando ? `<p>${escapeHtml(quando)}</p>` : '',
    `<p>${escapeHtml(onde)}</p>`,
    d.organizador ? `<p>por <a href="/o/${escapeHtml(d.organizador.slug)}">${escapeHtml(d.organizador.nome_publico)}</a></p>` : '',
    e.descricao ? `<p>${escapeHtml(e.descricao)}</p>` : '',
    ingressos ? `<h2>Ingressos</h2><ul>${ingressos}</ul>` : '',
    d.vendas_encerradas ? '<p>Vendas encerradas.</p>' : '',
    '</main>',
  ].join('')
  return aplicar(html, head, corpo)
}

export function renderizarOrganizador(html: string, d: OrganizadorSEO, base: string): string {
  const o = d.organizador
  const url = `${base}/o/${o.slug}`
  const descricao = resumo(o.descricao || `Eventos de ${o.nome_publico} no Evve: ingressos, datas e locais.`)
  const sameAs = [o.instagram ? `https://instagram.com/${o.instagram.replace(/^@/, '')}` : '', o.site].filter(Boolean)
  const ld = {
    '@context': 'https://schema.org',
    '@type': 'Organization',
    name: o.nome_publico,
    url,
    ...(o.logo_url ? { logo: o.logo_url } : {}),
    ...(o.descricao ? { description: resumo(o.descricao, 5000) } : {}),
    ...(sameAs.length ? { sameAs } : {}),
  }
  const head = [
    `<title>${escapeHtml(`${o.nome_publico} — eventos | Evve`)}</title>`,
    `<meta name="description" content="${escapeHtml(descricao)}" />`,
    `<link rel="canonical" href="${escapeHtml(url)}" />`,
    `<meta property="og:type" content="profile" />`,
    `<meta property="og:site_name" content="Evve" />`,
    `<meta property="og:title" content="${escapeHtml(o.nome_publico)}" />`,
    `<meta property="og:description" content="${escapeHtml(descricao)}" />`,
    `<meta property="og:url" content="${escapeHtml(url)}" />`,
    `<meta property="og:image" content="${escapeHtml(o.logo_url || `${base}/brand/og.png`)}" />`,
    `<meta name="twitter:card" content="summary" />`,
    `<script type="application/ld+json">${jsonSeguro(ld)}</script>`,
  ]
  // Links para cada evento: é por eles que o Google descobre as páginas.
  const lista = d.eventos
    .map((e) => `<li><a href="/e/${escapeHtml(e.slug)}">${escapeHtml(e.titulo)}</a>${e.cidade ? ` · ${escapeHtml(e.cidade)}/${escapeHtml(e.uf ?? '')}` : ''}</li>`)
    .join('')
  const corpo = [
    '<main>',
    `<h1>${escapeHtml(o.nome_publico)}</h1>`,
    o.descricao ? `<p>${escapeHtml(o.descricao)}</p>` : '',
    lista ? `<h2>Eventos</h2><ul>${lista}</ul>` : '',
    '</main>',
  ].join('')
  return aplicar(html, head, corpo)
}

// --- Páginas de lista: cidade, cidade + categoria e coleção (item 6.2) ---

export type ListaSEO = {
  titulo: string // "Eventos em Maceió"
  descricao: string
  caminho: string // "/eventos/maceio-al" (com ?quando= se houver)
  eventos: { slug: string; titulo: string; inicio_em: string | null; cidade?: string; uf?: string }[]
  semIndexar?: boolean // página vazia não deve ir para o Google
}

export function renderizarLista(html: string, d: ListaSEO, base: string): string {
  const url = `${base}${d.caminho}`
  const ld = {
    '@context': 'https://schema.org',
    '@type': 'ItemList',
    name: d.titulo,
    url,
    numberOfItems: d.eventos.length,
    itemListElement: d.eventos.slice(0, 50).map((e, i) => ({ '@type': 'ListItem', position: i + 1, url: `${base}/e/${e.slug}`, name: e.titulo })),
  }
  const head = [
    `<title>${escapeHtml(`${d.titulo} | Evve`)}</title>`,
    `<meta name="description" content="${escapeHtml(resumo(d.descricao))}" />`,
    `<link rel="canonical" href="${escapeHtml(url)}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:site_name" content="Evve" />`,
    `<meta property="og:title" content="${escapeHtml(d.titulo)}" />`,
    `<meta property="og:description" content="${escapeHtml(resumo(d.descricao))}" />`,
    `<meta property="og:url" content="${escapeHtml(url)}" />`,
    `<meta property="og:image" content="${escapeHtml(`${base}/brand/og.png`)}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<script type="application/ld+json">${jsonSeguro(ld)}</script>`,
  ]
  if (d.semIndexar) head.push(`<meta name="robots" content="noindex" />`)
  const lista = d.eventos
    .map((e) => {
      const quando = e.inicio_em ? ` · ${dataPorExtenso(e.inicio_em, 'America/Maceio')}` : ''
      return `<li><a href="/e/${escapeHtml(e.slug)}">${escapeHtml(e.titulo)}</a>${escapeHtml(quando)}</li>`
    })
    .join('')
  const corpo = ['<main>', `<h1>${escapeHtml(d.titulo)}</h1>`, `<p>${escapeHtml(d.descricao)}</p>`, lista ? `<ul>${lista}</ul>` : '<p>Nenhum evento no momento.</p>', '</main>'].join('')
  return aplicar(html, head, corpo)
}

const ROTULO_QUANDO: Record<string, string> = { hoje: 'hoje', amanha: 'amanhã', 'fim-de-semana': 'neste fim de semana' }

// Monta o título e a descrição de uma página de cidade (igual à tela).
export function textosCidade(cidade: string, uf: string, categoria: string | null, quando: string, total: number) {
  const q = ROTULO_QUANDO[quando] ? ` ${ROTULO_QUANDO[quando]}` : ''
  const titulo = `${categoria ? `${categoria} em` : 'Eventos em'} ${cidade}${q}`
  const descricao =
    total > 0
      ? `${total} evento(s)${categoria ? ` de ${categoria}` : ''} em ${cidade}/${uf}${q}: datas, locais e ingressos no Evve.`
      : `Eventos${categoria ? ` de ${categoria}` : ''} em ${cidade}/${uf}${q} no Evve.`
  return { titulo, descricao }
}
