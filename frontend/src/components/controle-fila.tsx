import { useState } from 'react'

import { ApiError } from '@/lib/api'
import { entrarNaFila, sairDaFila, type EntradaFila } from '@/lib/fila'
import { Button } from '@/components/ui/button'

type Props = {
  slug: string
  tipoId: number
  max: number
  entrada?: EntradaFila
  logado: boolean
  aoMudar: (lista: EntradaFila[]) => void
  pedirLogin: () => void
}

// Ingresso esgotado (item 7.3): entrar na lista de espera, ver a posição e
// sair. Quando uma vaga volta, ela fica guardada para a primeira pessoa.
export function ControleFila({ slug, tipoId, max, entrada, logado, aoMudar, pedirLogin }: Props) {
  const [quantidade, setQuantidade] = useState(1)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const limite = Math.max(1, Math.min(max || 5, 20))

  const agir = async (acao: () => Promise<EntradaFila[]>) => {
    setErro(null)
    setEnviando(true)
    try {
      aoMudar(await acao())
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível agora. Tente de novo.')
    } finally {
      setEnviando(false)
    }
  }

  if (entrada?.status === 'aguardando') {
    return (
      <div className="flex flex-col items-end gap-1 text-right">
        <span className="rounded-full bg-muted px-2.5 py-1 text-xs font-medium text-foreground">
          Na lista · {entrada.posicao}º da fila
        </span>
        <span className="text-xs text-muted-foreground">
          {entrada.quantidade} {entrada.quantidade === 1 ? 'ingresso' : 'ingressos'}
        </span>
        <button
          type="button"
          disabled={enviando}
          onClick={() => agir(() => sairDaFila(slug, tipoId))}
          className="text-xs text-muted-foreground underline underline-offset-4 hover:text-foreground"
        >
          Sair da lista
        </button>
        {erro && <span className="text-xs text-destructive">{erro}</span>}
      </div>
    )
  }

  return (
    <div className="flex flex-col items-end gap-1.5">
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Esgotado</span>
      <div className="flex items-center gap-2">
        <label className="sr-only" htmlFor={`fila-qtd-${tipoId}`}>
          Quantos ingressos
        </label>
        <select
          id={`fila-qtd-${tipoId}`}
          value={quantidade}
          onChange={(e) => setQuantidade(Number(e.target.value))}
          className="h-8 rounded-lg border border-border bg-background px-2 text-sm text-foreground"
        >
          {Array.from({ length: limite }, (_, k) => k + 1).map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </select>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={enviando}
          onClick={() => (logado ? agir(() => entrarNaFila(slug, tipoId, quantidade)) : pedirLogin())}
        >
          {enviando ? 'Entrando…' : 'Entrar na lista de espera'}
        </Button>
      </div>
      {erro && <span className="max-w-xs text-right text-xs text-destructive">{erro}</span>}
    </div>
  )
}
