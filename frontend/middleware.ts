// Vercel Edge Middleware: SEO das páginas públicas (itens 1.4, 6.1 e 6.2).
//
// Como o frontend é uma SPA (Vite), o HTML servido vem vazio. Para robôs de
// busca (Google, Bing...) e de redes sociais (WhatsApp, Instagram...), este
// middleware busca os dados na API e devolve o HTML já com título,
// descrição, link canônico, Open Graph, dados estruturados schema.org e o
// conteúdo principal (seo/render.ts). É o mesmo conteúdo que a pessoa vê
// depois que a SPA carrega ("dynamic rendering"); para humanos nada muda.
//
// Páginas: /e/:slug (evento), /o/:slug (organizador), /t/:slug (talento), /eventos/:cidade,
// /eventos/:cidade/:categoria (com ?quando=) e /c/:slug (coleção).
//
// Configuração necessária no projeto Vercel: BACKEND_API_URL apontando para
// a API em produção (ex.: https://evve-api.onrender.com/api).
import {
  renderizarEvento,
  renderizarLista,
  renderizarOrganizador,
  renderizarTalento,
  textosCidade,
  type EventoSEO,
  type ListaSEO,
  type OrganizadorSEO,
  type TalentoSEO,
} from './seo/render.ts'

export const config = {
  matcher: ['/e/:slug*', '/o/:slug*', '/t/:slug*', '/eventos/:path*', '/c/:slug*'],
}

// Buscadores + redes sociais. Testado contra os user-agents oficiais.
export const REGEX_ROBO =
  /Googlebot|Google-InspectionTool|GoogleOther|Storebot-Google|bingbot|BingPreview|DuckDuckBot|YandexBot|Applebot|Baiduspider|facebookexternalhit|Facebot|Twitterbot|WhatsApp|Slackbot|LinkedInBot|TelegramBot|Discordbot|Pinterest/i

const QUANDO_VALIDO = new Set(['hoje', 'amanha', 'fim-de-semana'])

type ItemLista = ListaSEO['eventos'][number]

async function json<T>(url: string): Promise<T | null> {
  const r = await fetch(url)
  return r.ok ? ((await r.json()) as T) : null
}

// Dados de uma página de cidade/categoria; null se a cidade não existir.
async function listaCidade(api: string, partes: string[], quando: string): Promise<ListaSEO | null> {
  const [, slugCidade, slugCategoria] = partes
  const d = await json<{ cidade: { cidade: string; uf: string } | null; categorias: { nome: string; slug: string }[] }>(
    `${api}/v1/descoberta?cidade=${encodeURIComponent(slugCidade)}`,
  )
  if (!d?.cidade) return null
  const categoria = slugCategoria ? d.categorias.find((c) => c.slug === slugCategoria) : undefined
  if (slugCategoria && !categoria) return null
  const p = new URLSearchParams({ cidade: d.cidade.cidade, uf: d.cidade.uf, por_pagina: '50' })
  if (categoria) p.set('categoria', categoria.nome)
  if (quando) p.set('atalho', quando)
  const lista = await json<{ eventos: ItemLista[]; total: number }>(`${api}/v1/eventos?${p}`)
  const eventos = lista?.eventos ?? []
  const caminho = `/eventos/${slugCidade}${categoria ? `/${categoria.slug}` : ''}${quando ? `?quando=${quando}` : ''}`
  return { ...textosCidade(d.cidade.cidade, d.cidade.uf, categoria?.nome ?? null, quando, lista?.total ?? 0), caminho, eventos, semIndexar: eventos.length === 0 }
}

export default async function middleware(request: Request): Promise<Response> {
  const respostaOrigem = await fetch(request)

  const userAgent = request.headers.get('user-agent') ?? ''
  if (!REGEX_ROBO.test(userAgent)) {
    return respostaOrigem
  }

  const url = new URL(request.url)
  const partes = url.pathname.split('/').filter(Boolean).map(decodeURIComponent)
  const apiURL = process.env.BACKEND_API_URL
  if (!apiURL || partes.length < 2) return respostaOrigem
  const [tipo, slug] = partes

  // Lê o HTML uma vez; daqui em diante só se devolvem respostas novas.
  const html = await respostaOrigem.text()
  const original = () => new Response(html, { status: respostaOrigem.status, headers: respostaOrigem.headers })
  try {
    let pronto: string | null = null

    if ((tipo === 'e' || tipo === 'o') && partes.length === 2) {
      // Subpáginas como /e/:slug/jurado seguem como SPA.
      const dados = await json<unknown>(`${apiURL}/v1/${tipo === 'e' ? 'eventos' : 'organizadores'}/${encodeURIComponent(slug)}`)
      if (dados) {
        pronto = tipo === 'e' ? renderizarEvento(html, dados as EventoSEO, url.origin) : renderizarOrganizador(html, dados as OrganizadorSEO, url.origin)
      }
    } else if (tipo === 'eventos' && partes.length <= 3) {
      const quando = QUANDO_VALIDO.has(url.searchParams.get('quando') ?? '') ? url.searchParams.get('quando')! : ''
      const lista = await listaCidade(apiURL, partes, quando)
      // Cidade ou categoria sem eventos: a tela mostra "ainda não há eventos";
      // não deixar o Google indexar uma página vazia.
      pronto = lista ? renderizarLista(html, lista, url.origin) : html.replace('</head>', '  <meta name="robots" content="noindex" />\n  </head>')
    } else if (tipo === 't' && partes.length === 2) {
      // Item 8.1: perfil do talento (o de menor sai com noindex).
      const t = await json<TalentoSEO>(`${apiURL}/v1/talentos/${encodeURIComponent(slug)}`)
      if (t) pronto = renderizarTalento(html, t, url.origin)
    } else if (tipo === 'c' && partes.length === 2) {
      const col = await json<{ slug: string; titulo: string; descricao: string; eventos: ItemLista[] }>(`${apiURL}/v1/colecoes/${encodeURIComponent(slug)}`)
      if (col) {
        pronto = renderizarLista(html, {
          titulo: col.titulo,
          descricao: col.descricao || `${col.titulo}: eventos selecionados pelo Evve.`,
          caminho: `/c/${col.slug}`,
          eventos: col.eventos,
          semIndexar: col.eventos.length === 0,
        }, url.origin)
      }
    }

    if (!pronto) return original()
    return new Response(pronto, {
      status: respostaOrigem.status,
      headers: { 'content-type': 'text/html; charset=utf-8', 'cache-control': 'public, max-age=300' },
    })
  } catch {
    return original()
  }
}
