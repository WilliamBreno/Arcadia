export type Organizador = {
  id: number
  nome_publico: string
  slug: string
  descricao: string
  logo_url: string
  tipo_pessoa: 'pf' | 'pj'
  documento: string
  chave_pix: string
  tipo_chave_pix: string
  instagram: string
  site: string
  status: 'pendente' | 'ativo' | 'suspenso'
}

export type Local = {
  id: number
  nome: string
  logradouro: string
  numero: string
  bairro: string
  cidade: string
  uf: string
  cep: string
  latitude: number | null
  longitude: number | null
  capacidade: number | null
  observacoes: string
}
