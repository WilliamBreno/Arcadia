import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '@/lib/api'
import {
  circuitoDoEvento,
  circuitosAbertos,
  criarSeletiva,
  dataCurta,
  decidirSeletiva,
  entrarNoCircuito,
  removerSeletiva,
  sairDoCircuito,
} from '@/lib/circuito'

const select = 'h-8 rounded-lg border border-border bg-background px-2.5 text-sm'

// Cartão do painel do evento (item 8.5): etapa de circuito e seletivas.
export function CircuitoCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const chave = ['org-circuito-evento', eventoId]
  const { data } = useQuery({ queryKey: chave, queryFn: () => circuitoDoEvento(eventoId) })
  const { data: abertos } = useQuery({ queryKey: ['org-circuitos-abertos'], queryFn: circuitosAbertos, enabled: !!data && !data.circuito })
  const [escolhido, setEscolhido] = useState('')
  const [seletiva, setSeletiva] = useState({ slug: '', vagas: '3' })
  const [erro, setErro] = useState<string | null>(null)

  const rodar = async (f: () => Promise<unknown>) => {
    setErro(null)
    try {
      await f()
      await queryClient.invalidateQueries({ queryKey: chave })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir agora.')
    }
  }
  if (!data) return null
  const pendentesParaMim = (data.classifica_para ?? []).filter((s) => s.status === 'pendente')

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Circuito e seletivas</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <section aria-labelledby="t-circuito" className="flex flex-col gap-2">
          <h3 id="t-circuito" className="text-sm font-medium text-foreground">
            Etapa de circuito
          </h3>
          {data.circuito ? (
            <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2">
              <p className="text-sm text-foreground">
                <Link to={`/circuitos/${data.circuito.slug}`} className="underline underline-offset-4">
                  {data.circuito.nome}
                </Link>{' '}
                <span className="text-muted-foreground">
                  · {data.circuito.status === 'aceito' ? 'etapa confirmada' : 'aguardando o dono do circuito aceitar'}
                </span>
              </p>
              <Button variant="ghost" size="sm" onClick={() => confirm('Tirar o evento do circuito? Os pontos desta etapa deixam de contar.') && rodar(() => sairDoCircuito(eventoId))}>
                Sair do circuito
              </Button>
            </div>
          ) : (
            <>
              <p className="text-xs text-muted-foreground">
                Os 10 primeiros de cada categoria somam pontos no ranking do circuito quando você libera o resultado. A data do evento precisa estar
                dentro da temporada.{' '}
                <Link to="/organizador/circuitos" className="underline underline-offset-4">
                  Criar um circuito
                </Link>
              </p>
              <div className="flex flex-wrap gap-2">
                <label htmlFor="circuito-escolha" className="sr-only">
                  Circuito
                </label>
                <select id="circuito-escolha" className={`${select} min-w-0 flex-1`} value={escolhido} onChange={(e) => setEscolhido(e.target.value)}>
                  <option value="">Escolha um circuito…</option>
                  {abertos?.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.nome} ({dataCurta(c.temporada_inicio)} a {dataCurta(c.temporada_fim)})
                    </option>
                  ))}
                </select>
                <Button size="sm" disabled={!escolhido} onClick={() => rodar(() => entrarNoCircuito(eventoId, Number(escolhido)))}>
                  Entrar no circuito
                </Button>
              </div>
            </>
          )}
        </section>

        <section aria-labelledby="t-seletivas" className="flex flex-col gap-2">
          <h3 id="t-seletivas" className="text-sm font-medium text-foreground">
            Seletivas desta final
          </h3>
          <p className="text-xs text-muted-foreground">
            Se este evento é uma final, escolha os eventos que classificam para ele. Quando a seletiva libera o resultado, os primeiros de cada categoria
            entram aqui já aprovados (e pagam a inscrição, se você cobrar). Evento de outro organizador precisa aceitar.
          </p>
          {(data.seletivas ?? []).map((s) => (
            <div key={s.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2">
              <p className="text-sm text-foreground">
                <Link to={`/e/${s.slug}`} className="underline-offset-4 hover:underline">
                  {s.titulo}
                </Link>{' '}
                <span className="text-muted-foreground">
                  · top {s.vagas} de cada categoria · {s.status === 'aceita' ? 'confirmada' : 'aguardando o organizador aceitar'}
                </span>
              </p>
              <Button variant="ghost" size="sm" onClick={() => rodar(() => removerSeletiva(eventoId, s.id))}>
                Remover
              </Button>
            </div>
          ))}
          <div className="flex flex-wrap items-end gap-2">
            <div className="flex min-w-48 flex-1 flex-col gap-1.5">
              <Label htmlFor="sel-slug">Endereço do evento seletiva</Label>
              <Input id="sel-slug" placeholder="ex.: concurso-maceio-2027" value={seletiva.slug} onChange={(e) => setSeletiva({ ...seletiva, slug: e.target.value.replace(/^.*\/e\//, '') })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="sel-vagas">Vagas</Label>
              <Input id="sel-vagas" type="number" min="1" max="10" className="w-20" value={seletiva.vagas} onChange={(e) => setSeletiva({ ...seletiva, vagas: e.target.value })} />
            </div>
            <Button
              disabled={!seletiva.slug}
              onClick={() => rodar(async () => {
                await criarSeletiva(eventoId, seletiva.slug.trim(), Number(seletiva.vagas))
                setSeletiva({ slug: '', vagas: '3' })
              })}
            >
              Adicionar seletiva
            </Button>
          </div>
        </section>

        {(data.classifica_para ?? []).length > 0 && (
          <section aria-labelledby="t-classifica" className="flex flex-col gap-2">
            <h3 id="t-classifica" className="text-sm font-medium text-foreground">
              Este evento classifica para
            </h3>
            {(data.classifica_para ?? []).map((s) => (
              <div key={s.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2">
                <p className="text-sm text-foreground">
                  <Link to={`/e/${s.slug}`} className="underline-offset-4 hover:underline">
                    {s.titulo}
                  </Link>{' '}
                  <span className="text-muted-foreground">· top {s.vagas} de cada categoria</span>
                </p>
                {s.status === 'pendente' ? (
                  <div className="flex gap-2">
                    <Button size="sm" onClick={() => rodar(() => decidirSeletiva(eventoId, s.id, true))}>
                      Aceitar
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => rodar(() => decidirSeletiva(eventoId, s.id, false))}>
                      Recusar
                    </Button>
                  </div>
                ) : (
                  <span className="text-xs text-muted-foreground">confirmada</span>
                )}
              </div>
            ))}
            {pendentesParaMim.length > 0 && (
              <p className="text-xs text-muted-foreground">
                Ao aceitar, os classificados (nome, dados da ficha e contato) passam para a inscrição da final, de outro organizador.
              </p>
            )}
          </section>
        )}

        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
