import type { RegraTaxas } from '@/lib/taxas'
import type { ResumoAvaliacoes } from '@/lib/avaliacoes'

export type EventoPublicoItem = {
  slug: string
  titulo: string
  capa_url: string
  categoria: string
  cidade?: string
  uf?: string
  inicio_em: string | null
  preco_a_partir_centavos: number | null
  gratuito: boolean
  latitude?: number
  longitude?: number
  // Ingresso mais barato já com a taxa que o comprador paga (item 6.3).
  preco_final_a_partir_centavos?: number
}

export type ListaEventosResposta = {
  eventos: EventoPublicoItem[]
  total: number
  pagina: number
  por_pagina: number
}

export type EventoDetalhe = {
  evento: {
    slug: string
    titulo: string
    descricao: string
    capa_url: string
    categoria: string
    inicio_em: string | null
    fim_em: string | null
    classificacao_etaria: string
    garantia_habilitada: boolean
    aprovacao_manual: boolean
    cor_tema: string
    politica_cancelamento_texto: string
    modo_participantes: 'nenhum' | 'convite' | 'inscricao_aberta' | 'ambos'
    inscricao_talento_centavos?: number // item 8.4
    timezone?: string
  }
  local: {
    nome: string
    logradouro: string
    numero: string
    bairro: string
    cidade: string
    uf: string
    latitude: number | null
    longitude: number | null
  } | null
  organizador: {
    id: number
    nome_publico: string
    slug: string
    descricao: string
    logo_url: string
    instagram: string
    site: string
    verificado: boolean
    pro?: boolean // item 10.4
  } | null
  ingressos: {
    id: number
    nome: string
    descricao: string
    preco_centavos: number
    quantidade: number
    ativo: boolean
    meia_entrada: boolean
    sessao_ids: number[]
    max_por_pedido: number
    // Item 7.3: sem vaga à venda agora (já contando as guardadas para a fila).
    esgotado?: boolean
  }[]
  taxas: RegraTaxas
  vendas_encerradas: boolean
  // Item 7.5: IDs dos pixels de anúncio do organizador (vazios = desligados).
  pixels?: { meta: string; ga4: string; tiktok: string }
  // Item 7.6: perguntas do organizador, respondidas por ingresso.
  perguntas?: import('@/lib/perguntas').Pergunta[]
  sessoes: { id: number; titulo: string; inicio_em: string; fim_em: string | null; status: 'ativa' | 'cancelada' }[]
  cronograma: { id: number; titulo: string; descricao: string; local: string; inicio_em: string; fim_em: string | null }[]
}

export type OrganizadorDetalhe = {
  organizador: {
    id: number
    nome_publico: string
    slug: string
    descricao: string
    logo_url: string
    instagram: string
    site: string
    verificado: boolean
    pro?: boolean // item 10.4
  }
  eventos: EventoPublicoItem[]
  seguidores: number
  avaliacoes?: ResumoAvaliacoes
}
