import { ApiError, obterAccessToken, urlApi } from '@/lib/api'
export type PostMural = { id: number; autor: string; texto: string; status: 'pendente' | 'aprovado' | 'rejeitado' | 'removido'; destaque: boolean; fotos: number[]; curtidas: number; criado_em: string; ia_status?: string; ia_alerta?: boolean; ia_motivos?: string[]; motivo?: string; denuncias?: number }
export type PaginaMural = { titulo: string; slug: string; posts: PostMural[]; destaques: PostMural[] }
export type EuMural = { pode_postar: boolean; motivo: string; moderador: boolean; entrada_livre: boolean; meus: PostMural[]; curtidas: number[] }
export const baseMural = (slug: string) => `/eventos/${encodeURIComponent(slug)}/mural`
export const rotulosMural = { pendente: 'Aguardando aprovação', aprovado: 'Publicado', rejeitado: 'Não aprovado', removido: 'Removido' }
export async function enviarMural(slug: string, texto: string, chave: string, fotos: File[], participando: boolean) {
 const body = new FormData(); body.append('texto', texto); body.append('chave', chave); body.append('consentiu', 'true'); body.append('participando', String(participando)); fotos.forEach(f => body.append('fotos', f))
 const r = await fetch(urlApi(baseMural(slug)), { method: 'POST', body, credentials: 'include', headers: { Authorization: `Bearer ${obterAccessToken()}` } })
 const dados = await r.json(); if (!r.ok) throw new ApiError(r.status, dados.erro ?? 'Não foi possível enviar a publicação.'); return dados as { id: number }
}
export async function fotoMural(slug: string, post: number, foto: number, signal?: AbortSignal) {
 const token = obterAccessToken()
 const r = await fetch(urlApi(`${baseMural(slug)}/${post}/fotos/${foto}`), { credentials: 'include', signal, headers: token ? { Authorization: `Bearer ${token}` } : {} })
 if (!r.ok) throw new ApiError(r.status, 'Foto indisponível. A publicação pode ter sido removida ou estar em revisão.')
 return r.blob()
}
