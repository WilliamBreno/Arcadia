// Inscrição de participante e inscrição paga (itens 1.6 e 8.4).
import { api } from '@/lib/api'

export type StatusFicha = 'rascunho' | 'pendente' | 'aprovado' | 'rejeitado' | 'lista_espera' | 'desistiu' | 'aguardando_pagamento'

export type MinhaFicha = {
  id: number
  status: StatusFicha
  motivo_rejeicao?: string
  prazo_pagamento_em?: string
}

export type ValoresInscricao = {
  preco_centavos: number
  taxa_centavos: number
  garantia_centavos: number
  garantia_disponivel: boolean
  prazo_em: string | null
  pedido_aberto_id: number | null
  pago: boolean
  item_pago_id: number | null
  total_pago_centavos: number
  garantia_contratada: boolean
}

export type DecisaoCancelamento = { pode: boolean; motivo: string; valor_reembolso_centavos: number }

export const obterMinhaFicha = (slug: string) => api<MinhaFicha>(`/eventos/${slug}/minha-ficha?papel=participante`)
export const obterValoresInscricao = (slug: string) => api<ValoresInscricao>(`/eventos/${slug}/inscricao/pagamento`)
export const pagarInscricao = (slug: string, garantia: boolean) =>
  api<{ id: number }>(`/eventos/${slug}/inscricao/pagamento`, { method: 'POST', body: { garantia_contratada: garantia } })
export const simularDesistencia = (itemId: number) => api<DecisaoCancelamento>(`/itens/${itemId}/cancelamento`)
export const desistir = (itemId: number) => api(`/itens/${itemId}/cancelar`, { method: 'POST' })
