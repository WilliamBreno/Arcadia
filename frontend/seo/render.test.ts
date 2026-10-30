// Testes do SEO das páginas públicas (item 6.1). Rodar: npm run test:seo
import assert from 'node:assert/strict'
import { test } from 'node:test'

import { jsonLdEvento, renderizarEvento, renderizarOrganizador, renderizarTalento, type EventoSEO, type TalentoSEO } from './render.ts'

const HTML = `<!doctype html><html><head>
    <title>Evve — Eventos sem limites</title>
    <meta name="description" content="genérica" />
    <meta property="og:title" content="genérico" />
    <meta name="twitter:card" content="summary_large_image" />
  </head><body><div id="root"><div id="splash"><img src="/brand/icon-192.png" alt="Evve" /></div></div></body></html>`

const taxas = {
  faixas: [
    { ate_centavos: 3000, taxa_centavos: 99 },
    { ate_centavos: 8000, taxa_centavos: 199 },
    { ate_centavos: -1, taxa_centavos: 699 },
  ],
  garantia_percentual_bp: 500,
  garantia_min_centavos: 199,
  garantia_max_centavos: 1490,
  limite_bp: null,
  taxa_paga_por: 'comprador' as const,
}

function evento(extra: Partial<EventoSEO['evento']> = {}): EventoSEO {
  return {
    evento: {
      titulo: 'Concurso de Cosplay <Maceió>',
      slug: 'cosplay-maceio',
      descricao: 'Desfile e apresentação. </script><script>alert(1)</script>',
      categoria: 'geek',
      capa_url: 'https://cdn.evve/capa.jpg',
      inicio_em: '2026-11-20T22:00:00Z',
      fim_em: '2026-11-21T03:00:00Z',
      timezone: 'America/Maceio',
      visibilidade: 'publico',
      ...extra,
    },
    local: { nome: 'Centro de Convenções', logradouro: 'Av. Brasil', numero: '100', bairro: 'Jaraguá', cidade: 'Maceió', uf: 'AL', latitude: -9.66, longitude: -35.73 },
    organizador: { nome_publico: 'Geek AL', slug: 'geek-al' },
    ingressos: [
      { nome: 'Inteira', preco_centavos: 5000, ativo: true },
      { nome: 'Gratuito', preco_centavos: 0, ativo: true },
      { nome: 'Lote antigo', preco_centavos: 3000, ativo: false },
    ],
    taxas,
  }
}

test('página do evento traz título, descrição, canônico e conteúdo', () => {
  const html = renderizarEvento(HTML, evento(), 'https://evve.app')
  assert.match(html, /<title>Concurso de Cosplay &lt;Maceió&gt; — ingressos \| Evve<\/title>/)
  assert.match(html, /<link rel="canonical" href="https:\/\/evve.app\/e\/cosplay-maceio" \/>/)
  assert.equal((html.match(/<title>/g) ?? []).length, 1, 'título genérico removido')
  assert.ok(!html.includes('content="genérica"'), 'descrição genérica removida')
  assert.match(html, /<div id="root"><main><h1>/)
  assert.match(html, /Inteira: R\$\s?50,00 \+ R\$\s?1,99 de taxa/)
  assert.ok(!html.includes('Lote antigo'), 'ingresso inativo fora')
  assert.ok(!html.includes('noindex'))
  assert.ok(!html.includes('id="splash"'), 'tela de carregamento removida')
  assert.match(html, /<\/main><\/div><\/body>/)
})

test('texto do organizador não consegue injetar script', () => {
  const html = renderizarEvento(HTML, evento(), 'https://evve.app')
  assert.ok(!html.includes('<script>alert(1)'), 'HTML do corpo escapado')
  const ld = html.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/)
  assert.ok(ld, 'JSON-LD presente')
  assert.ok(!ld[1].includes('</script'), 'JSON-LD não fecha a tag')
  JSON.parse(ld[1]) // continua sendo JSON válido
})

test('JSON-LD segue schema.org/Event com preço final', () => {
  const ld = jsonLdEvento(evento(), 'https://evve.app') as Record<string, any>
  assert.equal(ld['@type'], 'Event')
  assert.equal(ld.startDate, '2026-11-20T22:00:00Z')
  assert.equal(ld.eventAttendanceMode, 'https://schema.org/OfflineEventAttendanceMode')
  assert.equal(ld.location.address.addressLocality, 'Maceió')
  assert.equal(ld.location.geo.latitude, -9.66)
  assert.equal(ld.organizer.url, 'https://evve.app/o/geek-al')
  assert.deepEqual(ld.offers.map((o: any) => o.price), ['51.99', '0.00'])
  assert.equal(ld.offers[0].priceCurrency, 'BRL')
  assert.equal(ld.offers[0].availability, 'https://schema.org/InStock')
})

test('evento online usa VirtualLocation', () => {
  const d = evento()
  d.local = null
  const ld = jsonLdEvento(d, 'https://evve.app') as Record<string, any>
  assert.equal(ld.location['@type'], 'VirtualLocation')
  assert.equal(ld.eventAttendanceMode, 'https://schema.org/OnlineEventAttendanceMode')
})

test('organizador absorvendo a taxa: preço sem taxa', () => {
  const d = evento()
  d.taxas = { ...taxas, taxa_paga_por: 'organizador' }
  const ld = jsonLdEvento(d, 'https://evve.app') as Record<string, any>
  assert.equal(ld.offers[0].price, '50.00')
})

test('evento não listado fica fora do Google', () => {
  const html = renderizarEvento(HTML, evento({ visibilidade: 'nao_listado' }), 'https://evve.app')
  assert.match(html, /<meta name="robots" content="noindex" \/>/)
})

test('vendas encerradas: sem availability', () => {
  const d = evento()
  d.vendas_encerradas = true
  const ld = jsonLdEvento(d, 'https://evve.app') as Record<string, any>
  assert.equal(ld.offers[0].availability, undefined)
})

test('página do organizador lista os eventos com link', () => {
  const html = renderizarOrganizador(HTML, {
    organizador: { nome_publico: 'Geek AL', slug: 'geek-al', descricao: '', logo_url: '', instagram: '@geekal', site: '' },
    eventos: [{ slug: 'cosplay-maceio', titulo: 'Cosplay', inicio_em: null, cidade: 'Maceió', uf: 'AL' }],
  }, 'https://evve.app')
  assert.match(html, /<a href="\/e\/cosplay-maceio">Cosplay<\/a> · Maceió\/AL/)
  assert.match(html, /"sameAs":\["https:\/\/instagram.com\/geekal"\]/)
  assert.match(html, /<link rel="canonical" href="https:\/\/evve.app\/o\/geek-al" \/>/)
})

test('robôs reconhecidos e humanos não', async () => {
  const { REGEX_ROBO } = await import('../middleware.ts')
  const robos = [
    'Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)',
    'Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)',
    'Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)',
    'WhatsApp/2.23.20.0',
    'facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)',
  ]
  for (const ua of robos) assert.ok(REGEX_ROBO.test(ua), ua)
  assert.ok(!REGEX_ROBO.test('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36'))
})

test('página da cidade: título, ItemList e links', async () => {
  const { renderizarLista, textosCidade } = await import('./render.ts')
  const t = textosCidade('Maceió', 'AL', 'Dança', 'fim-de-semana', 2)
  assert.equal(t.titulo, 'Dança em Maceió neste fim de semana')
  const html = renderizarLista(HTML, {
    ...t,
    caminho: '/eventos/maceio-al/danca?quando=fim-de-semana',
    eventos: [{ slug: 'batalha', titulo: 'Batalha <de> Dança', inicio_em: '2026-11-21T22:00:00Z' }],
  }, 'https://evve.app')
  assert.match(html, /<title>Dança em Maceió neste fim de semana \| Evve<\/title>/)
  assert.match(html, /<link rel="canonical" href="https:\/\/evve.app\/eventos\/maceio-al\/danca\?quando=fim-de-semana" \/>/)
  assert.match(html, /"@type":"ItemList"/)
  assert.match(html, /<a href="\/e\/batalha">Batalha &lt;de&gt; Dança<\/a>/)
  assert.ok(!html.includes('noindex'))
})

test('lista vazia não vai para o Google', async () => {
  const { renderizarLista } = await import('./render.ts')
  const html = renderizarLista(HTML, { titulo: 'X', descricao: 'Y', caminho: '/c/x', eventos: [], semIndexar: true }, 'https://evve.app')
  assert.match(html, /noindex/)
})

// Item 8.1: perfil de talento adulto vai ao Google; o de menor, não.
const talento: TalentoSEO = {
  slug: 'kira-cosplay',
  nome_artistico: 'Kira Cosplay',
  bio: 'Cosplayer de Maceió </script><script>alert(1)</script>',
  instagram: 'kira.cos',
  indexar: true,
  participacoes: [{ evento_titulo: 'Geek Fest', evento_slug: 'geek-fest', tipo: 'cosplay', personagem: 'Nezuko', posicao: 1 }],
}

test('perfil de talento adulto: Person, sem noindex e sem injeção', () => {
  const out = renderizarTalento(HTML, talento, 'https://evve.app')
  assert.match(out, /"@type":"Person"/)
  assert.match(out, /https:\/\/instagram.com\/kira.cos/)
  assert.doesNotMatch(out, /noindex/)
  assert.doesNotMatch(out, /<script>alert/)
  assert.match(out, /Nezuko · 1º lugar/)
})

test('perfil de talento menor: noindex e sem dados estruturados', () => {
  const out = renderizarTalento(HTML, { ...talento, indexar: false, instagram: undefined }, 'https://evve.app')
  assert.match(out, /<meta name="robots" content="noindex, nofollow" \/>/)
  assert.doesNotMatch(out, /application\/ld\+json/)
})
