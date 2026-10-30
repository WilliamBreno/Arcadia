import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Share2 } from 'lucide-react'
import { useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { Button, buttonVariants } from '@/components/ui/button'
import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import type { Pedido } from '@/lib/checkout'
import { formatarCentavos } from '@/lib/evento'
import type { Grupo as GrupoTipo } from '@/lib/extras'

// Item 10.3: página do link do grupo. Cada pessoa paga a sua parte.
export default function Grupo() {
  const { token = '' } = useParams()
  const { usuario } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({
    queryKey: ['grupo', token, usuario?.id],
    queryFn: () => api<GrupoTipo>(`/grupos/${token}`),
    refetchInterval: 15_000,
  })
  const [erro, setErro] = useState<string | null>(null)
  const [copiado, setCopiado] = useState(false)
  const [pagando, setPagando] = useState(false)

  if (isLoading) return <Carregando />
  if (!data) {
    return (
      <main className="mx-auto max-w-md px-4 py-16 text-center">
        <h1 className="text-xl font-semibold text-foreground">Grupo não encontrado</h1>
        <p className="mt-2 text-muted-foreground">Confira o link com quem criou o grupo.</p>
      </main>
    )
  }

  const link = `${window.location.origin}/grupo/${token}`
  const copiar = async () => {
    try {
      await navigator.clipboard.writeText(link)
      setCopiado(true)
    } catch {
      setCopiado(false)
    }
  }
  const textoConvite = `Bora no ${data.evento_titulo}? Cada um paga a sua parte (${formatarCentavos(data.total_por_pessoa_centavos)}) por aqui: ${link}`

  const pagarMinhaParte = async () => {
    if (!usuario) {
      navigate('/login', { state: { de: location.pathname } })
      return
    }
    setErro(null)
    setPagando(true)
    try {
      const pedido = await api<Pedido>(`/eventos/${data.evento_slug}/pedidos`, {
        method: 'POST',
        body: { itens: [{ tipo_ingresso_id: data.tipo_ingresso_id, quantidade: 1, grupo_token: token }] },
      })
      navigate(`/pedidos/${pedido.id}`)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível reservar a sua parte.')
      queryClient.invalidateQueries({ queryKey: ['grupo', token] })
    } finally {
      setPagando(false)
    }
  }

  const aberto = data.status === 'aberto'
  const pagasPct = Math.round((data.pagas / data.quantidade) * 100)

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <p className="text-sm font-medium text-primary">Compra em grupo · criada por {data.criador}</p>
      <h1 className="mt-1 text-2xl font-semibold text-foreground">
        <Link to={`/e/${data.evento_slug}`} className="hover:underline">
          {data.evento_titulo}
        </Link>
      </h1>
      {data.evento_inicio_em && (
        <p className="text-muted-foreground">{new Date(data.evento_inicio_em).toLocaleString('pt-BR', { dateStyle: 'full', timeStyle: 'short' })}</p>
      )}

      <section className="mt-6 rounded-xl border border-border p-4">
        <div className="flex items-baseline justify-between gap-3">
          <p className="text-foreground">
            {data.quantidade}× {data.tipo_nome}
          </p>
          <p className="font-semibold tabular-nums text-foreground">{formatarCentavos(data.total_por_pessoa_centavos)} por pessoa</p>
        </div>
        <div className="mt-3 h-2 overflow-hidden rounded-full bg-muted" role="presentation">
          <div className="h-full rounded-full bg-primary" style={{ width: `${pagasPct}%` }} />
        </div>
        <p className="mt-2 text-sm text-muted-foreground" aria-live="polite">
          {data.pagas} de {data.quantidade} pagas
          {data.em_pagamento > 0 && ` · ${data.em_pagamento} pagando agora`}
          {aberto && ` · ${data.restantes} livre(s)`}
        </p>
        <p className="mt-1 text-sm text-muted-foreground">
          {aberto
            ? `As partes não pagas até ${new Date(data.expira_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })} voltam à venda.`
            : data.status === 'cancelado'
              ? 'Quem criou encerrou o grupo. Quem pagou continua com o ingresso.'
              : 'O prazo do grupo terminou. Quem pagou continua com o ingresso; as outras partes voltaram à venda.'}
        </p>
      </section>

      {data.minha_parte === 'pago' && (
        <p role="status" className="mt-4 rounded-lg bg-muted p-3 text-sm text-foreground">
          Sua parte está paga. O ingresso está em{' '}
          <Link to="/meus-ingressos" className="underline underline-offset-4">
            Meus ingressos
          </Link>
          .
        </p>
      )}
      {data.minha_parte === 'reservado' && data.meu_pedido_id && (
        <Link to={`/pedidos/${data.meu_pedido_id}`} className={`${buttonVariants()} mt-4 w-full`}>
          Terminar o pagamento da minha parte
        </Link>
      )}
      {!data.minha_parte && aberto && data.restantes > 0 && (
        <Button className="mt-4 w-full" onClick={pagarMinhaParte} disabled={pagando}>
          {pagando ? 'Reservando…' : usuario ? `Pagar a minha parte (${formatarCentavos(data.total_por_pessoa_centavos)})` : 'Entrar para pagar a minha parte'}
        </Button>
      )}
      {!data.minha_parte && aberto && data.restantes === 0 && (
        <p className="mt-4 text-sm text-muted-foreground">Todas as partes já foram pegas. Se alguém não pagar, a vaga volta para o grupo.</p>
      )}
      {erro && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {erro}
        </p>
      )}

      {aberto && (
        <section aria-labelledby="titulo-convite" className="mt-8">
          <h2 id="titulo-convite" className="font-semibold text-foreground">
            Chame a turma
          </h2>
          <div className="mt-3 flex flex-wrap gap-2">
            <Button variant="outline" size="sm" onClick={copiar}>
              <Copy className="size-4" aria-hidden="true" />
              {copiado ? 'Link copiado' : 'Copiar o link'}
            </Button>
            <a
              className={buttonVariants({ variant: 'outline', size: 'sm' })}
              href={`https://wa.me/?text=${encodeURIComponent(textoConvite)}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              <Share2 className="size-4" aria-hidden="true" />
              Mandar no WhatsApp
            </a>
          </div>
        </section>
      )}

      {data.sou_criador && aberto && (
        <Button
          variant="ghost"
          size="sm"
          className="mt-8"
          onClick={async () => {
            if (!window.confirm('Encerrar o grupo? As partes não pagas voltam à venda.')) return
            try {
              await api(`/grupos/${token}/cancelar`, { method: 'POST' })
              queryClient.invalidateQueries({ queryKey: ['grupo', token] })
            } catch (e) {
              setErro(e instanceof ApiError ? e.message : 'Não foi possível encerrar.')
            }
          }}
        >
          Encerrar o grupo
        </Button>
      )}
    </main>
  )
}
