import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useParams } from 'react-router-dom'

import { api } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { calcularItem } from '@/lib/taxas'
import type { EventoDetalhe } from '@/lib/publico'

// Widget de compra (item 7.5) para o organizador colar no site dele num
// <iframe>. Mostra o evento e os ingressos com o preço final; a compra abre o
// Evve numa aba nova (dentro de um iframe de outro site, o login e o
// pagamento não funcionam bem). Sem pixels aqui: o site do organizador já
// mede a própria página.
export default function WidgetEvento() {
  const { slug = '' } = useParams()
  const { data, isLoading, error } = useQuery({
    queryKey: ['evento-publico', slug],
    queryFn: () => api<EventoDetalhe>(`/eventos/${slug}`, { semAuth: true }),
  })
  useEffect(() => {
    document.documentElement.style.background = 'transparent'
  }, [])

  const link = `${window.location.origin}/e/${slug}?utm_source=widget&utm_medium=site_organizador`

  if (isLoading) return <p className="p-4 text-sm text-muted-foreground">Carregando…</p>
  if (error || !data) return <p className="p-4 text-sm text-muted-foreground">Evento indisponível.</p>

  const { evento, local, ingressos, taxas, vendas_encerradas: encerradas } = data
  const quando = evento.inicio_em
    ? new Date(evento.inicio_em).toLocaleString('pt-BR', {
        weekday: 'short',
        day: '2-digit',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit',
      })
    : ''

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-border bg-background p-4 text-foreground">
      <div>
        <p className="text-lg font-semibold leading-tight">{evento.titulo}</p>
        <p className="text-sm text-muted-foreground">
          {quando}
          {local ? ` · ${local.nome}, ${local.cidade}/${local.uf}` : ''}
        </p>
      </div>
      <ul className="flex flex-col divide-y divide-border">
        {ingressos.map((i) => (
          <li key={i.id} className="flex items-center justify-between gap-3 py-2 text-sm">
            <span>{i.nome}</span>
            <span className="tabular-nums">
              {i.esgotado ? (
                <span className="text-muted-foreground">Esgotado</span>
              ) : i.preco_centavos === 0 ? (
                'Gratuito'
              ) : (
                formatarCentavos(calcularItem(taxas, i.preco_centavos, false).total)
              )}
            </span>
          </li>
        ))}
      </ul>
      <a
        href={link}
        target="_blank"
        rel="noopener"
        className="rounded-lg bg-foreground px-4 py-2.5 text-center text-sm font-medium text-background transition-opacity hover:opacity-90 active:opacity-80"
      >
        {encerradas ? 'Ver evento' : ingressos.every((i) => i.esgotado) ? 'Entrar na lista de espera' : 'Comprar ingressos'}
      </a>
      <p className="text-center text-xs text-muted-foreground">Preço final, já com a taxa. Venda segura pelo Evve.</p>
    </div>
  )
}
