export type ItemPedido = {
  id: number
  tipo_ingresso_id: number
  titular_nome: string
  titular_email: string
  preco_centavos: number
  taxa_plataforma_centavos: number
  garantia_contratada: boolean
  garantia_centavos: number
  desconto_centavos: number
  total_centavos: number
  status: 'reservado' | 'pago' | 'utilizado' | 'cancelado' | 'reembolsado' | 'expirado'
  codigo: string
  qr_token?: string
}

export type Pedido = {
  id: number
  evento_id: number
  status: 'aberto' | 'aguardando_pagamento' | 'pago' | 'expirado' | 'cancelado' | 'reembolsado_parcial' | 'reembolsado'
  total_centavos: number
  expira_em: string | null
  itens: ItemPedido[]
}
