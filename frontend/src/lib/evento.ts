export type Evento = {
  id: number
  local_id: number | null
  titulo: string
  slug: string
  descricao: string
  categoria: string
  capa_url: string
  inicio_em: string | null
  fim_em: string | null
  timezone: string
  classificacao_etaria: string
  visibilidade: 'publico' | 'nao_listado' | 'privado'
  tipo_acesso: 'ingresso' | 'cadastro'
  status: 'rascunho' | 'publicado' | 'encerrado' | 'cancelado'
  modo_participantes: 'nenhum' | 'convite' | 'inscricao_aberta' | 'ambos'
  capacidade_total: number | null
  garantia_habilitada: boolean
  aprovacao_manual: boolean
  qr_rotativo: boolean
  cor_tema: string
  politica_cancelamento_texto: string
  max_itens_por_pedido: number
  publicado_em: string | null
}

export type TipoIngresso = {
  id: number
  nome: string
  descricao: string
  preco_centavos: number
  quantidade: number
  vendas_inicio: string | null
  vendas_fim: string | null
  min_por_pedido: number
  max_por_pedido: number
  ordem: number
  lote_grupo: string
  meia_entrada: boolean
  sessao_id: number | null
  ativo: boolean
}

export function formatarCentavos(centavos: number): string {
  return (centavos / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
}
