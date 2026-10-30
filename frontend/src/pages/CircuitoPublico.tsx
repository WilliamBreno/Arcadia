import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { VoltarBotao } from '@/components/voltar-link'
import { ApiError } from '@/lib/api'
import { dataCurta, obterCircuito, rotuloCategoria } from '@/lib/circuito'

// Página pública do circuito (item 8.5): /circuitos/:slug.
// Ranking por categoria com pontos das etapas que já liberaram o resultado.
export default function CircuitoPublico() {
  const { slug = '' } = useParams()
  const { data, isLoading, error } = useQuery({ queryKey: ['circuito', slug], queryFn: () => obterCircuito(slug) })
  const [aba, setAba] = useState<string | null>(null)

  useEffect(() => {
    if (data) document.title = `${data.circuito.nome} | Evve`
  }, [data])

  if (isLoading) return <Carregando />
  if (error instanceof ApiError && error.status === 404) {
    return (
      <main className="mx-auto max-w-5xl px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-foreground">Circuito não encontrado</h1>
        <Link to="/" className="mt-6 inline-block text-sm font-medium underline underline-offset-4">
          Voltar para o início
        </Link>
      </main>
    )
  }
  if (!data) return null

  const categorias = Object.keys(data.ranking).sort((a, b) => (rotuloCategoria[a] ?? a).localeCompare(rotuloCategoria[b] ?? b))
  const atual = aba && data.ranking[aba] ? aba : categorias[0]
  const linhas = atual ? data.ranking[atual] : []
  const { circuito: c } = data

  return (
    <main className="mx-auto max-w-5xl px-4 py-10">
      <VoltarBotao fallback="/" />
      <p className="text-sm font-medium uppercase tracking-wide text-muted-foreground">Circuito</p>
      <h1 className="mt-1 text-3xl font-bold text-foreground text-balance sm:text-4xl">{c.nome}</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Temporada de {dataCurta(c.temporada_inicio)} a {dataCurta(c.temporada_fim)}
      </p>
      {c.descricao && <p className="mt-3 max-w-2xl text-foreground text-pretty">{c.descricao}</p>}

      <div className="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
        <section aria-labelledby="titulo-ranking" className="lg:col-span-2">
          <h2 id="titulo-ranking" className="text-xl font-semibold text-foreground">
            Ranking
          </h2>
          {categorias.length === 0 ? (
            <p className="mt-3 rounded-xl border border-border p-5 text-sm text-muted-foreground">
              O ranking começa quando a primeira etapa liberar o resultado.
            </p>
          ) : (
            <>
              {categorias.length > 1 && (
                <div role="tablist" aria-label="Categoria" className="mt-3 flex flex-wrap gap-2">
                  {categorias.map((t) => (
                    <button
                      key={t}
                      type="button"
                      role="tab"
                      aria-selected={t === atual}
                      onClick={() => setAba(t)}
                      className={`min-h-9 rounded-full border px-4 text-sm ${
                        t === atual ? 'border-foreground bg-foreground text-background' : 'border-border text-foreground hover:bg-muted'
                      }`}
                    >
                      {rotuloCategoria[t] ?? t}
                    </button>
                  ))}
                </div>
              )}
              <div className="mt-4 overflow-x-auto rounded-xl border border-border">
                <table className="w-full text-sm">
                  <caption className="sr-only">Ranking de {rotuloCategoria[atual] ?? atual}</caption>
                  <thead className="bg-muted text-left text-xs uppercase tracking-wide text-muted-foreground">
                    <tr>
                      <th scope="col" className="px-3 py-2 font-medium">
                        Pos.
                      </th>
                      <th scope="col" className="px-3 py-2 font-medium">
                        Talento
                      </th>
                      <th scope="col" className="px-3 py-2 text-right font-medium">
                        Pontos
                      </th>
                      <th scope="col" className="hidden px-3 py-2 text-right font-medium sm:table-cell">
                        Vitórias
                      </th>
                      <th scope="col" className="hidden px-3 py-2 text-right font-medium sm:table-cell">
                        Etapas
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {linhas.map((l, i) => (
                      <tr key={`${atual}-${i}`} className="border-t border-border">
                        <td className="px-3 py-2.5 font-semibold tabular-nums text-foreground">{l.posicao}º</td>
                        <td className="px-3 py-2.5 text-foreground">
                          {l.perfil_slug ? (
                            <Link to={`/t/${l.perfil_slug}`} className="underline underline-offset-4">
                              {l.nome}
                            </Link>
                          ) : (
                            l.nome
                          )}
                        </td>
                        <td className="px-3 py-2.5 text-right font-semibold tabular-nums text-foreground">{l.pontos}</td>
                        <td className="hidden px-3 py-2.5 text-right tabular-nums text-muted-foreground sm:table-cell">{l.vitorias}</td>
                        <td className="hidden px-3 py-2.5 text-right tabular-nums text-muted-foreground sm:table-cell">{l.eventos}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
          <details className="mt-4 text-sm text-muted-foreground">
            <summary className="cursor-pointer text-foreground">Como os pontos funcionam</summary>
            <p className="mt-2 text-pretty">
              Em cada etapa, os 10 primeiros de cada categoria pontuam:{' '}
              {data.pontos_por_posicao.map((p, i) => `${i + 1}º ${p}`).join(' · ')}. Os pontos entram quando o organizador da etapa libera o resultado.
              Empate: vence quem tem mais vitórias, depois mais segundos lugares.
            </p>
          </details>
        </section>

        <section aria-labelledby="titulo-etapas">
          <h2 id="titulo-etapas" className="text-xl font-semibold text-foreground">
            Etapas
          </h2>
          {data.etapas.length === 0 ? (
            <p className="mt-3 text-sm text-muted-foreground">Nenhuma etapa confirmada ainda.</p>
          ) : (
            <ol className="mt-3 flex flex-col gap-2">
              {data.etapas.map((e) => (
                <li key={e.slug}>
                  <Link to={`/e/${e.slug}`} className="block rounded-xl border border-border p-3 hover:bg-muted">
                    <span className="block font-medium text-foreground">{e.titulo}</span>
                    <span className="mt-0.5 block text-xs text-muted-foreground">
                      {e.inicio_em ? new Date(e.inicio_em).toLocaleDateString('pt-BR', { dateStyle: 'medium' }) : 'Data a definir'}
                      {' · '}
                      {e.resultado_liberado ? 'resultado no ranking' : 'resultado ainda não saiu'}
                    </span>
                  </Link>
                </li>
              ))}
            </ol>
          )}
        </section>
      </div>
    </main>
  )
}
