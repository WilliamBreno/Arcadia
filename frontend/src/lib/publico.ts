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
    politica_cancelamento_texto: string
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
  }[]
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
