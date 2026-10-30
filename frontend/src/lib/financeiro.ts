import { api } from '@/lib/api'

export type FinanceiroEvento = {
  bruto_centavos: number
  taxa_processador_centavos: number
  taxa_absorvida_centavos: number
  reserva_centavos: number
  liquido_centavos: number
  liberar_em: string | null
  repasse_status: 'calculado' | 'pendente' | 'pago' | 'cancelado' | null
  metodos: {
    pix_quantidade: number
    pix_valor_centavos: number
    pix_taxa_centavos: number
    cartao_quantidade: number
    cartao_valor_centavos: number
    cartao_taxa_centavos: number
    taxa_cartao_bp: number
    taxa_cartao_estimada: boolean
    economia_pix_centavos: number
  } | null
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
  taxa_absorvida_centavos: number
  valor_liquido_centavos: number
  valor_a_pagar_agora_centavos: number
  reserva_centavos: number
  reserva_status: 'nenhuma' | 'retida' | 'paga' | 'usada'
  reserva_liberar_em: string | null
  reserva_pago_em: string | null
  reserva_observacao: string
  status: 'calculado' | 'pendente' | 'pago' | 'cancelado'
  liberar_em: string
  pago_em: string | null
  comprovante_url: string
  observacao: string
  // Item 9.1: Pix automático pelo Mercado Pago.
  envio_status: 'manual' | 'na_fila' | 'aguardando_aprovacao' | 'enviando' | 'processando' | 'enviado' | 'falhou'
  envio_rotulo: string
  envio_motivo?: string
}

export const obterFinanceiroEvento = (eventoId: number) => api<FinanceiroEvento>(`/org/eventos/${eventoId}/financeiro`)
export const obterExtrato = () =>
  api<{ a_receber_centavos: number; reserva_retida_centavos: number; repasses: Repasse[] }>('/org/repasses')
export const pagarReserva = (id: number, comprovante_url: string) =>
  api(`/admin/repasses/${id}/reserva/pagar`, { method: 'POST', body: { comprovante_url } })
export const usarReserva = (id: number, observacao: string) =>
  api(`/admin/repasses/${id}/reserva/usar`, { method: 'POST', body: { observacao } })
export const listarRepassesAdmin = (status: string) => api<Repasse[]>(`/admin/repasses${status ? `?status=${status}` : ''}`)
export const pagarRepasse = (id: number, comprovante_url: string, observacao: string) =>
  api(`/admin/repasses/${id}/pagar`, { method: 'POST', body: { comprovante_url, observacao } })

// Item 9.1: o admin aprova e o Pix sai pelo Mercado Pago.
export const enviarPixRepasse = (id: number) => api<{ envio_status: string }>(`/admin/repasses/${id}/enviar`, { method: 'POST' })
