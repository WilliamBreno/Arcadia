// Artist alley e expositores (item 8.6).
import { api } from '@/lib/api'
import type { ValoresInscricao } from '@/lib/inscricao'

export type TipoEspaco = { id: number; nome: string; descricao: string; preco_centavos: number; credenciais: number; ativo: boolean }

export type MesaPublica = { codigo: string; setor: string; tipo: string; ocupada: boolean; marca?: string }
export type ExpositorPublico = { marca: string; categoria: string; descricao: string; instagram: string; mesa: string }
export type ExpositoresPublico = { aberto: boolean; tipos: TipoEspaco[]; mesas: MesaPublica[]; expositores: ExpositorPublico[] }

export type StatusExpositor = 'pendente' | 'aguardando_pagamento' | 'aprovado' | 'rejeitado' | 'desistiu'

export type Credencial = { id: number; nome: string; payload: string; entrou_em: string | null }

export type MeuEspaco = {
  id: number
  status: StatusExpositor
  marca: string
  motivo_rejeicao: string
  tipo: TipoEspaco
  mesa: string | null
  valores: ValoresInscricao | null
  credenciais: Credencial[]
}

export type DadosCandidatura = {
  tipo_espaco_id: number
  marca: string
  categoria: string
  descricao: string
  portfolio_url: string
  instagram: string
  responsavel_nome: string
  telefone: string
}

export type MesaPainel = { id: number; tipo_espaco_id: number; setor: string; codigo: string }
export type ExpositorPainel = DadosCandidatura & {
  id: number
  espaco_id: number | null
  status: StatusExpositor
  motivo_rejeicao: string
  prazo_pagamento_em: string | null
}
export type PainelExpositores = { tipos: TipoEspaco[]; mesas: MesaPainel[]; expositores: ExpositorPainel[] }

export const rotuloStatusExpositor: Record<StatusExpositor, string> = {
  pendente: 'Aguardando análise',
  aguardando_pagamento: 'Aguardando pagamento',
  aprovado: 'Confirmado',
  rejeitado: 'Recusado',
  desistiu: 'Desistiu',
}

export const expositoresDoEvento = (slug: string) => api<ExpositoresPublico>(`/eventos/${slug}/expositores`, { semAuth: true })
export const candidatar = (slug: string, d: DadosCandidatura) => api<{ id: number }>(`/eventos/${slug}/expositores`, { method: 'POST', body: d })
export const meuEspaco = (slug: string) => api<MeuEspaco>(`/eventos/${slug}/meu-espaco`)
export const pagarEspaco = (slug: string, garantia: boolean) =>
  api<{ id: number }>(`/eventos/${slug}/meu-espaco/pagamento`, { method: 'POST', body: { garantia_contratada: garantia } })
export const renomearCredencial = (id: number, nome: string) => api(`/credenciais-expositor/${id}`, { method: 'PUT', body: { nome } })
export const lerCredencial = (eventoId: number, payload: string) =>
  api<{ nome: string; marca: string; mesa: string; primeira_vez: boolean; entrou_em: string | null }>(`/checkin/eventos/${eventoId}/credencial`, {
    method: 'POST',
    body: { payload },
  })

const base = (id: number) => `/org/eventos/${id}`
export const painelExpositores = (id: number) => api<PainelExpositores>(`${base(id)}/expositores`)
export const criarTipoEspaco = (id: number, d: Omit<TipoEspaco, 'id'>) => api(`${base(id)}/tipos-espaco`, { method: 'POST', body: d })
export const atualizarTipoEspaco = (id: number, t: TipoEspaco) => api(`${base(id)}/tipos-espaco/${t.id}`, { method: 'PUT', body: t })
export const excluirTipoEspaco = (id: number, tid: number) => api(`${base(id)}/tipos-espaco/${tid}`, { method: 'DELETE' })
export const criarMesas = (id: number, d: { tipo_espaco_id: number; setor: string; de: number; ate: number }) =>
  api<{ criadas: number }>(`${base(id)}/mesas`, { method: 'POST', body: d })
export const excluirMesa = (id: number, mid: number) => api(`${base(id)}/mesas/${mid}`, { method: 'DELETE' })
export const aprovarExpositor = (id: number, eid: number, espacoId: number) =>
  api(`${base(id)}/expositores/${eid}/aprovar`, { method: 'POST', body: { espaco_id: espacoId } })
export const rejeitarExpositor = (id: number, eid: number, motivo: string) =>
  api(`${base(id)}/expositores/${eid}/rejeitar`, { method: 'POST', body: { motivo } })
