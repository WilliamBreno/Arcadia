// Check-in offline: pacote de ingressos + fila de entradas no IndexedDB.
// O pacote leva o HASH (SHA-256) do token de cada ingresso; o aparelho
// compara o hash do QR lido, sem ter os tokens em claro.

import { api } from '@/lib/api'
import type { ValidarCheckinResposta } from '@/lib/checkin'

export type SessaoPacote = { id: number; titulo: string; inicio_em: string; fim_em: string | null; status: 'ativa' | 'cancelada' }

export type IngressoPacote = {
  codigo: string
  hash_token: string
  nome: string
  tipo: string
  meia_entrada: boolean
  status: 'pago' | 'utilizado'
  utilizado_em: string | null
  sessao_ids: number[]
}

export type Pacote = { evento_id: number; gerado_em: string; sessoes: SessaoPacote[]; ingressos: IngressoPacote[] }

export type EntradaFila = { entrada_id: string; evento_id: number; codigo: string; sessao_id: number | null; lido_em: string; nome: string }

const BANCO = 'evve-checkin'

function abrir(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(BANCO, 1)
    req.onupgradeneeded = () => {
      const db = req.result
      db.createObjectStore('pacotes', { keyPath: 'evento_id' })
      db.createObjectStore('fila', { keyPath: 'entrada_id' })
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

async function tx<T>(store: string, modo: IDBTransactionMode, fn: (s: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await abrir()
  return new Promise((resolve, reject) => {
    const req = fn(db.transaction(store, modo).objectStore(store))
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export const salvarPacote = (p: Pacote) => tx('pacotes', 'readwrite', (s) => s.put(p))
export const lerPacote = (eventoId: number) => tx<Pacote | undefined>('pacotes', 'readonly', (s) => s.get(eventoId))
export const enfileirar = (e: EntradaFila) => tx('fila', 'readwrite', (s) => s.put(e))
export const removerDaFila = (id: string) => tx('fila', 'readwrite', (s) => s.delete(id))
export async function listarFila(eventoId: number): Promise<EntradaFila[]> {
  const todas = await tx<EntradaFila[]>('fila', 'readonly', (s) => s.getAll())
  return todas.filter((e) => e.evento_id === eventoId)
}

export async function sha256Hex(texto: string): Promise<string> {
  const buf = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(texto))
  return Array.from(new Uint8Array(buf))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

const FOLGA_ANTES_MS = 3 * 3600_000
const DURACAO_PADRAO_MS = 12 * 3600_000

// Espelho de SessaoAtual do backend (mesma janela: 3h antes até o fim).
export function sessaoAtual(sessoes: SessaoPacote[], agora: Date): SessaoPacote | null {
  let melhor: SessaoPacote | null = null
  for (const s of sessoes) {
    if (s.status !== 'ativa') continue
    const ini = new Date(s.inicio_em).getTime()
    const fim = s.fim_em ? new Date(s.fim_em).getTime() : ini + DURACAO_PADRAO_MS
    const t = agora.getTime()
    if (t < ini - FOLGA_ANTES_MS || t > fim) continue
    if (!melhor || Math.abs(t - ini) < Math.abs(t - new Date(melhor.inicio_em).getTime())) melhor = s
  }
  return melhor
}

export type DecisaoOffline = { resposta: ValidarCheckinResposta; entrada?: EntradaFila }

function novoId(): string {
  return crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

// validarOffline decide a leitura com o pacote + fila local. Não grava nada:
// quem chama enfileira `entrada` quando o resultado é válido.
export async function validarOffline(pacote: Pacote, fila: EntradaFila[], codigo: string, token: string, agora = new Date()): Promise<DecisaoOffline> {
  if (token.startsWith('R')) {
    return { resposta: { resultado: 'requer_internet' } }
  }
  const ing = pacote.ingressos.find((i) => i.codigo === codigo)
  if (!ing || (await sha256Hex(token)) !== ing.hash_token) {
    return { resposta: { resultado: 'nao_encontrado' } }
  }
  const base = { titular_nome: ing.nome, tipo_ingresso_nome: ing.tipo, meia_entrada: ing.meia_entrada, codigo: ing.codigo }

  const temSessoes = pacote.sessoes.length > 0
  let sessaoId: number | null = null
  if (temSessoes) {
    const atual = sessaoAtual(pacote.sessoes, agora)
    if (!atual || (ing.sessao_ids.length > 0 && !ing.sessao_ids.includes(atual.id))) {
      return { resposta: { resultado: 'fora_da_sessao', ...base } }
    }
    sessaoId = atual.id
  }

  const jaNaFila = fila.find((e) => e.codigo === codigo && e.sessao_id === sessaoId)
  if (jaNaFila) {
    return { resposta: { resultado: 'ja_utilizado', ...base, utilizado_em: new Date(jaNaFila.lido_em).toLocaleTimeString('pt-BR') } }
  }
  // sem sessões, quem já constava como utilizado no pacote não entra de novo
  if (!temSessoes && ing.status === 'utilizado') {
    return { resposta: { resultado: 'ja_utilizado', ...base, utilizado_em: ing.utilizado_em ? new Date(ing.utilizado_em).toLocaleTimeString('pt-BR') : undefined } }
  }

  const entrada: EntradaFila = { entrada_id: novoId(), evento_id: pacote.evento_id, codigo, sessao_id: sessaoId, lido_em: agora.toISOString(), nome: ing.nome }
  return { resposta: { resultado: 'valido', ...base, utilizado_em: agora.toLocaleTimeString('pt-BR') }, entrada }
}

export type ResultadoSync = { entrada_id: string; codigo: string; resultado: string }

// sincronizarFila envia as entradas pendentes; devolve o que o servidor decidiu.
// Só remove da fila o que o servidor confirmou (qualquer resultado final).
export async function sincronizarFila(eventoId: number): Promise<{ enviados: number; resultados: ResultadoSync[]; nomes: Record<string, string> }> {
  const fila = await listarFila(eventoId)
  if (fila.length === 0) return { enviados: 0, resultados: [], nomes: {} }
  const r = await api<{ resultados: ResultadoSync[] }>('/checkin/sincronizar', {
    method: 'POST',
    body: {
      evento_id: eventoId,
      entradas: fila.map((e) => ({ entrada_id: e.entrada_id, codigo: e.codigo, sessao_id: e.sessao_id, lido_em: e.lido_em })),
    },
  })
  const nomes: Record<string, string> = {}
  for (const e of fila) nomes[e.entrada_id] = e.nome
  for (const res of r.resultados) await removerDaFila(res.entrada_id)
  return { enviados: fila.length, resultados: r.resultados, nomes }
}

export const baixarPacote = (eventoId: number) => api<Pacote>(`/checkin/eventos/${eventoId}/pacote`)
