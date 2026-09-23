import { api } from '@/lib/api'

export type VendaItem = {
  id: number
  titular_nome: string
  titular_email: string
  tipo_ingresso_nome: string
  preco_centavos: number
  status: string
  codigo: string
  criado_em: string
  utilizado_em: string | null
}

export type ResumoVendasTipo = {
  tipo_ingresso_id: number
  tipo_ingresso_nome: string
  quantidade: number
  receita_centavos: number
}

export type Vendas = {
  total_vendido: number
  receita_centavos: number
  por_tipo: ResumoVendasTipo[]
  itens: VendaItem[]
}

export function obterVendas(eventoId: number) {
  return api<Vendas>(`/org/eventos/${eventoId}/vendas`)
}
