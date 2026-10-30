// Item 10.4: Plano Pro e destaque na vitrine (cobranças da Evve).
export type Precos = { pro_mensal_centavos: number; pro_anual_centavos: number; destaque_dia_centavos: number; destaque_vagas_dia: number }
export type SituacaoPro = { pro: boolean; ate: string | null; destaque_incluido_usado: boolean; precos: Precos }
export type Cobranca = {
  id: number
  tipo: 'pro_mensal' | 'pro_anual' | 'destaque'
  descricao: string
  valor_centavos: number
  status: 'pendente' | 'paga' | 'devolvida' | 'expirada'
  expira_em: string
  pago_em: string | null
}
export type DestaqueView = {
  id: number
  evento_id: number
  evento_titulo: string
  alvo_tipo: 'cidade' | 'categoria'
  alvo_valor: string
  dia: string
  status: 'reservado' | 'pago' | 'devolvido'
  incluido_pro: boolean
}
