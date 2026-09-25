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

export const CATEGORIAS_EVENTO = [
  'Games e Geek',
  'Cosplay e Cultura Pop',
  'Anime e Mangá',
  'Dança',
  'Música e Shows',
  'Canto e Karaokê',
  'Teatro e Atuação',
  'Festivais',
  'Feiras e Convenções',
  'Torneios e E-sports',
  'Workshops e Cursos',
  'Arte e Exposições',
  'Gastronomia',
  'Infantil',
] as const

export const DESCRICAO_TIPO_ACESSO = {
  ingresso:
    'Você define lotes e preços, o público compra pelo site e recebe um QR Code para a entrada. É cobrada uma taxa de serviço por ingresso, e o repasse chega depois do evento.',
  cadastro:
    'O público se inscreve no evento, de graça ou com um valor (ex.: torneios e concursos). Cada inscrição também gera QR Code. Eventos gratuitos devem usar esta opção.',
} as const
