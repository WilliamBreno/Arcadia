// Perfil público do talento (item 8.1).
export type Participacao = {
  ficha_id: number
  evento_titulo: string
  evento_slug: string
  data: string | null
  tipo: string
  personagem?: string
  obra?: string
  posicao?: number
  total?: number
  oculta?: boolean
}

export type PerfilPublico = {
  slug: string
  nome_artistico: string
  bio: string
  foto_url?: string
  cidade?: string
  uf?: string
  instagram?: string
  tiktok?: string
  youtube?: string
  indexar: boolean
  participacoes: Participacao[]
}

export type DadosPerfil = {
  nome_artistico: string
  bio: string
  foto_url: string
  cidade: string
  uf: string
  instagram: string
  tiktok: string
  youtube: string
  data_nascimento: string
  publico: boolean
  responsavel_nome: string
  responsavel_email: string
}

export type MeuPerfil = {
  perfil: DadosPerfil | null
  slug?: string
  situacao: { visivel: boolean; menor: boolean; indexar: boolean; mostra_foto: boolean; motivo: string }
  autorizado: boolean
  autoriza_foto: boolean
  participacoes: Participacao[]
}

export const rotuloTipo: Record<string, string> = { cosplay: 'Cosplay', danca: 'Dança', canto: 'Canto', atuacao: 'Atuação' }

// Idade em anos completos (o servidor decide; aqui é só para mostrar os campos certos).
export function idade(nascimento: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(nascimento)) return null
  const [a, m, d] = nascimento.split('-').map(Number)
  const hoje = new Date()
  let anos = hoje.getFullYear() - a
  if (hoje.getMonth() + 1 < m || (hoje.getMonth() + 1 === m && hoje.getDate() < d)) anos--
  return anos
}
