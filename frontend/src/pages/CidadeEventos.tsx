import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'

import { ApiError } from '@/lib/api'
import { listarEventos, obterDescoberta, obterDestaques, QUANDO } from '@/lib/descoberta'
import { ChipLink } from '@/components/chip'
import { BotaoSeguir } from '@/components/botao-seguir'
import { EventoCard } from '@/components/evento-card'
import { TrilhoEventos } from '@/components/trilho-eventos'
import { Carregando } from '@/components/carregando'

const ROTULO_QUANDO: Record<string, string> = {
  hoje: 'hoje',
  amanha: 'amanhã',
  'fim-de-semana': 'neste fim de semana',
}

// Página de descoberta por cidade, categoria e data (item 6.2):
// /eventos/maceio-al, /eventos/maceio-al/danca, ?quando=fim-de-semana.
// São as páginas que o Google mostra para "eventos em Maceió".
export default function CidadeEventos() {
  const { cidade: slugCidade = '', categoria: slugCategoria } = useParams()
  const [params] = useSearchParams()
  const quando = params.get('quando') ?? ''

  const descoberta = useQuery({ queryKey: ['descoberta', slugCidade], queryFn: () => obterDescoberta(slugCidade) })
  const cidade = descoberta.data?.cidade
  const categoria = descoberta.data?.categorias.find((c) => c.slug === slugCategoria)

  const eventos = useQuery({
    queryKey: ['eventos-cidade', cidade?.cidade, cidade?.uf, categoria?.nome, quando],
    enabled: !!cidade && (!slugCategoria || !!categoria),
    queryFn: () => listarEventos({ cidade: cidade!.cidade, uf: cidade!.uf, categoria: categoria?.nome, quando }),
  })
  const destaques = useQuery({
    queryKey: ['destaques', slugCidade],
    enabled: !!cidade && !slugCategoria && !quando,
    queryFn: () => obterDestaques(slugCidade),
  })

  const titulo = cidade
    ? `${categoria ? `${categoria.nome} em` : 'Eventos em'} ${cidade.cidade}${quando ? ` ${ROTULO_QUANDO[quando] ?? ''}` : ''}`
    : ''

  useEffect(() => {
    if (titulo) document.title = `${titulo} | Evve`
  }, [titulo])

  if (descoberta.isLoading) return <Carregando />
  if (descoberta.error instanceof ApiError && descoberta.error.status === 404) {
    return (
      <main className="mx-auto max-w-5xl px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-foreground">Ainda não há eventos nesta cidade</h1>
        <p className="mt-2 text-muted-foreground">Veja o que está rolando em outras cidades.</p>
        <Link to="/" className="mt-6 inline-block text-sm font-medium underline underline-offset-4">
          Voltar para o início
        </Link>
      </main>
    )
  }
  if (!cidade) return null

  const base = `/eventos/${cidade.slug}`
  const comQuando = (caminho: string, q: string) => (q ? `${caminho}?quando=${q}` : caminho)
  const caminhoCategoria = slugCategoria && categoria ? `${base}/${categoria.slug}` : base

  return (
    <main className="mx-auto max-w-5xl px-4 py-10">
      <nav aria-label="Você está em" className="mb-2 text-sm text-muted-foreground">
        <Link to="/" className="hover:text-foreground">
          Início
        </Link>{' '}
        /{' '}
        {categoria ? (
          <Link to={base} className="hover:text-foreground">
            {cidade.cidade}
          </Link>
        ) : (
          <span>{cidade.cidade}</span>
        )}
        {categoria && <> / <span>{categoria.nome}</span></>}
      </nav>
      <h1 className="text-3xl font-bold text-foreground text-balance sm:text-4xl">{titulo}</h1>
      <p className="mt-1 text-muted-foreground">
        {cidade.cidade}/{cidade.uf} · {eventos.data ? `${eventos.data.total} evento(s)` : `${cidade.eventos} evento(s) programados`}
      </p>
      {categoria && (
        <div className="mt-3">
          <BotaoSeguir
            tipo="categoria"
            alvo={`${categoria.slug}@${cidade.slug}`}
            rotulo={`${categoria.nome} em ${cidade.cidade}`}
            texto={`Seguir ${categoria.nome} em ${cidade.cidade}`}
          />
        </div>
      )}

      <div className="mt-6 flex flex-col gap-3">
        <div className="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1" aria-label="Categorias">
          <ChipLink to={comQuando(base, quando)} ativoAgora={!slugCategoria}>
            Todas
          </ChipLink>
          {descoberta.data?.categorias.map((c) => (
            <ChipLink key={c.slug} to={comQuando(`${base}/${c.slug}`, quando)} ativoAgora={c.slug === slugCategoria}>
              {c.nome} <span className="tabular-nums opacity-70">{c.eventos}</span>
            </ChipLink>
          ))}
        </div>
        <div className="flex flex-wrap gap-2" aria-label="Quando">
          {QUANDO.map((q) => (
            <ChipLink key={q.valor} to={comQuando(caminhoCategoria, q.valor)} ativoAgora={quando === q.valor}>
              {q.rotulo}
            </ChipLink>
          ))}
        </div>
      </div>

      <div className="mt-8">
        {destaques.data && (
          <>
            <TrilhoEventos titulo={`Em alta em ${cidade.cidade}`} subtitulo="Os mais comprados nos últimos 7 dias" eventos={destaques.data.em_alta} ranking />
            <TrilhoEventos titulo="Quase esgotando" eventos={destaques.data.quase_esgotando} etiqueta="Quase esgotando" />
          </>
        )}

        <h2 className="mb-3 text-xl font-bold text-foreground">{destaques.data ? 'Todos os eventos' : 'Eventos'}</h2>
        {eventos.isLoading && <p className="text-muted-foreground">Carregando…</p>}
        {eventos.data?.eventos.length === 0 && (
          <p className="rounded-xl border border-border p-6 text-muted-foreground">
            Nenhum evento {quando ? ROTULO_QUANDO[quando] : 'por aqui'}.{' '}
            {quando && (
              <Link to={caminhoCategoria} className="font-medium text-foreground underline underline-offset-4">
                Ver todas as datas
              </Link>
            )}
          </p>
        )}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
          {eventos.data?.eventos.map((e) => (
            <EventoCard key={e.slug} evento={e} />
          ))}
        </div>
      </div>

      {(descoberta.data?.cidades.length ?? 0) > 1 && (
        <section className="mt-12 border-t border-border pt-6">
          <h2 className="mb-3 text-sm font-medium text-muted-foreground">Outras cidades</h2>
          <div className="flex flex-wrap gap-2">
            {descoberta.data!.cidades
              .filter((c) => c.slug !== cidade.slug)
              .map((c) => (
                <ChipLink key={c.slug} to={`/eventos/${c.slug}`}>
                  {c.cidade}/{c.uf}
                </ChipLink>
              ))}
          </div>
        </section>
      )}
    </main>
  )
}
