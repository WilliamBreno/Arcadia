import { api } from '@/lib/api'

export type FinanceiroEvento = {
  bruto_centavos: number
  taxa_processador_centavos: number
  liquido_centavos: number
  liberar_em: string | null
  repasse_status: 'calculado' | 'pendente' | 'pago' | 'cancelado' | null
}

export type Repasse = {
  id: number
  evento_id: number
  evento_titulo?: string
  organizador_nome?: string
  chave_pix?: string
  tipo_chave_pix?: string
  valor_bruto_centavos: number
  taxa_processador_centavos: number
  valor_liquido_centavos: number
  status: 'calculado' | 'pendente' | 'pago' | 'cancelado'
  liberar_em: string
  pago_em: string | null
  comprovante_url: string
  observacao: string
}

export const obterFinanceiroEvento = (eventoId: number) => api<FinanceiroEvento>(`/org/eventos/${eventoId}/financeiro`)
export const obterExtrato = () => api<{ a_receber_centavos: number; repasses: Repasse[] }>('/org/repasses')
export const listarRepassesAdmin = (status: string) => api<Repasse[]>(`/admin/repasses${status ? `?status=${status}` : ''}`)
export const pagarRepasse = (id: number, comprovante_url: string, observacao: string) =>
  api(`/admin/repasses/${id}/pagar`, { method: 'POST', body: { comprovante_url, observacao } })
