import { api } from '@/lib/api'

export type ResultadoCheckin = 'valido' | 'ja_utilizado' | 'cancelado' | 'outro_evento' | 'nao_encontrado'

export type ValidarCheckinResposta = {
  resultado: ResultadoCheckin
  titular_nome?: string
  tipo_ingresso_nome?: string
  codigo?: string
  utilizado_em?: string
}

export type ItemBusca = {
  id: number
  titular_nome: string
  titular_email: string
  codigo: string
  status: string
  tipo_ingresso_nome: string
}

export type ResumoCheckin = {
  total_pagos: number
  total_utilizados: number
}

export function validarCheckin(eventoId: number, codigo: string, qrToken: string) {
  return api<ValidarCheckinResposta>('/checkin/validar', {
    method: 'POST',
    body: { evento_id: eventoId, codigo, qr_token: qrToken },
  })
}

export function buscarCheckin(eventoId: number, q: string) {
  const query = q ? `?q=${encodeURIComponent(q)}` : ''
  return api<ItemBusca[]>(`/checkin/eventos/${eventoId}/busca${query}`)
}

export function resumoCheckin(eventoId: number) {
  return api<ResumoCheckin>(`/checkin/eventos/${eventoId}/resumo`)
}

export type Staff = {
  usuario_id: number
  nome: string
  email: string
}

export function listarStaff(eventoId: number) {
  return api<Staff[]>(`/org/eventos/${eventoId}/staff`)
}

export function adicionarStaff(eventoId: number, email: string) {
  return api<Staff>(`/org/eventos/${eventoId}/staff`, { method: 'POST', body: { email } })
}

export function removerStaff(eventoId: number, usuarioId: number) {
  return api(`/org/eventos/${eventoId}/staff/${usuarioId}`, { method: 'DELETE' })
}
