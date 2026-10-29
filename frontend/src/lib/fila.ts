// Lista de espera de ingresso esgotado (item 7.3).
import { api } from '@/lib/api'

export type EntradaFila = {
  tipo_ingresso_id: number
  quantidade: number
  status: 'aguardando' | 'convidado'
  posicao?: number
  prazo_em?: string
}

export const minhaFila = (slug: string) => api<EntradaFila[]>(`/eventos/${slug}/fila`)

export const entrarNaFila = (slug: string, tipoId: number, quantidade: number) =>
  api<EntradaFila[]>(`/eventos/${slug}/fila`, { method: 'POST', body: { tipo_ingresso_id: tipoId, quantidade } })

export const sairDaFila = (slug: string, tipoId: number) =>
  api<EntradaFila[]>(`/eventos/${slug}/fila/${tipoId}`, { method: 'DELETE' })

export const horaPrazo = (iso: string) =>
  new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
