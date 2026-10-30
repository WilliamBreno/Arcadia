import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import {
  desistir,
  obterMinhaFicha,
  obterValoresInscricao,
  pagarInscricao,
  simularDesistencia,
  type DecisaoCancelamento,
  type MinhaFicha,
  type ValoresInscricao,
} from '@/lib/inscricao'

type EventoInscricao = {
  slug: string
  modo_participantes: string
  inscricao_talento_centavos?: number
  timezone?: string
}

const caixa = 'mt-6 scroll-mt-20 rounded-xl border border-border p-4 sm:p-5'

function quando(iso: string | null | undefined, fuso?: string) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short', timeZone: fuso || 'America/Maceio' })
}

// Bloco "Sua inscrição" na página do evento (itens 1.6 e 8.4): convida a se
// inscrever, mostra a situação e, na inscrição paga, cobra depois da
// aprovação e permite desistir pelas regras da garantia.
export function MinhaInscricao({ evento }: { evento: EventoInscricao }) {
  const { usuario } = useAuth()
  const aberta = evento.modo_participantes === 'inscricao_aberta' || evento.modo_participantes === 'ambos'
  const valor = evento.inscricao_talento_centavos ?? 0

  const { data: ficha, isLoading } = useQuery({
    queryKey: ['minha-ficha', evento.slug, usuario?.id],
    enabled: !!usuario,
    retry: false,
    queryFn: () => obterMinhaFicha(evento.slug).catch((e) => (e instanceof ApiError && e.status === 404 ? null : Promise.reject(e))),
  })

  if (usuario && isLoading) return null
  if (!ficha) {
    if (!aberta) return null
    return (
      <div className={`${caixa} flex flex-wrap items-center justify-between gap-3`}>
        <div>
          <p className="text-sm text-foreground">Quer participar do concurso deste evento?</p>
          {valor > 0 && (
            <p className="mt-1 text-xs text-muted-foreground">
              Inscrição de {formatarCentavos(valor)} + taxa. Você só paga se o organizador aprovar.
            </p>
          )}
        </div>
        <Link to={`/e/${evento.slug}/inscricao`} className="text-sm font-medium text-primary underline-offset-4 hover:underline">
          Inscrever-se
        </Link>
      </div>
    )
  }
  return <SituacaoInscricao evento={evento} ficha={ficha} />
}

function SituacaoInscricao({ evento, ficha }: { evento: EventoInscricao; ficha: MinhaFicha }) {
  const titulo = (
    <h2 id="titulo-minha-inscricao" className="text-xl font-semibold text-foreground">
      Sua inscrição
    </h2>
  )
  const caixaCom = (conteudo: React.ReactNode) => (
    <section id="minha-inscricao" aria-labelledby="titulo-minha-inscricao" className={caixa}>
      {titulo}
      {conteudo}
    </section>
  )

  switch (ficha.status) {
    case 'pendente':
    case 'lista_espera':
      return caixaCom(
        <p className="mt-1 text-sm text-muted-foreground">
          {ficha.status === 'pendente' ? 'Enviada. Aguardando a análise do organizador.' : 'Você está na lista de espera do concurso.'}
          {(evento.inscricao_talento_centavos ?? 0) > 0 && ' Se for aprovada, você recebe um e-mail para pagar a inscrição.'}
        </p>,
      )
    case 'aguardando_pagamento':
      return caixaCom(<PagarInscricao evento={evento} />)
    case 'aprovado':
      return caixaCom(<InscricaoAprovada slug={evento.slug} />)
    case 'rejeitado':
      return caixaCom(
        <p className="mt-1 text-sm text-muted-foreground">
          Sua inscrição não foi aprovada{ficha.motivo_rejeicao ? `: ${ficha.motivo_rejeicao}` : '.'}
        </p>,
      )
    case 'desistiu':
      return caixaCom(<p className="mt-1 text-sm text-muted-foreground">Você desistiu da participação.</p>)
    default:
      return null
  }
}

function PagarInscricao({ evento }: { evento: EventoInscricao }) {
  const { data: v } = useQuery({ queryKey: ['valores-inscricao', evento.slug], queryFn: () => obterValoresInscricao(evento.slug) })
  if (!v) return null
  return (
    <ResumoPagamento
      valores={v}
      rotuloItem="Inscrição"
      aviso={
        <>
          Aprovada! Para confirmar a vaga, pague a inscrição{v.prazo_em ? <> até <strong>{quando(v.prazo_em, evento.timezone)}</strong></> : null}. Sem o
          pagamento, a vaga é liberada.
        </>
      }
      pagar={(g) => pagarInscricao(evento.slug, g)}
    />
  )
}

// Resumo (preço + taxa + garantia opcional) e botão de pagar. Usado na
// inscrição paga (8.4) e no espaço de expositor (8.6).
export function ResumoPagamento({
  valores: v,
  rotuloItem,
  aviso,
  pagar,
}: {
  valores: ValoresInscricao
  rotuloItem: string
  aviso: React.ReactNode
  pagar: (garantia: boolean) => Promise<{ id: number }>
}) {
  const navigate = useNavigate()
  const [garantia, setGarantia] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const total = v.preco_centavos + v.taxa_centavos + (garantia ? v.garantia_centavos : 0)
  const clicar = async () => {
    setErro(null)
    setEnviando(true)
    try {
      const p = await pagar(garantia)
      navigate(`/pedidos/${p.id}`)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível abrir o pagamento.')
      setEnviando(false)
    }
  }

  return (
    <div className="mt-2 flex flex-col gap-3">
      <p className="text-sm text-foreground">{aviso}</p>
      <dl className="flex flex-col gap-1 text-sm">
        <div className="flex justify-between">
          <dt className="text-muted-foreground">{rotuloItem}</dt>
          <dd className="text-foreground">{formatarCentavos(v.preco_centavos)}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-muted-foreground">Taxa de serviço</dt>
          <dd className="text-foreground">{formatarCentavos(v.taxa_centavos)}</dd>
        </div>
        {garantia && (
          <div className="flex justify-between">
            <dt className="text-muted-foreground">Garantia</dt>
            <dd className="text-foreground">{formatarCentavos(v.garantia_centavos)}</dd>
          </div>
        )}
        <div className="flex justify-between border-t border-border pt-2 font-semibold">
          <dt className="text-foreground">Total</dt>
          <dd className="text-foreground">{formatarCentavos(total)}</dd>
        </div>
      </dl>
      {v.garantia_disponivel && (
        <label className="flex items-start gap-2 text-sm text-foreground">
          <input type="checkbox" className="mt-0.5" checked={garantia} onChange={(e) => setGarantia(e.target.checked)} />
          <span>
            Quero a <strong>garantia</strong> (+{formatarCentavos(v.garantia_centavos)}): se eu desistir até o início do evento, recebo tudo de volta. Sem ela,
            só dá para desistir com reembolso em até 7 dias do pagamento.
          </span>
        </label>
      )}
      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={clicar} disabled={enviando}>
          {enviando ? 'Abrindo…' : `Pagar ${formatarCentavos(total)}`}
        </Button>
        {v.pedido_aberto_id && (
          <Link to={`/pedidos/${v.pedido_aberto_id}`} className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground">
            Continuar o pagamento já iniciado
          </Link>
        )}
      </div>
    </div>
  )
}

function InscricaoAprovada({ slug }: { slug: string }) {
  const queryClient = useQueryClient()
  const { data: v } = useQuery({
    queryKey: ['valores-inscricao', slug],
    retry: false,
    queryFn: () => obterValoresInscricao(slug).catch(() => null),
  })
  const pago = v?.pago && v.item_pago_id
  return (
    <div className="mt-1 flex flex-col gap-3">
      <p className="text-sm text-foreground">
        Aprovada{pago ? <> e paga ({formatarCentavos(v.total_pago_centavos)})</> : null}. Sua vaga no concurso está confirmada.
        {pago && v.garantia_contratada && ' Com garantia: se desistir até o início do evento, recebe tudo de volta.'}
      </p>
      {pago && v.item_pago_id && (
        <Desistencia
          itemId={v.item_pago_id}
          rotulo="Desistir da participação"
          manter="Manter inscrição"
          aoConcluir={() =>
            Promise.all([
              queryClient.invalidateQueries({ queryKey: ['minha-ficha', slug] }),
              queryClient.invalidateQueries({ queryKey: ['meu-qr-participante', slug] }),
              queryClient.invalidateQueries({ queryKey: ['valores-inscricao', slug] }),
            ])
          }
        />
      )}
    </div>
  )
}

// Desistir com a simulação do reembolso antes (regras da garantia/arrependimento).
export function Desistencia({
  itemId,
  rotulo,
  manter,
  aoConcluir,
}: {
  itemId: number
  rotulo: string
  manter: string
  aoConcluir: () => Promise<unknown>
}) {
  const [decisao, setDecisao] = useState<DecisaoCancelamento | null>(null)
  const [ocupado, setOcupado] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const verificar = async () => {
    setErro(null)
    setOcupado(true)
    try {
      setDecisao(await simularDesistencia(itemId))
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível verificar agora.')
    } finally {
      setOcupado(false)
    }
  }
  const confirmar = async () => {
    setErro(null)
    setOcupado(true)
    try {
      await desistir(itemId)
      await aoConcluir()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível desistir agora.')
      setOcupado(false)
    }
  }
  return (
    <div className="flex flex-col gap-2">
      {!decisao && (
        <div>
          <Button variant="outline" size="sm" onClick={verificar} disabled={ocupado}>
            {rotulo}
          </Button>
        </div>
      )}
      {decisao && (
        <div className="flex flex-col gap-2 rounded-lg border border-border p-3 text-sm">
          {decisao.pode ? (
            <>
              <p className="text-foreground">
                {decisao.valor_reembolso_centavos > 0
                  ? `Você recebe de volta ${formatarCentavos(decisao.valor_reembolso_centavos)} pelo mesmo meio de pagamento.`
                  : 'Não há valor a devolver.'}{' '}
                A vaga é liberada e não dá para desfazer.
              </p>
              <div className="flex flex-wrap gap-2">
                <Button variant="destructive" size="sm" onClick={confirmar} disabled={ocupado}>
                  Confirmar desistência
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setDecisao(null)} disabled={ocupado}>
                  {manter}
                </Button>
              </div>
            </>
          ) : (
            <p className="text-muted-foreground">Não dá para desistir com reembolso agora: {decisao.motivo}.</p>
          )}
        </div>
      )}
      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
    </div>
  )
}
