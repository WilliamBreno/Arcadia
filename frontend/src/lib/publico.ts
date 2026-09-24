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
    nome_publico: string
    slug: string
    descricao: string
    logo_url: string
    instagram: string
    site: string
  } | null
  ingressos: {
    id: number
    nome: string
    descricao: string
    preco_centavos: number
    quantidade: number
    ativo: boolean
    meia_entrada: boolean
    sessao_id: number | null
  }[]
  taxa_plataforma_centavos: number
  garantia_centavos: number
  sessoes: { id: number; titulo: string; inicio_em: string; fim_em: string | null; status: 'ativa' | 'cancelada' }[]
  cronograma: { id: number; titulo: string; descricao: string; local: string; inicio_em: string; fim_em: string | null }[]
}

export type OrganizadorDetalhe = {
  organizador: {
    nome_publico: string
    slug: string
    descricao: string
    logo_url: string
    instagram: string
    site: string
  }
  eventos: EventoPublicoItem[]
}
