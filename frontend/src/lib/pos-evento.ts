// Item 9.7: certificados e pesquisa pós-evento.
export type CertificadoPublico = {
  codigo: string
  nome: string
  funcao: 'publico' | 'participante' | 'jurado' | 'equipe'
  detalhe?: string
  evento_titulo: string
  evento_slug: string
  evento_data: string
  organizador?: string
  carga_horaria_min?: number
  emitido_em: string
  pode_corrigir: boolean
}

export const rotuloFuncaoCert: Record<CertificadoPublico['funcao'], string> = {
  publico: 'Participação',
  participante: 'Competidor(a)',
  jurado: 'Jurado(a)',
  equipe: 'Equipe',
}

export function cargaHoraria(min?: number) {
  if (!min) return ''
  const h = Math.floor(min / 60)
  const m = min % 60
  if (m === 0) return h === 1 ? '1 hora' : `${h} horas`
  return h === 0 ? `${m} minutos` : `${h}h${String(m).padStart(2, '0')}`
}

export type ConfigCertificado = {
  ativo: boolean
  publico: boolean
  participantes: boolean
  jurados: boolean
  equipe: boolean
  carga_horaria_min: number
  texto: string
  assinatura_nome: string
  assinatura_cargo: string
}

export type ResultadoPesquisa = {
  respostas: number
  minimo: number
  nps: number | null
  promotores: number
  neutros: number
  detratores: number
  distribuicao: number[]
  comentarios: { nota: number; comentario: string; data: string }[]
}
