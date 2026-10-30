import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Check } from 'lucide-react'
import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

// Voto do público e devolutiva dos jurados (item 8.2).

const rotuloTipo: Record<string, string> = {
  cosplay: 'Cosplay',
  danca: 'Dança',
  canto: 'Canto',
  atuacao: 'Atuação',
  '': 'Concurso',
}

type Candidato = { ficha_id: number; nome: string; personagem?: string }
type Votacao = { aberta: boolean; categorias: Record<string, Candidato[]> }
type MeusVotos = { votos: Record<string, number>; ingressos: number }
export type LinhaVoto = { posicao: number; ficha_id: number; nome: string; votos: number; perfil_slug?: string }

// Votação na página do evento: aparece só com a votação aberta.
export function VotacaoPublico({ slug }: { slug: string }) {
  const { usuario } = useAuth()
  const location = useLocation()
  const queryClient = useQueryClient()
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState<number | null>(null)
  const { data: v } = useQuery({
    queryKey: ['votacao', slug],
    queryFn: () => api<Votacao>(`/eventos/${slug}/votacao`, { semAuth: true }),
    refetchInterval: 60_000,
  })
  const { data: meus } = useQuery({
    queryKey: ['meus-votos', slug, usuario?.id],
    enabled: !!usuario && !!v?.aberta,
    queryFn: () => api<MeusVotos>(`/eventos/${slug}/meus-votos`),
  })
  if (!v?.aberta) return null
  const categorias = Object.entries(v.categorias).filter(([, cs]) => cs.length > 0)
  if (categorias.length === 0) return null

  const votar = async (tipo: string, fichaId: number) => {
    setErro(null)
    setEnviando(fichaId)
    try {
      await api(`/eventos/${slug}/votos`, { method: 'POST', body: { tipo, ficha_id: fichaId } })
      queryClient.invalidateQueries({ queryKey: ['meus-votos', slug] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível votar.')
    } finally {
      setEnviando(null)
    }
  }

  return (
    <section aria-labelledby="titulo-votacao" className="mt-8 rounded-xl border border-foreground p-4 sm:p-5">
      <h2 id="titulo-votacao" className="text-xl font-semibold text-foreground">
        Voto do público
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">
        Escolha seu favorito em cada categoria. Vale para quem entrou no evento: cada ingresso usado é um voto. Dá para trocar
        enquanto a votação estiver aberta. O resultado sai junto com o dos jurados.
      </p>
      {!usuario && (
        <p className="mt-3 text-sm text-foreground">
          <Link to="/login" state={{ de: location.pathname }} className="font-medium underline underline-offset-4">
            Entre na sua conta
          </Link>{' '}
          (a mesma da compra do ingresso) para votar.
        </p>
      )}
      {usuario && meus && meus.ingressos === 0 && (
        <p className="mt-3 rounded-lg bg-muted p-3 text-sm text-foreground">
          Só quem passou pelo check-in vota. Seu ingresso ainda não foi lido na entrada.
        </p>
      )}
      {usuario && meus && meus.ingressos > 1 && (
        <p className="mt-3 text-sm text-foreground">Seu voto vale por {meus.ingressos} ingressos usados.</p>
      )}
      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}
      <div className="mt-4 flex flex-col gap-5">
        {categorias.map(([tipo, cs]) => (
          <div key={tipo}>
            <h3 className="mb-2 text-sm font-medium text-foreground">{rotuloTipo[tipo] ?? tipo}</h3>
            <ul className="grid gap-2 sm:grid-cols-2">
              {cs.map((c) => {
                const meu = meus?.votos[tipo] === c.ficha_id
                const pode = !!usuario && (meus?.ingressos ?? 0) > 0
                return (
                  <li key={c.ficha_id}>
                    <button
                      type="button"
                      disabled={!pode || enviando !== null}
                      aria-pressed={meu}
                      onClick={() => votar(tipo, c.ficha_id)}
                      className={`flex w-full items-center justify-between gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors active:opacity-90 disabled:cursor-not-allowed ${
                        meu ? 'border-foreground bg-muted' : 'border-border hover:bg-muted'
                      }`}
                    >
                      <span className="min-w-0">
                        <span className="block font-medium text-foreground">{c.nome}</span>
                        {c.personagem && <span className="block text-xs text-muted-foreground">{c.personagem}</span>}
                      </span>
                      {meu ? (
                        <span className="flex shrink-0 items-center gap-1 text-xs font-medium text-foreground">
                          <Check className="size-4" aria-hidden="true" /> Seu voto
                        </span>
                      ) : enviando === c.ficha_id ? (
                        <span className="shrink-0 text-xs text-muted-foreground">Votando…</span>
                      ) : null}
                    </button>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}
      </div>
    </section>
  )
}

// Prêmio do público no resultado (aparece com o resultado liberado).
export function PremioPublico({ publico }: { publico: Record<string, LinhaVoto[]> }) {
  const tipos = Object.keys(publico).filter((t) => publico[t].length > 0)
  if (tipos.length === 0) return null
  return (
    <div className="mt-4 border-t border-border pt-4">
      <h3 className="mb-2 font-semibold text-foreground">Prêmio do público</h3>
      {tipos.map((t) => (
        <div key={t} className="mb-3">
          <p className="mb-1 text-sm font-medium text-foreground">{rotuloTipo[t] ?? t}</p>
          {publico[t].map((l) => (
            <div key={l.ficha_id} className="flex items-center justify-between border-b border-border py-1 text-sm">
              <span className="text-foreground">
                {l.posicao}º ·{' '}
                {l.perfil_slug ? (
                  <Link to={`/t/${l.perfil_slug}`} className="underline underline-offset-4">
                    {l.nome}
                  </Link>
                ) : (
                  l.nome
                )}
              </span>
              <span className="tabular-nums text-muted-foreground">
                {l.votos} {l.votos === 1 ? 'voto' : 'votos'}
              </span>
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}

type CriterioDevolutiva = {
  nome: string
  media: number
  nota_min: number
  nota_max: number
  jurados: number
  comentarios: { jurado: string; texto: string }[]
}
type Devolutiva = {
  ficha_id: number
  tipo: string
  personagem?: string
  posicao?: number
  nota_final?: number
  criterios: CriterioDevolutiva[]
}

// Devolutiva dos jurados para o participante (só a dele).
export function MinhaDevolutiva({ slug }: { slug: string }) {
  const { usuario } = useAuth()
  const { data } = useQuery({
    queryKey: ['minha-devolutiva', slug, usuario?.id],
    enabled: !!usuario,
    retry: false,
    queryFn: async () => {
      try {
        return await api<Devolutiva[]>(`/eventos/${slug}/minha-devolutiva`)
      } catch {
        return null
      }
    },
  })
  if (!data || data.length === 0) return null
  return (
    <section aria-labelledby="titulo-devolutiva" className="mt-8 rounded-xl bg-muted p-4 sm:p-5">
      <h2 id="titulo-devolutiva" className="text-xl font-semibold text-foreground">
        Sua devolutiva dos jurados
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">Só você vê. Os jurados aparecem numerados, sem nome.</p>
      {data.map((d) => (
        <div key={d.ficha_id} className="mt-4">
          <p className="font-medium text-foreground">
            {d.personagem || rotuloTipo[d.tipo] || 'Sua apresentação'}
            {d.posicao ? ` · ${d.posicao}º lugar` : ''}
            {d.nota_final ? ` · nota final ${d.nota_final.toFixed(2)}` : ''}
          </p>
          <ul className="mt-2 flex flex-col gap-3">
            {d.criterios.map((c) => (
              <li key={c.nome} className="rounded-lg bg-background p-3">
                <div className="flex items-baseline justify-between gap-2">
                  <span className="font-medium text-foreground">{c.nome}</span>
                  <span className="tabular-nums text-foreground">
                    {c.media.toLocaleString('pt-BR')} <span className="text-xs text-muted-foreground">de {c.nota_max}</span>
                  </span>
                </div>
                {c.comentarios.length > 0 && (
                  <ul className="mt-2 flex flex-col gap-1 text-sm">
                    {c.comentarios.map((x, i) => (
                      <li key={i} className="text-foreground">
                        <span className="text-muted-foreground">{x.jurado}:</span> {x.texto}
                      </li>
                    ))}
                  </ul>
                )}
              </li>
            ))}
          </ul>
        </div>
      ))}
    </section>
  )
}

type Situacao = {
  aberta: boolean
  aberta_em: string | null
  fechada_em: string | null
  devolutiva_liberada: boolean
  contagem: Record<string, LinhaVoto[]>
}

// Painel do organizador: abrir/fechar a votação, ver a parcial e liberar a devolutiva.
export function VotacaoCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const chave = ['org-votacao', eventoId]
  const { data } = useQuery({
    queryKey: chave,
    queryFn: () => api<Situacao>(`/org/eventos/${eventoId}/votacao`),
    refetchInterval: 15_000,
  })
  const [enviando, setEnviando] = useState(false)
  const acao = async (caminho: string, body: object) => {
    setEnviando(true)
    try {
      await api(`/org/eventos/${eventoId}/${caminho}`, { method: 'POST', body })
      queryClient.invalidateQueries({ queryKey: chave })
    } finally {
      setEnviando(false)
    }
  }
  if (!data) return null
  const total = Object.values(data.contagem).reduce((s, ls) => s + ls.reduce((a, l) => a + l.votos, 0), 0)
  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Voto do público e devolutiva</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <p className="text-sm text-muted-foreground">
            Abra a votação quando a plateia já tiver visto as apresentações. Só quem passou pelo check-in vota (um voto por
            ingresso, por categoria). O público não vê a contagem; o "Prêmio do público" sai junto com o resultado, depois que
            você fecha a votação.
          </p>
          <p role="status" className="text-sm font-medium text-foreground">
            {data.aberta ? 'Votação aberta' : data.fechada_em ? 'Votação encerrada' : 'Votação ainda não aberta'} · {total}{' '}
            {total === 1 ? 'voto' : 'votos'}
          </p>
          <div>
            <Button
              type="button"
              variant={data.aberta ? 'outline' : 'default'}
              disabled={enviando}
              onClick={() => acao('votacao', { aberta: !data.aberta })}
            >
              {data.aberta ? 'Encerrar votação' : data.fechada_em ? 'Reabrir votação' : 'Abrir votação'}
            </Button>
          </div>
          {total > 0 && (
            <div className="mt-2">
              <p className="mb-1 text-xs text-muted-foreground">Parcial (só você vê):</p>
              <PremioPublico publico={data.contagem} />
            </div>
          )}
        </div>
        <div className="flex flex-col gap-2 border-t border-border pt-4">
          <p className="text-sm text-muted-foreground">
            A devolutiva mostra a cada participante só a dele: a média de cada critério e os comentários dos jurados, numerados
            (sem nome). Libere depois do resultado.
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              variant="outline"
              disabled={enviando}
              onClick={() => acao('devolutiva', { liberar: !data.devolutiva_liberada })}
            >
              {data.devolutiva_liberada ? 'Recolher devolutiva' : 'Liberar devolutiva'}
            </Button>
            <span className="text-sm text-foreground">
              {data.devolutiva_liberada ? 'Liberada para os participantes.' : 'Ainda não liberada.'}
            </span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
