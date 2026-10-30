// Circuitos, ranking por temporada e seletivas (item 8.5).
import { api } from '@/lib/api'

export type Circuito = {
  id: number
  nome: string
  slug: string
  descricao: string
  temporada_inicio: string
  temporada_fim: string
  status?: 'pendente' | 'aceito'
}

export type LinhaCircuito = { posicao: number; nome: string; pontos: number; vitorias: number; eventos: number; perfil_slug?: string }

export type CircuitoPublico = {
  circuito: Circuito
  etapas: { titulo: string; slug: string; inicio_em: string | null; resultado_liberado: boolean }[]
  ranking: Record<string, LinhaCircuito[]>
  pontos_por_posicao: number[]
}

export type SeletivaInfo = { id: number; vagas: number; status: 'pendente' | 'aceita'; titulo: string; slug: string; inicio_em: string | null }

export type CircuitoDoEvento = { circuito?: Circuito | null; seletivas?: SeletivaInfo[]; classifica_para?: SeletivaInfo[] }

export type MeuCircuito = Circuito & {
  pedidos: { id: number; evento_titulo: string; evento_slug: string; inicio_em: string | null; status: 'pendente' | 'aceito' }[]
}

export const rotuloCategoria: Record<string, string> = { cosplay: 'Cosplay', danca: 'Dança', canto: 'Canto', atuacao: 'Atuação', '': 'Geral' }

// "2027-03-01" -> "01/03/2027" (data pura, sem fuso).
export const dataCurta = (iso: string) => iso.split('-').reverse().join('/')

export const obterCircuito = (slug: string) => api<CircuitoPublico>(`/circuitos/${slug}`, { semAuth: true })
export const circuitoDoEventoPublico = (slug: string) => api<CircuitoDoEvento>(`/eventos/${slug}/circuito`, { semAuth: true })

export const meusCircuitos = () => api<MeuCircuito[]>('/org/circuitos')
export const circuitosAbertos = () => api<Circuito[]>('/org/circuitos/abertos')
export const criarCircuito = (d: { nome: string; descricao: string; temporada_inicio: string; temporada_fim: string }) =>
  api<Circuito>('/org/circuitos', { method: 'POST', body: d })
export const decidirPedidoCircuito = (id: number, aceitar: boolean) =>
  api(`/org/circuitos/pedidos/${id}/decidir`, { method: 'POST', body: { aceitar } })

export const circuitoDoEvento = (eventoId: number) => api<CircuitoDoEvento>(`/org/eventos/${eventoId}/circuito`)
export const entrarNoCircuito = (eventoId: number, circuitoId: number) =>
  api<{ status: string }>(`/org/eventos/${eventoId}/circuito`, { method: 'POST', body: { circuito_id: circuitoId } })
export const sairDoCircuito = (eventoId: number) => api(`/org/eventos/${eventoId}/circuito`, { method: 'DELETE' })
export const criarSeletiva = (finalId: number, slug: string, vagas: number) =>
  api<{ status: string }>(`/org/eventos/${finalId}/seletivas`, { method: 'POST', body: { slug, vagas } })
export const removerSeletiva = (finalId: number, id: number) => api(`/org/eventos/${finalId}/seletivas/${id}`, { method: 'DELETE' })
export const decidirSeletiva = (eventoId: number, id: number, aceitar: boolean) =>
  api(`/org/eventos/${eventoId}/seletivas/${id}/decidir`, { method: 'POST', body: { aceitar } })
