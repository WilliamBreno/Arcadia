import { useQuery } from '@tanstack/react-query'

import { Carregando } from '@/components/carregando'
import { api } from '@/lib/api'
import type { ComponenteStatus, StatusPlataforma } from '@/lib/ajuda'

const rotulo: Record<ComponenteStatus['status'], string> = { operacional: 'Funcionando', degradado: 'Instável', falha: 'Fora do ar' }
const bolinha: Record<ComponenteStatus['status'], string> = { operacional: 'bg-green-600', degradado: 'bg-yellow-500', falha: 'bg-red-600' }
const manchete: Record<ComponenteStatus['status'], string> = {
  operacional: 'Tudo funcionando',
  degradado: 'Parte da plataforma está instável',
  falha: 'Parte da plataforma está fora do ar',
}
const nivel: Record<string, string> = { aviso: 'Aviso', falha: 'Falha', manutencao: 'Manutenção' }
const quando = (iso: string) => new Date(iso).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })

// Item 9.9: status público da plataforma (atualiza a cada 30 s).
export default function Status() {
  const { data, isLoading, isError, dataUpdatedAt } = useQuery({
    queryKey: ['status-plataforma'],
    queryFn: () => api<StatusPlataforma>('/status', { semAuth: true }),
    refetchInterval: 30_000,
  })
  if (isLoading) return <Carregando />

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <h1 className="text-3xl font-semibold text-foreground">Status</h1>
      {isError || !data ? (
        <p className="mt-4 rounded-xl border-2 border-destructive p-4 text-destructive">Não conseguimos falar com a plataforma agora. Tente de novo em instantes.</p>
      ) : (
        <>
          <p className="mt-4 flex items-center gap-3 rounded-xl border border-border p-4 text-lg font-medium text-foreground" aria-live="polite">
            <span className={`size-3 shrink-0 rounded-full ${bolinha[data.geral]}`} aria-hidden="true" />
            {manchete[data.geral]}
          </p>

          {data.avisos.length > 0 && (
            <section aria-labelledby="titulo-avisos" className="mt-6 flex flex-col gap-3">
              <h2 id="titulo-avisos" className="text-lg font-semibold text-foreground">
                Acontecendo agora
              </h2>
              {data.avisos.map((a) => (
                <article key={a.id} className="rounded-xl border border-border p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    {nivel[a.nivel]} · desde {quando(a.inicio_em)}
                  </p>
                  <h3 className="mt-1 font-semibold text-foreground">{a.titulo}</h3>
                  {a.mensagem && <p className="mt-1 text-sm text-foreground">{a.mensagem}</p>}
                </article>
              ))}
            </section>
          )}

          <ul className="mt-6 divide-y divide-border rounded-xl border border-border">
            {data.componentes.map((c) => (
              <li key={c.chave} className="flex items-center justify-between gap-3 px-4 py-3">
                <span className="text-foreground">{c.nome}</span>
                <span className="flex items-center gap-2 text-sm text-muted-foreground">
                  {c.detalhe && <span className="hidden sm:inline">{c.detalhe} ·</span>}
                  <span className={`size-2.5 rounded-full ${bolinha[c.status]}`} aria-hidden="true" />
                  {rotulo[c.status]}
                </span>
              </li>
            ))}
          </ul>
          <p className="mt-2 text-xs text-muted-foreground">Conferido às {new Date(dataUpdatedAt).toLocaleTimeString('pt-BR')}. Atualiza sozinho a cada 30 segundos.</p>

          {data.recentes.length > 0 && (
            <section aria-labelledby="titulo-recentes" className="mt-10">
              <h2 id="titulo-recentes" className="mb-3 text-lg font-semibold text-foreground">
                Últimos 14 dias
              </h2>
              <ul className="flex flex-col gap-3">
                {data.recentes.map((a) => (
                  <li key={a.id} className="text-sm">
                    <p className="font-medium text-foreground">{a.titulo}</p>
                    <p className="text-muted-foreground">
                      {quando(a.inicio_em)} até {a.fim_em ? quando(a.fim_em) : '—'} · resolvido
                    </p>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}
    </main>
  )
}
