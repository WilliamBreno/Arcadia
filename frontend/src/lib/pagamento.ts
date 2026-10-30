// Pagamento dentro da página do Evve (item 7.1).
import { api } from '@/lib/api'

export type PixNaTela = { copia_cola: string; expira_em: string }

export type ResultadoCartao = {
  situacao: 'aprovado' | 'recusado' | 'desafio' | 'em_analise'
  mensagem?: string
  pagamento_id?: string
  desafio_url?: string
  desafio_creq?: string
}

export type DadosCartao = {
  token: string
  payment_method_id: string
  installments?: number
  issuer_id?: string
  documento_tipo?: string
  documento_numero?: string
}

// Chave pública do formulário de cartão. Vazia: o cartão abre no site do
// Mercado Pago (plano B); o Pix aparece na página de qualquer jeito.
export const obterConfigPagamento = () => api<{ public_key: string }>('/pagamento/config', { semAuth: true })

export const gerarPix = (pedidoId: number) => api<PixNaTela>(`/pedidos/${pedidoId}/pix`, { method: 'POST' })

export const pagarCartao = (pedidoId: number, dados: DadosCartao) =>
  api<ResultadoCartao>(`/pedidos/${pedidoId}/cartao`, { method: 'POST', body: dados })

export const verificarPagamento = (pedidoId: number, pagamentoId?: string) =>
  api(`/pedidos/${pedidoId}/verificar`, { method: 'POST', body: { pagamento_id: pagamentoId ?? '' } })

// "mm:ss" que falta até o fim do Pix.
export function tempoRestante(expiraEm: string, agora: number) {
  const s = Math.max(0, Math.floor((new Date(expiraEm).getTime() - agora) / 1000))
  return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`
}
