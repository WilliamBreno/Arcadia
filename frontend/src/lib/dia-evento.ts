import { api } from '@/lib/api'
import type { ValidarCheckinResposta } from '@/lib/checkin'

// Item 9.6: painel ao vivo, portões, importados e credenciais impressas.

export type PainelAoVivo = {
  presentes: number
  esperados: number
  faltam: number
  importados_total: number
  importados_entraram: number
  credenciais_total: number
  credenciais_entraram: number
  por_minuto: { minuto: string; entradas: number }[]
  ritmo_por_minuto: number
  previsao_minutos: number | null
  por_portao: { portao: string; entradas: number; ultimos_10: number }[]
  porta_pedidos: number
  porta_ingressos: number
  porta_dinheiro_centavos: number
  porta_pix_centavos: number
  atualizado_em: string
}

export type CredencialImpressa = {
  id: number
  nome: string
  funcao: 'equipe' | 'jurado' | 'imprensa' | 'convidado' | 'expositor'
  detalhe?: string
  qr: string
  manual: boolean
  entrou_em: string | null
}

export const rotuloFuncao: Record<CredencialImpressa['funcao'], string> = {
  equipe: 'Equipe',
  jurado: 'Jurado',
  imprensa: 'Imprensa',
  convidado: 'Convidado',
  expositor: 'Expositor',
}

export const painelAoVivo = (eventoId: number) => api<PainelAoVivo>(`/checkin/eventos/${eventoId}/ao-vivo`)

export const portoesCheckin = (eventoId: number) => api<{ portoes: string[] }>(`/checkin/eventos/${eventoId}/portoes`)

export const validarImportado = (eventoId: number, codigo: string, portao: string) =>
  api<ValidarCheckinResposta>(`/checkin/eventos/${eventoId}/importado`, { method: 'POST', body: { codigo, portao } })

export const lerCredencialEvento = (eventoId: number, texto: string, portao: string) =>
  api<{ nome: string; funcao: CredencialImpressa['funcao']; primeira_vez: boolean }>(`/checkin/eventos/${eventoId}/credencial-evento`, {
    method: 'POST',
    body: { texto, portao },
  })

// Portão escolhido neste aparelho (vale até trocar).
export function portaoSalvo(eventoId: number): string {
  try {
    return localStorage.getItem(`portao:${eventoId}`) ?? ''
  } catch {
    return ''
  }
}

export function salvarPortao(eventoId: number, portao: string) {
  try {
    localStorage.setItem(`portao:${eventoId}`, portao)
  } catch {
    // sem armazenamento: escolhe de novo ao abrir
  }
}
