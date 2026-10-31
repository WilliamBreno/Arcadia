import { api, ApiError, obterAccessToken, urlApi } from '@/lib/api'

export type ChatParticipante = { usuario_id: number; nome: string; papel: string; pode_fixar: boolean; lido_ate: number }
export type ChatMensagem = { id: number; usuario_id: number; nome: string; texto: string; tem_foto: boolean; fixada: boolean; criado_em: string }
export type ChatAtencao = { id: number; jurado_id: number; nome: string; mensagem_id: number; criado_em: string; atendido_em: string | null }
export type ChatPagina = { titulo: string; mensagens: ChatMensagem[]; fixadas: ChatMensagem[]; participantes: ChatParticipante[]; atencoes: ChatAtencao[]; lido_ate: number; email_ativo: boolean; pode_fixar: boolean }
export type ChatCanal = { evento_id: number; titulo: string; nao_lidas: number }
export const baseChat = (id: number) => `/me/chats-eventos/${id}`
export const papeisChat: Record<string, string> = { organizador: 'Organizador', gestor: 'Gestão', financeiro: 'Financeiro', leitura: 'Equipe', checkin: 'Portaria', staff: 'Equipe do evento', coprodutor: 'Coprodutor' }
export async function enviarChat(id: number, texto: string, chave: string, foto: File | null) {
  if (!foto) return api<{ id: number }>(`${baseChat(id)}/mensagens`, { method: 'POST', body: { texto, chave } })
  const body = new FormData(); body.append('texto', texto); body.append('chave', chave); body.append('arquivo', foto)
  const r = await fetch(urlApi(`${baseChat(id)}/mensagens`), { method: 'POST', body, credentials: 'include', headers: { Authorization: `Bearer ${obterAccessToken()}` } })
  const dados = await r.json()
  if (!r.ok) throw new ApiError(r.status, dados.erro ?? 'Não foi possível enviar a foto.')
  return dados as { id: number }
}
export async function fotoChat(id: number, mensagem: number, signal: AbortSignal) {
  const r = await fetch(urlApi(`${baseChat(id)}/mensagens/${mensagem}/foto`), { credentials: 'include', signal, headers: { Authorization: `Bearer ${obterAccessToken()}` } })
  if (!r.ok) throw new ApiError(r.status, 'Foto indisponível ou acesso encerrado.')
  return r.blob()
}
