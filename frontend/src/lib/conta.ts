export type MeuEvento = {
  evento_id: number
  titulo: string
  slug: string
  inicio_em: string | null
  selos: ('organizador' | 'jurado' | 'participante' | 'participante_especial' | 'staff' | 'ingresso')[]
}

export type MeuIngresso = {
  id: number
  tipo_ingresso_id: number
  titular_nome: string
  titular_email: string
  preco_centavos: number
  taxa_plataforma_centavos: number
  total_centavos: number
  status: string
  codigo: string
  qr_token: string
  utilizado_em: string | null
  evento_titulo: string
  evento_slug: string
  evento_inicio_em: string | null
}
