// Galeria oficial do evento (item 8.7).
import { api } from '@/lib/api'
import { enviarFotoGaleria } from '@/lib/upload'

export type Foto = { id: number; url: string; credito: string; marcados: { nome: string; slug: string }[] }
export type PainelGaleria = {
  fotos: { id: number; url: string; oculta: boolean }[]
  remocoes: { id: number; foto_id: number; url: string; motivo: string; criado_em: string }[]
  limite: number
}
export type Fotografo = { usuario_id: number; nome: string; email: string }

export const galeria = (slug: string) => api<Foto[]>(`/eventos/${slug}/galeria`, { semAuth: true })
export const galeriaEu = (slug: string) => api<{ pode_enviar: boolean; marcadas: number[] }>(`/eventos/${slug}/galeria/eu`)
export const painelGaleria = (slug: string) => api<PainelGaleria>(`/eventos/${slug}/galeria/painel`)
export const enviarFoto = (slug: string, arquivo: File) => enviarFotoGaleria(slug, arquivo)
export const excluirFoto = (slug: string, id: number) => api(`/eventos/${slug}/galeria/${id}`, { method: 'DELETE' })
export const marcarFoto = (id: number, marcar: boolean) => api(`/galeria/fotos/${id}/marcar`, { method: 'POST', body: { marcar } })
export const pedirRemocao = (id: number, motivo: string) =>
  api(`/galeria/fotos/${id}/remocao`, { method: 'POST', body: { motivo }, semAuth: true })
export const decidirRemocao = (slug: string, id: number, manter: boolean) =>
  api(`/eventos/${slug}/galeria/remocoes/${id}`, { method: 'POST', body: { manter } })

export const fotografos = (eventoId: number) => api<Fotografo[]>(`/org/eventos/${eventoId}/fotografos`)
export const adicionarFotografo = (eventoId: number, email: string) => api(`/org/eventos/${eventoId}/fotografos`, { method: 'POST', body: { email } })
export const removerFotografo = (eventoId: number, uid: number) => api(`/org/eventos/${eventoId}/fotografos/${uid}`, { method: 'DELETE' })
