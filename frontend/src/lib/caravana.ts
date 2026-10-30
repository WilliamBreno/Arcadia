// Caravanas (item 8.7).
import { api } from '@/lib/api'

export type DadosCaravana = {
  nome: string
  cidade: string
  uf: string
  ponto_embarque: string
  saida_em: string | null
  vagas: number | null
  telefone: string
}

export type Caravana = DadosCaravana & {
  codigo: string
  membros: number
  evento_titulo: string
  evento_slug: string
  responsavel: boolean
  membro: boolean
  responsavel_nome: string
}

export type MinhaCaravana = { codigo: string; nome: string; cidade: string; ponto_embarque: string; saida_em: string | null; responsavel: boolean }
export type LinhaEmbarque = { id: number; nome: string; telefone: string; tem_ingresso: boolean; embarcou_em: string | null }
export type ResumoCaravana = { nome: string; cidade: string; uf: string; saida_em: string | null; membros: number; com_ingresso: number; responsavel: string }

export const criarCaravana = (slug: string, d: DadosCaravana) => api<{ codigo: string }>(`/eventos/${slug}/caravanas`, { method: 'POST', body: d })
export const minhasCaravanas = (slug: string) => api<MinhaCaravana[]>(`/eventos/${slug}/minhas-caravanas`)
export const verCaravana = (codigo: string) => api<Caravana>(`/caravanas/${codigo}`)
export const atualizarCaravana = (codigo: string, d: DadosCaravana) => api(`/caravanas/${codigo}`, { method: 'PUT', body: d })
export const encerrarCaravana = (codigo: string) => api(`/caravanas/${codigo}`, { method: 'DELETE' })
export const entrarCaravana = (codigo: string, nome: string, telefone: string, aceite: boolean) =>
  api(`/caravanas/${codigo}/entrar`, { method: 'POST', body: { nome, telefone, aceite } })
export const sairCaravana = (codigo: string) => api(`/caravanas/${codigo}/eu`, { method: 'DELETE' })
export const listaEmbarque = (codigo: string) => api<LinhaEmbarque[]>(`/caravanas/${codigo}/lista`)
export const marcarEmbarque = (codigo: string, id: number, embarcou: boolean) =>
  api(`/caravanas/${codigo}/membros/${id}/embarque`, { method: 'POST', body: { embarcou } })
export const caravanasDoEvento = (eventoId: number) => api<ResumoCaravana[]>(`/org/eventos/${eventoId}/caravanas`)
