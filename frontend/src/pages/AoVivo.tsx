import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { VoltarBotao } from '@/components/voltar-link'
import { ApiError } from '@/lib/api'
import { painelAoVivo, type PainelAoVivo } from '@/lib/dia-evento'
import { formatarCentavos } from '@/lib/evento'

// Item 9.6: painel do dia do evento para o organizador e a equipe.
// Atualiza a cada 10 s.
export default function AoVivo() {
  const id = Number(useParams().eventoId)
  const { data, error, isLoading, dataUpdatedAt } = useQuery({
    queryKey: ['ao-vivo', id],
    queryFn: () => painelAoVivo(id),
    refetchInterval: 10_000,
    refetchIntervalInBackground: true,
  })

  if (isLoading) return <Carregando />
  if (!data) {
    return (
      <main className="mx-auto max-w-md px-4 py-12 text-center">
        <p className="text-destructive">{error instanceof ApiError ? error.message : 'Sem conexão com o painel.'}</p>
      </main>
    )
  }
  const pct = data.esperados > 0 ? Math.round((data.presentes / data.esperados) * 100) : 0

  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <VoltarBotao fallback={`/checkin/${id}`} />
      <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">Ao vivo</h1>
          <p className="text-sm text-muted-foreground" aria-live="polite">
            Atualizado às {new Date(dataUpdatedAt).toLocaleTimeString('pt-BR')} · a cada 10 segundos
            {error ? ' · sem conexão, mostrando o último dado' : ''}
          </p>
        </div>
        <Link to={`/checkin/${id}`} className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground">
          Abrir o check-in
        </Link>
      </div>

      <section aria-label="Presença" className="grid gap-3 sm:grid-cols-3">
        <div className="rounded-xl border border-border p-4 sm:col-span-1">
          <p className="text-sm text-muted-foreground">Presentes</p>
          <p className="text-5xl font-semibold tabular-nums text-foreground">{data.presentes}</p>
          <p className="mt-1 text-sm text-muted-foreground tabular-nums">
            de {data.esperados} ({pct}%)
          </p>
          <div className="mt-3 h-2 overflow-hidden rounded-full bg-muted" role="presentation">
            <div className="h-full rounded-full bg-primary" style={{ width: `${pct}%` }} />
          </div>
        </div>
        <Numero rotulo="Faltam entrar" valor={data.faltam} nota={previsao(data)} />
        <Numero rotulo="Ritmo agora" valor={`${data.ritmo_por_minuto.toLocaleString('pt-BR', { maximumFractionDigits: 1 })}/min`} nota="média dos últimos 10 minutos" />
      </section>

      <section aria-labelledby="titulo-minuto" className="mt-6 rounded-xl border border-border p-4">
        <h2 id="titulo-minuto" className="text-base font-semibold text-foreground">
          Entradas por minuto (última hora)
        </h2>
        <GraficoMinutos dados={data.por_minuto} />
      </section>

      <div className="mt-6 grid gap-3 md:grid-cols-2">
        <section aria-labelledby="titulo-portoes" className="rounded-xl border border-border p-4">
          <h2 id="titulo-portoes" className="mb-3 text-base font-semibold text-foreground">
            Por portão
          </h2>
          {data.por_portao.length === 0 ? (
            <p className="text-sm text-muted-foreground">Nenhuma entrada registrada ainda.</p>
          ) : (
            <ul className="flex flex-col gap-3">
              {data.por_portao.map((p) => {
                const max = Math.max(...data.por_portao.map((x) => x.entradas), 1)
                return (
                  <li key={p.portao} className="flex flex-col gap-1">
                    <div className="flex justify-between gap-2 text-sm">
                      <span className="text-foreground">{p.portao || 'Sem portão'}</span>
                      <span className="tabular-nums text-muted-foreground">
                        <span className="font-medium text-foreground">{p.entradas}</span> · {p.ultimos_10} nos últimos 10 min
                      </span>
                    </div>
                    <div className="h-2 rounded-full bg-muted" role="presentation">
                      <div className="h-full rounded-full bg-primary" style={{ width: `${(p.entradas / max) * 100}%` }} />
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </section>

        <section aria-labelledby="titulo-outros" className="rounded-xl border border-border p-4">
          <h2 id="titulo-outros" className="mb-3 text-base font-semibold text-foreground">
            Porta, importados e credenciais
          </h2>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
            <dt className="text-muted-foreground">Vendas na porta</dt>
            <dd className="text-right tabular-nums text-foreground">
              {data.porta_ingressos} ingresso(s) em {data.porta_pedidos} venda(s)
            </dd>
            <dt className="text-muted-foreground">Dinheiro recebido</dt>
            <dd className="text-right tabular-nums text-foreground">{formatarCentavos(data.porta_dinheiro_centavos)}</dd>
            <dt className="text-muted-foreground">Pix na porta</dt>
            <dd className="text-right tabular-nums text-foreground">{formatarCentavos(data.porta_pix_centavos)}</dd>
            <dt className="text-muted-foreground">Importados</dt>
            <dd className="text-right tabular-nums text-foreground">
              {data.importados_entraram} de {data.importados_total} entraram
            </dd>
            <dt className="text-muted-foreground">Credenciais</dt>
            <dd className="text-right tabular-nums text-foreground">
              {data.credenciais_entraram} de {data.credenciais_total} entraram
            </dd>
          </dl>
        </section>
      </div>
    </main>
  )
}

function previsao(d: PainelAoVivo) {
  if (d.faltam === 0) return 'todo mundo entrou'
  if (d.previsao_minutos == null) return 'sem entradas nos últimos 10 minutos'
  return `cerca de ${d.previsao_minutos} min no ritmo atual`
}

function Numero({ rotulo, valor, nota }: { rotulo: string; valor: number | string; nota: string }) {
  return (
    <div className="rounded-xl border border-border p-4">
      <p className="text-sm text-muted-foreground">{rotulo}</p>
      <p className="text-5xl font-semibold tabular-nums text-foreground">{valor}</p>
      <p className="mt-1 text-sm text-muted-foreground">{nota}</p>
    </div>
  )
}

// Barras finas, uma série só (a cor primária); passar o mouse ou focar
// mostra o minuto e o número. A tabela abaixo serve ao leitor de tela.
function GraficoMinutos({ dados }: { dados: PainelAoVivo['por_minuto'] }) {
  const max = Math.max(...dados.map((d) => d.entradas), 1)
  const hora = (iso: string) => new Date(iso).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
  return (
    <div className="mt-3">
      <div className="flex h-40 items-end gap-0.5 border-b border-border" aria-hidden="true">
        {dados.map((d) => (
          <div key={d.minuto} className="group relative flex h-full flex-1 items-end">
            <div
              className="w-full rounded-t-sm bg-primary transition-opacity group-hover:opacity-80"
              style={{ height: d.entradas > 0 ? `${Math.max((d.entradas / max) * 100, 3)}%` : '0%' }}
            />
            <span className="pointer-events-none absolute bottom-full left-1/2 z-10 mb-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md bg-foreground px-2 py-1 text-xs text-background group-hover:block">
              {hora(d.minuto)} · {d.entradas}
            </span>
          </div>
        ))}
      </div>
      <div className="mt-1 flex justify-between text-xs text-muted-foreground tabular-nums" aria-hidden="true">
        <span>{hora(dados[0]?.minuto ?? new Date().toISOString())}</span>
        <span>{hora(dados[dados.length - 1]?.minuto ?? new Date().toISOString())}</span>
      </div>
      <table className="sr-only">
        <caption>Entradas por minuto</caption>
        <thead>
          <tr>
            <th>Minuto</th>
            <th>Entradas</th>
          </tr>
        </thead>
        <tbody>
          {dados
            .filter((d) => d.entradas > 0)
            .map((d) => (
              <tr key={d.minuto}>
                <td>{hora(d.minuto)}</td>
                <td>{d.entradas}</td>
              </tr>
            ))}
        </tbody>
      </table>
    </div>
  )
}
