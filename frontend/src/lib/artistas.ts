// Item 11.1: grupos (dança, cosplay, canto, teatro) e artistas solo.
import { api } from '@/lib/api'

export type TipoArtista = 'danca' | 'cosplay' | 'canto' | 'teatro' | 'outro'

export const tiposArtista: { valor: TipoArtista; nome: string }[] = [
  { valor: 'danca', nome: 'Dança' },
  { valor: 'cosplay', nome: 'Cosplay' },
  { valor: 'canto', nome: 'Canto e música' },
  { valor: 'teatro', nome: 'Teatro e atuação' },
  { valor: 'outro', nome: 'Outro' },
]
export const nomeTipoArtista = (t: string) => tiposArtista.find((x) => x.valor === t)?.nome ?? t
// Rótulo da experiência ("8 anos de dança", "3 anos de cosplay").
export const textoAnos = (anos: number, tipo: string) => {
  if (anos <= 0) return ''
  const area = tipo === 'cosplay' ? 'cosplay' : tipo === 'danca' ? 'dança' : tipo === 'canto' ? 'música' : tipo === 'teatro' ? 'palco' : 'experiência'
  return `${anos} ${anos === 1 ? 'ano' : 'anos'} de ${area}`
}

export type Midia = {
  id: number
  tipo: 'foto' | 'video'
  url: string
  embed?: string
  plataforma?: 'youtube' | 'instagram' | 'tiktok'
  legenda: string
  evento_slug?: string
  evento_titulo?: string
}

export type ApresentacaoArtista = {
  id: number
  status: 'convidado' | 'confirmado' | 'recusado'
  evento_slug: string
  evento_titulo: string
  evento_inicio: string | null
  cidade: string
  uf: string
}

export type ResultadoArtista = { evento_titulo: string; evento_slug: string; data: string | null; tipo: string; posicao?: number; total?: number }

export type IntegrantePublico = { nome_artistico: string; funcao: string; anos: number; foto_url?: string; instagram?: string }

export type ArtistaPublico = {
  slug: string
  nome: string
  tipo: TipoArtista
  tipo_nome: string
  solo: boolean
  bio: string
  cidade: string
  uf: string
  instagram: string
  tiktok: string
  youtube: string
  logo_url: string
  capa_url: string
  verificado: boolean
  integrantes: IntegrantePublico[]
  outros_integrantes: number
  midias: Midia[]
  proximas: ApresentacaoArtista[]
  passadas: ApresentacaoArtista[]
  resultados: ResultadoArtista[]
  indexar: boolean
}

export type ArtistaItem = {
  slug: string
  nome: string
  tipo: TipoArtista
  solo: boolean
  cidade: string
  uf: string
  logo_url: string
  capa_url: string
  instagram: string
  verificado: boolean
  integrantes: number
}

export type MeuArtista = {
  id: number
  slug: string
  nome: string
  tipo: TipoArtista
  solo: boolean
  logo_url: string
  lider: boolean
  meu_status?: string
  integrantes: number
  atracoes_pendentes: number
}

export type DadosArtista = {
  id?: number
  slug?: string
  nome: string
  tipo: TipoArtista
  solo?: boolean
  bio: string
  cidade: string
  uf: string
  instagram: string
  tiktok: string
  youtube: string
  logo_url: string
  capa_url: string
  publico?: boolean
}

export type IntegrantePainel = {
  id: number
  email: string
  nome_artistico: string
  funcao: string
  desde_ano: number | null
  anos: number
  foto_url: string
  status: 'convidado' | 'ativo'
  visivel: boolean
  menor: boolean
  pendente?: string
  sou_eu: boolean
}

export type PainelArtista = {
  grupo: DadosArtista & { id: number; slug: string; solo: boolean; publico: boolean }
  integrantes: IntegrantePainel[]
  midias: Midia[]
  atracoes: ApresentacaoArtista[]
  instagram_codigo: string
  instagram_pedido: boolean
  instagram_verificado: boolean
}

// Convite de integrante (o que a pessoa vê antes de aceitar).
export type ConviteIntegrante = {
  grupo_nome: string
  grupo_tipo: TipoArtista
  grupo_slug: string
  solo: boolean
  nome_artistico: string
  funcao: string
  desde_ano: number | null
  email_convite: string
  data_nascimento?: string
}

// Autorização do responsável por integrante menor (regras do item 8.1).
export type AutorizacaoIntegrante = {
  grupo_nome: string
  nome_artistico: string
  funcao: string
  tem_foto: boolean
  autorizado: boolean
  autoriza_foto: boolean
  responsavel: string
}

export type PedidoSeloInstagram = { grupo_id: number; nome: string; slug: string; instagram: string; codigo: string; pedido_em: string }

export const obterArtista = (slug: string) => api<ArtistaPublico>(`/artistas/${encodeURIComponent(slug)}`, { semAuth: true })
export const listarMeusArtistas = () => api<MeuArtista[]>('/me/artistas')
export const obterPainelArtista = (id: number) => api<PainelArtista>(`/me/artistas/${id}`)
export const verConviteIntegrante = (token: string, logado: boolean) =>
  api<ConviteIntegrante>(logado ? `/me/artistas-convites/${token}` : `/artistas-convites/${token}`, logado ? {} : { semAuth: true })
export const aceitarConviteIntegrante = (
  token: string,
  corpo: { nome_artistico: string; data_nascimento: string; foto_url: string; instagram: string; responsavel_nome: string; responsavel_email: string },
) => api<{ grupo_id: number; aguardando_responsavel: boolean }>(`/me/artistas-convites/${token}/aceitar`, { method: 'POST', body: corpo })
export const recusarConviteIntegrante = (token: string) => api(`/me/artistas-convites/${token}/recusar`, { method: 'POST' })
export const verAutorizacaoIntegrante = (t: string) => api<AutorizacaoIntegrante>(`/artistas-autorizacao?t=${encodeURIComponent(t)}`, { semAuth: true })
export const decidirAutorizacaoIntegrante = (token: string, autorizar: boolean, foto: boolean) =>
  api<AutorizacaoIntegrante>('/artistas-autorizacao', { method: 'POST', body: { token, autorizar, foto }, semAuth: true })
export const listarPedidosSelo = () => api<PedidoSeloInstagram[]>('/admin/artistas/instagram')
export const decidirSelo = (grupoId: number, aprovar: boolean, instagram: string) =>
  api(`/admin/artistas/${grupoId}/instagram`, { method: 'POST', body: { aprovar, instagram } })
