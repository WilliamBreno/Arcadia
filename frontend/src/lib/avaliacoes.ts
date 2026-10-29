// Avaliação do evento pelo público (item 6.5): só quem fez check-in avalia.
export type ResumoAvaliacoes = { media: number; total: number; mostrar_media: boolean }

export type AvaliacaoPublica = { id: number; nome: string; nota: number; comentario: string; criado_em: string }

export type AvaliacoesEvento = { resumo: ResumoAvaliacoes; avaliacoes: AvaliacaoPublica[] }

export type MinhaAvaliacao = {
  pode_avaliar: boolean
  motivo: string
  avaliacao: { nota: number; comentario: string; ocultada: boolean } | null
}

export const MAX_COMENTARIO = 500

// "4,7" com vírgula, como se escreve no Brasil.
export const formatarMedia = (m: number) => m.toLocaleString('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
