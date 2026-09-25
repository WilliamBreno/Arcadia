// Vercel Edge Middleware: injeta meta tags Open Graph em /e/:slug para
// crawlers de redes sociais (WhatsApp, Instagram, Twitter/X, etc.).
//
// Como o frontend é uma SPA (Vite), o HTML servido não tem as tags OG do
// evento — crawlers não executam JS, então precisam receber o HTML já
// com <meta property="og:..."> preenchido. Só reescreve a resposta para
// user-agents conhecidos de crawler; para humanos, a SPA carrega normal.
//
// Configuração necessária no projeto Vercel (Settings > Environment
// Variables): BACKEND_API_URL apontando para a API em produção
// (ex.: https://evve-api.onrender.com/api). Não é o mesmo que
// VITE_API_URL — aquela é embutida no build do cliente, esta só existe
// em runtime de Edge Function.
export const config = {
  matcher: '/e/:slug*',
}

const REGEX_CRAWLER = /facebookexternalhit|Twitterbot|WhatsApp|Slackbot|LinkedInBot|TelegramBot|Discordbot|Pinterest/i

function escapeHtml(valor: string): string {
  return valor
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export default async function middleware(request: Request): Promise<Response> {
  const respostaOrigem = await fetch(request)

  const userAgent = request.headers.get('user-agent') ?? ''
  if (!REGEX_CRAWLER.test(userAgent)) {
    return respostaOrigem
  }

  const url = new URL(request.url)
  const slug = url.pathname.split('/e/')[1]?.split('/')[0]
  const apiURL = process.env.BACKEND_API_URL
  if (!slug || !apiURL) {
    return respostaOrigem
  }

  try {
    const respostaEvento = await fetch(`${apiURL}/v1/eventos/${slug}`)
    if (!respostaEvento.ok) return respostaOrigem

    const { evento } = (await respostaEvento.json()) as {
      evento: { titulo: string; descricao: string; capa_url: string }
    }
    const titulo = escapeHtml(evento.titulo ?? '')
    const descricao = escapeHtml((evento.descricao || `Ingressos para ${evento.titulo}`).slice(0, 200))
    const imagem = escapeHtml(evento.capa_url || `${url.origin}/brand/og.png`)

    const html = await respostaOrigem.text()
    const metaTags = [
      `<meta property="og:type" content="website" />`,
      `<meta property="og:title" content="${titulo}" />`,
      `<meta property="og:description" content="${descricao}" />`,
      `<meta property="og:url" content="${escapeHtml(url.toString())}" />`,
      `<meta property="og:image" content="${imagem}" />`,
      `<meta name="twitter:card" content="summary_large_image" />`,
    ].join('\n    ')

    const semMetaGenerica = html.replace(/<meta (?:property="og:|name="twitter:)[^>]*>\s*/g, '')
    const htmlComMeta = semMetaGenerica.replace('</head>', `    ${metaTags}\n  </head>`)

    return new Response(htmlComMeta, {
      status: respostaOrigem.status,
      headers: { 'content-type': 'text/html; charset=utf-8' },
    })
  } catch {
    return respostaOrigem
  }
}
