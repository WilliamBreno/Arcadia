// Modo palco/telão e camarim (item 8.3).
import { api } from '@/lib/api'

export type StatusPalco = 'aguardando' | 'presente' | 'no_palco' | 'apresentou' | 'ausente'
export type ModoTelao = 'intervalo' | 'apresentacao' | 'resultado'

export type ItemFila = {
  ficha_id: number
  ordem: number | null
  nome: string
  personagem?: string
  obra?: string
  tipo: string
  status: StatusPalco
  menor: boolean
  foto_referencia?: string
  foto_participante?: string
}

export type EstadoPalco = { atual_id: number | null; modo: ModoTelao; telao_ativo: boolean; fila: ItemFila[] }

export type EstadoTelao = {
  evento: string
  modo: ModoTelao
  atual?: ItemFila
  proximos: { nome: string; personagem?: string }[]
  resultado?: Record<string, { posicao: number; nome: string }[]>
  publico?: Record<string, { posicao: number; nome: string; votos: number }[]>
}

export const rotuloStatus: Record<StatusPalco, string> = {
  aguardando: 'Aguardando',
  presente: 'No camarim',
  no_palco: 'No palco',
  apresentou: 'Apresentou',
  ausente: 'Faltou',
}

export const rotuloTipo: Record<string, string> = { cosplay: 'Cosplay', danca: 'Dança', canto: 'Canto', atuacao: 'Atuação' }

const base = (id: string | number) => `/palco/eventos/${id}`
export const obterPalco = (id: string) => api<EstadoPalco>(base(id))
export const marcarStatus = (id: string, ficha: number, status: StatusPalco) =>
  api<EstadoPalco>(`${base(id)}/fichas/${ficha}/status`, { method: 'POST', body: { status } })
export const lerQRCamarim = (id: string, payload: string) =>
  api<{ participante: ItemFila; chegou_agora: boolean }>(`${base(id)}/camarim/qr`, { method: 'POST', body: { payload } })
export const chamar = (id: string, ficha: number) =>
  api<EstadoPalco>(`${base(id)}/chamar`, { method: 'POST', body: { ficha_id: ficha } })
export const proximo = (id: string) => api<EstadoPalco>(`${base(id)}/proximo`, { method: 'POST' })
export const encerrar = (id: string) => api<EstadoPalco>(`${base(id)}/encerrar`, { method: 'POST' })
export const definirModo = (id: string, modo: ModoTelao) =>
  api<EstadoPalco>(`${base(id)}/modo`, { method: 'POST', body: { modo } })
export const gerarLinkTelao = (id: string) => api<{ token: string }>(`/org/eventos/${id}/telao/link`, { method: 'POST' })
export const obterTelao = (token: string) => api<EstadoTelao>(`/telao/${token}`, { semAuth: true })
