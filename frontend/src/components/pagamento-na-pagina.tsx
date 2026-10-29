import { useQuery } from '@tanstack/react-query'
import { Check, Copy, CreditCard, QrCode } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { lazy, Suspense, useEffect, useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { gerarPix, obterConfigPagamento, tempoRestante, verificarPagamento, type PixNaTela } from '@/lib/pagamento'
import { useAuth } from '@/hooks/use-auth'
import { Button } from '@/components/ui/button'

const CartaoMercadoPago = lazy(() => import('@/components/cartao-mercado-pago'))

type Props = { pedidoId: number; totalCentavos: number; aoConfirmar: () => void }

// Escolha do pagamento na página do pedido (item 7.1): Pix com QR na tela
// (padrão) ou cartão no formulário seguro do Mercado Pago. Pagar no site do
// Mercado Pago fica como plano B.
export function PagamentoNaPagina({ pedidoId, totalCentavos, aoConfirmar }: Props) {
  const { usuario } = useAuth()
  const [metodo, setMetodo] = useState<'pix' | 'cartao'>('pix')
  const [cartaoFalhou, setCartaoFalhou] = useState(false)
  const [redirecionando, setRedirecionando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const config = useQuery({ queryKey: ['pagamento-config'], queryFn: obterConfigPagamento, staleTime: 3_600_000 })
  const publicKey = config.data?.public_key ?? ''

  const pagarNoSiteDoMP = async () => {
    setErro(null)
    setRedirecionando(true)
    try {
      const r = await api<{ checkout_url: string }>(`/pedidos/${pedidoId}/pagar`, { method: 'POST', body: { metodo: 'cartao' } })
      window.location.href = r.checkout_url
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível abrir o Mercado Pago.')
      setRedirecionando(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div role="radiogroup" aria-label="Forma de pagamento" className="grid grid-cols-2 gap-2">
        <OpcaoMetodo ativo={metodo === 'pix'} aoEscolher={() => setMetodo('pix')} icone={<QrCode className="size-4" aria-hidden="true" />}>
          Pix
        </OpcaoMetodo>
        <OpcaoMetodo ativo={metodo === 'cartao'} aoEscolher={() => setMetodo('cartao')} icone={<CreditCard className="size-4" aria-hidden="true" />}>
          Cartão
        </OpcaoMetodo>
      </div>

      {metodo === 'pix' && <BlocoPix pedidoId={pedidoId} aoConfirmar={aoConfirmar} />}

      {metodo === 'cartao' &&
        (config.isLoading ? (
          <p className="text-sm text-muted-foreground">Carregando…</p>
        ) : publicKey && !cartaoFalhou ? (
          <Suspense fallback={<p className="text-sm text-muted-foreground">Carregando o formulário seguro…</p>}>
            <CartaoMercadoPago
              publicKey={publicKey}
              pedidoId={pedidoId}
              totalCentavos={totalCentavos}
              email={usuario?.email}
              aoConfirmar={aoConfirmar}
              aoFalhar={() => setCartaoFalhou(true)}
            />
          </Suspense>
        ) : (
          <div className="flex flex-col gap-2">
            {cartaoFalhou && <p className="text-sm text-muted-foreground">O formulário do cartão não carregou aqui.</p>}
            <Button size="lg" onClick={pagarNoSiteDoMP} disabled={redirecionando}>
              {redirecionando ? 'Abrindo o Mercado Pago…' : 'Pagar com cartão no Mercado Pago'}
            </Button>
          </div>
        ))}

      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
      {metodo === 'cartao' && publicKey && !cartaoFalhou && (
        <button
          type="button"
          onClick={pagarNoSiteDoMP}
          disabled={redirecionando}
          className="self-center text-xs text-muted-foreground underline underline-offset-4 hover:text-foreground"
        >
          {redirecionando ? 'Abrindo o Mercado Pago…' : 'Prefere pagar no site do Mercado Pago (cartão ou saldo da conta)?'}
        </button>
      )}
      <p className="text-center text-xs text-muted-foreground">Pagamento processado com segurança pelo Mercado Pago.</p>
    </div>
  )
}

function OpcaoMetodo({ ativo, aoEscolher, icone, children }: { ativo: boolean; aoEscolher: () => void; icone: React.ReactNode; children: React.ReactNode }) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={ativo}
      onClick={aoEscolher}
      className={`flex items-center justify-center gap-2 rounded-lg border px-3 py-3 text-sm font-medium transition-colors active:scale-95 ${
        ativo ? 'border-foreground bg-muted text-foreground ring-1 ring-foreground' : 'border-border text-muted-foreground hover:bg-muted hover:text-foreground'
      }`}
    >
      {icone}
      {children}
    </button>
  )
}

function BlocoPix({ pedidoId, aoConfirmar }: { pedidoId: number; aoConfirmar: () => void }) {
  const [pix, setPix] = useState<PixNaTela | null>(null)
  const [gerando, setGerando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const [copiado, setCopiado] = useState(false)
  const [verificando, setVerificando] = useState(false)
  const [aviso, setAviso] = useState<string | null>(null)
  const [agora, setAgora] = useState(() => Date.now())

  useEffect(() => {
    if (!pix) return
    const t = window.setInterval(() => setAgora(Date.now()), 1000)
    return () => window.clearInterval(t)
  }, [pix])

  const gerar = async () => {
    setErro(null)
    setGerando(true)
    try {
      setPix(await gerarPix(pedidoId))
      // A reserva passa a terminar junto com o QR: atualiza o horário mostrado.
      aoConfirmar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível gerar o Pix. Tente de novo.')
    } finally {
      setGerando(false)
    }
  }

  const copiar = async () => {
    if (!pix) return
    try {
      await navigator.clipboard.writeText(pix.copia_cola)
      setCopiado(true)
      window.setTimeout(() => setCopiado(false), 2500)
    } catch {
      setErro('Não deu para copiar. Selecione o código e copie manualmente.')
    }
  }

  const jaPaguei = async () => {
    setAviso(null)
    setVerificando(true)
    try {
      const p = (await verificarPagamento(pedidoId)) as { status: string }
      if (p.status === 'pago') aoConfirmar()
      else setAviso('Ainda não recebemos. Pode levar alguns segundos depois de pagar; a página atualiza sozinha.')
    } catch (e) {
      setAviso(e instanceof ApiError ? e.message : 'Não foi possível conferir agora.')
    } finally {
      setVerificando(false)
    }
  }

  if (!pix) {
    return (
      <div className="flex flex-col gap-2">
        <Button size="lg" onClick={gerar} disabled={gerando}>
          {gerando ? 'Gerando o Pix…' : 'Gerar QR do Pix'}
        </Button>
        <p className="text-center text-xs text-muted-foreground">Confirma na hora e o ingresso chega em segundos.</p>
        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
      </div>
    )
  }

  const restante = tempoRestante(pix.expira_em, agora)
  const vencido = restante === '00:00'

  if (vencido) {
    return <p className="rounded-lg bg-muted p-4 text-sm text-foreground">Este Pix venceu e a reserva foi liberada. Volte ao evento para comprar de novo.</p>
  }

  return (
    <div className="flex flex-col items-center gap-4 rounded-xl border border-border p-4">
      <div className="rounded-lg bg-white p-3">
        <QRCodeSVG value={pix.copia_cola} size={208} level="M" aria-label="QR Code do Pix" />
      </div>
      <p className="text-sm text-foreground">
        Vale por <span className="font-semibold tabular-nums">{restante}</span>
      </p>
      <ol className="w-full list-decimal pl-5 text-sm text-foreground">
        <li>Abra o app do seu banco e escolha Pix.</li>
        <li>Leia o QR Code ou use o código "copia e cola".</li>
        <li>Pague: o ingresso aparece aqui na hora.</li>
      </ol>
      <div className="flex w-full flex-col gap-2">
        <label htmlFor="pix-copia-cola" className="text-xs text-muted-foreground">
          Pix copia e cola
        </label>
        <input
          id="pix-copia-cola"
          readOnly
          value={pix.copia_cola}
          onFocus={(e) => e.currentTarget.select()}
          className="w-full truncate rounded-md border border-border bg-muted px-3 py-2 font-mono text-xs text-foreground"
        />
        <Button variant="outline" onClick={copiar}>
          {copiado ? <Check className="size-4" aria-hidden="true" /> : <Copy className="size-4" aria-hidden="true" />}
          {copiado ? 'Código copiado' : 'Copiar código'}
        </Button>
      </div>
      <p className="flex items-center gap-2 text-sm text-muted-foreground" aria-live="polite">
        <span className="size-2 animate-pulse rounded-full bg-primary" aria-hidden="true" />
        Aguardando o pagamento…
      </p>
      <button type="button" onClick={jaPaguei} disabled={verificando} className="text-sm text-foreground underline underline-offset-4">
        {verificando ? 'Conferindo…' : 'Já paguei'}
      </button>
      {aviso && <p className="text-center text-xs text-muted-foreground">{aviso}</p>}
      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
    </div>
  )
}
