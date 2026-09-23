import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import type { EventoDetalhe as EventoDetalheTipo } from '@/lib/publico'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export default function EventoDetalhe() {
  const { slug } = useParams()
  const { data, isLoading, error } = useQuery({
    queryKey: ['evento-publico', slug],
    queryFn: () => api<EventoDetalheTipo>(`/eventos/${slug}`),
  })

  if (isLoading) return <p className="p-8 text-center text-muted-foreground">Carregando…</p>
  if (error instanceof ApiError && error.status === 404) {
    return <p className="p-8 text-center text-muted-foreground">Evento não encontrado.</p>
  }
  if (!data) return null

  const { evento, local, organizador, ingressos } = data

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      {evento.capa_url && (
        <img src={evento.capa_url} alt="" className="mb-6 aspect-video w-full rounded-xl object-cover" />
      )}

      <p className="text-sm font-medium text-primary">
        {evento.inicio_em && new Date(evento.inicio_em).toLocaleString('pt-BR', { dateStyle: 'full', timeStyle: 'short' })}
      </p>
      <h1 className="mt-1 text-3xl font-semibold text-foreground">{evento.titulo}</h1>

      {organizador && (
        <Link to={`/o/${organizador.slug}`} className="mt-2 inline-block text-sm text-muted-foreground hover:text-foreground">
          por {organizador.nome_publico}
        </Link>
      )}

      {evento.descricao && <p className="mt-6 whitespace-pre-wrap text-foreground">{evento.descricao}</p>}

      {local && (
        <Card className="mt-6">
          <CardContent className="py-4">
            <p className="font-medium text-foreground">{local.nome}</p>
            <p className="text-sm text-muted-foreground">
              {local.logradouro}
              {local.numero ? `, ${local.numero}` : ''} — {local.bairro} · {local.cidade}/{local.uf}
            </p>
          </CardContent>
        </Card>
      )}

      <section className="mt-8">
        <h2 className="mb-3 text-xl font-semibold text-foreground">Ingressos</h2>
        <div className="flex flex-col gap-3">
          {ingressos.length === 0 && <p className="text-sm text-muted-foreground">Nenhum ingresso disponível no momento.</p>}
          {ingressos.map((i) => (
            <Card key={i.id}>
              <CardContent className="flex items-center justify-between py-4">
                <div>
                  <p className="font-medium text-foreground">{i.nome}</p>
                  {i.descricao && <p className="text-sm text-muted-foreground">{i.descricao}</p>}
                </div>
                <div className="flex items-center gap-3">
                  <span className="font-medium text-foreground">
                    {i.preco_centavos === 0 ? 'Gratuito' : formatarCentavos(i.preco_centavos)}
                  </span>
                  <Button size="sm" disabled>
                    Em breve
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          Checkout ainda não está disponível — chega no item 1.7 do plano.
        </p>
      </section>

      {evento.garantia_habilitada && (
        <p className="mt-6 text-sm text-muted-foreground">
          Este evento oferece <strong>garantia de vaga</strong>: cancele até o início do evento e receba tudo de volta.
        </p>
      )}

      {(evento.modo_participantes === 'inscricao_aberta' || evento.modo_participantes === 'ambos') && (
        <div className="mt-6 flex items-center justify-between rounded-lg border border-border p-4">
          <p className="text-sm text-foreground">Quer participar do concurso deste evento?</p>
          <Link to={`/e/${evento.slug}/inscricao`} className="text-sm font-medium text-primary underline-offset-4 hover:underline">
            Inscrever-se
          </Link>
        </div>
      )}

      <p className="mt-4 text-center">
        <Link to={`/e/${evento.slug}/jurado`} className="text-xs text-muted-foreground hover:text-foreground">
          Área do jurado
        </Link>
      </p>
    </main>
  )
}
