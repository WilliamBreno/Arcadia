// Item 9.8: coprodução com divisão de repasse (percentual em pontos-base,
// 10000 = 100%).
export type Coproducao = {
  id: number
  evento_id: number
  evento_titulo: string
  evento_inicio_em: string | null
  organizador_id: number
  organizador_nome: string
  principal_nome: string
  percentual_bp: number
  proposto_bp: number | null
  status: 'convidado' | 'ativa' | 'recusada' | 'encerrada'
  pode_sair?: boolean
}

export type ResumoCoproducao = { coprodutores: Coproducao[]; principal_bp: number; tem_venda: boolean; travada: boolean }

export const formatarBP = (bp: number) => `${(bp / 100).toLocaleString('pt-BR', { maximumFractionDigits: 2 })}%`

// "30" ou "12,5" → 3000 / 1250.
export function lerPercentual(texto: string): number | null {
  const n = Number(texto.replace(',', '.'))
  if (!texto.trim() || Number.isNaN(n)) return null
  return Math.round(n * 100)
}
