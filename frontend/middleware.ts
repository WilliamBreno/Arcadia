// Vercel Edge Middleware: SEO das páginas públicas (itens 1.4 e 6.1).
//
// Como o frontend é uma SPA (Vite), o HTML servido vem vazio. Para robôs de
// busca (Google, Bing...) e de redes sociais (WhatsApp, Instagram...), este
// middleware busca o evento ou organizador na API e devolve o HTML já com
// título, descrição, link canônico, Open Graph, dados estruturados
// schema.org e o conteúdo principal (seo/render.ts). É o mesmo conteúdo que
// a pessoa vê depois que a SPA carrega ("dynamic rendering"); para humanos
// nada muda e a SPA carrega normal.
//
// Configuração necessária no projeto Vercel (Settings > Environment
// Variables): BACKEND_API_URL apontando para a API em produção
// (ex.: https://evve-api.onrender.com/api). Não é o mesmo que
// VITE_API_URL — aquela é embutida no build do cliente, esta só existe
// em runtime de Edge Function.
import { renderizarEvento, renderizarOrganizador, type EventoSEO, type OrganizadorSEO } from './seo/render.ts'

export const config = {
  matcher: ['/e/:slug*', '/o/:slug*'],
}

// Buscadores + redes sociais. Testado contra os user-agents oficiais.
export const REGEX_ROBO =
  /Googlebot|Google-InspectionTool|GoogleOther|Storebot-Google|bingbot|BingPreview|DuckDuckBot|YandexBot|Applebot|Baiduspider|facebookexternalhit|Facebot|Twitterbot|WhatsApp|Slackbot|LinkedInBot|TelegramBot|Discordbot|Pinterest/i

export default async function middleware(request: Request): Promise<Response> {
  const respostaOrigem = await fetch(request)

  const userAgent = request.headers.get('user-agent') ?? ''
  if (!REGEX_ROBO.test(userAgent)) {
    return respostaOrigem
  }

  const url = new URL(request.url)
  // Só a página principal (/e/:slug ou /o/:slug); subpáginas como
  // /e/:slug/jurado seguem como SPA.
  const partes = url.pathname.split('/').filter(Boolean)
  const apiURL = process.env.BACKEND_API_URL
  if (partes.length !== 2 || !apiURL) {
    return respostaOrigem
  }
  const [tipo, slug] = partes

  try {
    const caminho = tipo === 'e' ? `/v1/eventos/${encodeURIComponent(slug)}` : `/v1/organizadores/${encodeURIComponent(slug)}`
    const resposta = await fetch(`${apiURL}${caminho}`)
    if (!resposta.ok) return respostaOrigem

    const dados = await resposta.json()
    const html = await respostaOrigem.text()
    const pronto =
      tipo === 'e'
        ? renderizarEvento(html, dados as EventoSEO, url.origin)
        : renderizarOrganizador(html, dados as OrganizadorSEO, url.origin)

    return new Response(pronto, {
      status: respostaOrigem.status,
      headers: { 'content-type': 'text/html; charset=utf-8', 'cache-control': 'public, max-age=300' },
    })
  } catch {
    return respostaOrigem
  }
}
