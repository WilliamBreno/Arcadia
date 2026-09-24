import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import type { Pedido } from '@/lib/checkout'
import { formatarCentavos } from '@/lib/evento'
import { estiloTema } from '@/lib/tema'
import type { EventoDetalhe as EventoDetalheTipo } from '@/lib/publico'
import { useAuth } from '@/hooks/use-auth'
import { Button } from '@/components/ui/button'
import { ListaCronograma } from '@/components/cronograma-cards'
import { ListaRanking } from '@/components/resultado-cards'
import { Card, CardContent } from '@/components/ui/card'

export default function EventoDetalhe() {
  const { slug } = useParams()
  const { usuario } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const [quantidades, setQuantidades] = useState<Record<number, number>>({})
  const [garantia, setGarantia] = useState(false)
  const [cupomCodigo, setCupomCodigo] = useState('')
  const [cupom, setCupom] = useState<{ codigo: string; tipo: 'percentual' | 'valor'; valor: number } | null>(null)
  const [erroCupom, setErroCupom] = useState<string | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [comprando, setComprando] = useState(false)

  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const chaveRef = `ref:${slug}`
  useEffect(() => {
    const ref = searchParams.get('ref')
    if (ref) {
      try {
        sessionStorage.setItem(chaveRef, ref)
      } catch {
        // sem sessionStorage: a atribuição só vale se comprar na mesma carga
      }
    }
  }, [searchParams, chaveRef])
  const { data, isLoading, error } = useQuery({
    queryKey: ['evento-publico', slug],
    queryFn: () => api<EventoDetalheTipo>(`/eventos/${slug}`),
  })
  const { data: resultado } = useQuery({
    queryKey: ['resultado', slug],
    retry: false,
    queryFn: async () => {
      try {
        return await api<{ ranking: Record<string, { posicao: number; nome: string; nota_final: number }[]> }>(`/eventos/${slug}/resultado`, { semAuth: true })
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null
        throw e
      }
    },
  })
  const { data: solicitacao } = useQuery({
    queryKey: ['minha-solicitacao', slug],
    enabled: !!usuario && !!data?.evento.aprovacao_manual,
    queryFn: async () => {
      try {
        return await api<{ status: 'pendente' | 'aprovada' | 'rejeitada' | 'lista_espera' }>(`/eventos/${slug}/solicitacao`)
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null
        throw e
      }
    },
  })

  if (isLoading) return <p className="p-8 text-center text-muted-foreground">Carregando…</p>
  if (error instanceof ApiError && error.status === 404) {
    return <p className="p-8 text-center text-muted-foreground">Evento não encontrado.</p>
  }
  if (!data) return null

  const { evento, local, organizador, ingressos } = data

  const precisaAprovacao = evento.aprovacao_manual && solicitacao?.status !== 'aprovada'

  const solicitar = async () => {
    if (!usuario) {
      navigate('/login', { state: { de: location.pathname } })
      return
    }
    setErro(null)
    try {
      await api(`/eventos/${slug}/solicitacao`, { method: 'POST' })
      queryClient.invalidateQueries({ queryKey: ['minha-solicitacao', slug] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao solicitar')
    }
  }

  const totalUnidades = Object.values(quantidades).reduce((a, b) => a + b, 0)
  const totalSelecionado = ingressos.reduce((soma, i) => {
    const qtd = quantidades[i.id] ?? 0
    const desconto = !cupom
      ? 0
      : Math.min(i.preco_centavos, cupom.tipo === 'percentual' ? Math.floor((i.preco_centavos * cupom.valor) / 100) : cupom.valor)
    const porUnidade = i.preco_centavos - desconto + data.taxa_plataforma_centavos + (garantia ? data.garantia_centavos : 0)
    return soma + qtd * porUnidade
  }, 0)

  const aplicarCupom = async () => {
    setErroCupom(null)
    if (!usuario) {
      navigate('/login', { state: { de: location.pathname } })
      return
    }
    try {
      setCupom(await api(`/eventos/${slug}/cupom`, { method: 'POST', body: { codigo: cupomCodigo } }))
    } catch (e) {
      setCupom(null)
      setErroCupom(e instanceof ApiError ? e.message : 'Erro ao validar cupom')
    }
  }

  const refAtual = () => {
    try {
      return searchParams.get('ref') ?? sessionStorage.getItem(chaveRef) ?? ''
    } catch {
      return searchParams.get('ref') ?? ''
    }
  }

  const comprar = async () => {
    if (!usuario) {
      navigate('/login', { state: { de: location.pathname } })
      return
    }
    setErro(null)
    setComprando(true)
    try {
      const itens = Object.entries(quantidades)
        .filter(([, qtd]) => qtd > 0)
        .map(([tipoIngressoId, quantidade]) => ({
          tipo_ingresso_id: Number(tipoIngressoId),
          quantidade,
          garantia_contratada: garantia,
        }))

      const pedido = await api<Pedido>(`/eventos/${slug}/pedidos`, {
        method: 'POST',
        body: { itens, cupom: cupom?.codigo ?? '', ref: refAtual() },
      })
      navigate(`/pedidos/${pedido.id}`)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar pedido')
    } finally {
      setComprando(false)
    }
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12" style={estiloTema(evento.cor_tema)}>
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
        {precisaAprovacao && (
          <div className="mb-3 rounded-lg border border-border p-4 text-sm">
            <p className="font-medium text-foreground">Este evento tem aprovação manual</p>
            {!solicitacao && (
              <>
                <p className="mt-1 text-muted-foreground">Solicite sua participação; depois da aprovação do organizador você poderá comprar.</p>
                <Button className="mt-3" size="sm" onClick={solicitar}>
                  Solicitar participação
                </Button>
              </>
            )}
            {solicitacao?.status === 'pendente' && <p className="mt-1 text-muted-foreground">Solicitação enviada — aguardando o organizador.</p>}
            {solicitacao?.status === 'lista_espera' && (
              <p className="mt-1 text-muted-foreground">Você foi aprovado, mas as vagas acabaram. Está na lista de espera e será avisado por e-mail se abrir uma vaga.</p>
            )}
            {solicitacao?.status === 'rejeitada' && <p className="mt-1 text-destructive">Sua solicitação não foi aprovada.</p>}
          </div>
        )}
        <div className="flex flex-col gap-3">
          {ingressos.length === 0 && <p className="text-sm text-muted-foreground">Nenhum ingresso disponível no momento.</p>}
          {ingressos.map((i) => (
            <Card key={i.id}>
              <CardContent className="flex items-center justify-between py-4">
                <div>
                  <p className="font-medium text-foreground">{i.nome}</p>
                  {data.sessoes.length > 1 && (
                    <p className="text-xs text-muted-foreground">
                      {i.sessao_ids.length === 0
                        ? 'Válido para todos os dias'
                        : `Válido somente: ${i.sessao_ids
                            .map((id) => {
                              const s = data.sessoes.find((x) => x.id === id)
                              return s ? s.titulo || new Date(s.inicio_em).toLocaleDateString('pt-BR') : ''
                            })
                            .join(', ')}`}
                    </p>
                  )}
                  {i.meia_entrada && (
                    <p className="text-xs text-muted-foreground">
                      Meia-entrada (estudante, PcD, jovem de baixa renda): apresente o documento na entrada.
                    </p>
                  )}
                  {i.descricao && <p className="text-sm text-muted-foreground">{i.descricao}</p>}
                  <p className="text-sm font-medium text-foreground">
                    {i.preco_centavos === 0 ? 'Gratuito' : formatarCentavos(i.preco_centavos)}
                    <span className="text-xs font-normal text-muted-foreground"> + taxa</span>
                  </p>
                </div>
                <div className={`flex items-center gap-2 ${precisaAprovacao ? 'hidden' : ''}`}>
                  <Button
                    variant="outline"
                    size="icon-sm"
                    type="button"
                    onClick={() => setQuantidades((q) => ({ ...q, [i.id]: Math.max(0, (q[i.id] ?? 0) - 1) }))}
                  >
                    −
                  </Button>
                  <span className="w-6 text-center text-sm text-foreground">{quantidades[i.id] ?? 0}</span>
                  <Button
                    variant="outline"
                    size="icon-sm"
                    type="button"
                    onClick={() => setQuantidades((q) => ({ ...q, [i.id]: (q[i.id] ?? 0) + 1 }))}
                  >
                    +
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        {totalUnidades > 0 && (
          <div className="mt-4 flex flex-col gap-3 rounded-lg border border-border p-4">
            {evento.garantia_habilitada && (
              <label className="flex items-start gap-2 text-sm text-foreground">
                <input
                  type="checkbox"
                  className="mt-0.5"
                  checked={garantia}
                  onChange={(e) => setGarantia(e.target.checked)}
                />
                <span>
                  Quero <strong>garantia de vaga</strong> (+{formatarCentavos(data.garantia_centavos)} por item) — cancele a
                  qualquer momento até o início do evento e receba tudo de volta.
                </span>
              </label>
            )}
            <div className="flex items-start gap-2">
              <input
                className="h-8 flex-1 rounded-lg border border-border bg-background px-2.5 text-sm uppercase"
                placeholder="Cupom de desconto"
                value={cupomCodigo}
                onChange={(e) => setCupomCodigo(e.target.value)}
              />
              <Button type="button" variant="outline" size="sm" onClick={aplicarCupom} disabled={!cupomCodigo}>
                Aplicar
              </Button>
            </div>
            {cupom && <p className="text-xs text-primary">Cupom {cupom.codigo} aplicado.</p>}
            {erroCupom && <p className="text-xs text-destructive">{erroCupom}</p>}
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">
                  {totalUnidades} item(ns) · preço + {formatarCentavos(data.taxa_plataforma_centavos)} de taxa
                  {garantia && ` + ${formatarCentavos(data.garantia_centavos)} de garantia`}
                </p>
                <p className="font-medium text-foreground">Total: {formatarCentavos(totalSelecionado)}</p>
              </div>
              <Button onClick={comprar} disabled={comprando}>
                {comprando ? 'Processando…' : 'Continuar'}
              </Button>
            </div>
          </div>
        )}
        {erro && <p className="mt-2 text-sm text-destructive">{erro}</p>}
      </section>

      {evento.garantia_habilitada && (
        <p className="mt-6 text-sm text-muted-foreground">
          Este evento oferece <strong>garantia de vaga</strong>: cancele até o início do evento e receba tudo de volta.
        </p>
      )}

      {data.sessoes.filter((s) => s.status === 'ativa').length > 1 && (
        <Card className="mt-6">
          <CardContent className="py-4">
            <h2 className="mb-2 text-lg font-semibold text-foreground">Sessões</h2>
            <ul className="flex flex-col gap-1 text-sm">
              {data.sessoes.map((s) => (
                <li key={s.id} className={s.status === 'cancelada' ? 'text-muted-foreground line-through' : 'text-foreground'}>
                  {s.titulo ? `${s.titulo} — ` : ''}
                  {new Date(s.inicio_em).toLocaleString('pt-BR', { dateStyle: 'full', timeStyle: 'short' })}
                  {s.status === 'cancelada' && ' (cancelada)'}
                </li>
              ))}
            </ul>
            <p className="mt-2 text-xs text-muted-foreground">Um ingresso comum vale para todas as sessões.</p>
          </CardContent>
        </Card>
      )}

      {data.cronograma.length > 0 && (
        <Card className="mt-6">
          <CardContent className="py-4">
            <h2 className="mb-2 text-lg font-semibold text-foreground">Cronograma</h2>
            <ListaCronograma itens={data.cronograma} />
          </CardContent>
        </Card>
      )}

      {resultado && Object.keys(resultado.ranking).length > 0 && (
        <Card className="mt-6">
          <CardContent className="py-4">
            <h2 className="mb-2 text-lg font-semibold text-foreground">Resultado do concurso</h2>
            <ListaRanking ranking={resultado.ranking} />
          </CardContent>
        </Card>
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
