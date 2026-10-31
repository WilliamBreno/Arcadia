// Item 11.6: limite de jurados e participantes por evento.
import { api } from '@/lib/api'

export type SituacaoVagas = { limite: number | null; ocupadas: number; esgotado: boolean }
export type Modalidade = 'cosplay' | 'danca' | 'canto' | 'atuacao'

export const modalidades: { valor: Modalidade; nome: string }[] = [
  { valor: 'cosplay', nome: 'Cosplay' },
  { valor: 'danca', nome: 'Dança' },
  { valor: 'canto', nome: 'Canto' },
  { valor: 'atuacao', nome: 'Atuação' },
]

export type ResumoVagas = {
  jurados: SituacaoVagas
  participantes: SituacaoVagas
  por_tipo: Record<Modalidade, SituacaoVagas>
  lista_espera: boolean
}

export type VagasPublicas = Omit<ResumoVagas, 'jurados'>

export const obterVagasOrg = (eventoId: number) => api<ResumoVagas>(`/org/eventos/${eventoId}/vagas`)
export const obterVagasPublicas = (slug: string) => api<VagasPublicas>(`/eventos/${slug}/vagas`, { semAuth: true })

export const restantes = (s: SituacaoVagas) => (s.limite === null ? null : Math.max(s.limite - s.ocupadas, 0))
