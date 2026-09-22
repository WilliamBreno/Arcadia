const BASE_URL = `${import.meta.env.VITE_API_URL ?? '/api'}/v1`

export class ApiError extends Error {
  status: number
  constructor(status: number, mensagem: string) {
    super(mensagem)
    this.status = status
  }
}

let accessTokenAtual: string | null = null

export function definirAccessToken(token: string | null) {
  accessTokenAtual = token
}

type Opcoes = {
  method?: string
  body?: unknown
  semAuth?: boolean
}

// api() fala com o backend em /api/v1. Sempre manda cookies (o refresh
// token httpOnly viaja junto) e, quando há access token em memória,
// manda também o header Authorization.
export async function api<T>(caminho: string, opcoes: Opcoes = {}): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (accessTokenAtual && !opcoes.semAuth) {
    headers.Authorization = `Bearer ${accessTokenAtual}`
  }

  const resposta = await fetch(`${BASE_URL}${caminho}`, {
    method: opcoes.method ?? 'GET',
    headers,
    credentials: 'include',
    body: opcoes.body ? JSON.stringify(opcoes.body) : undefined,
  })

  const dados = await resposta.json().catch(() => null)

  if (!resposta.ok) {
    throw new ApiError(resposta.status, dados?.erro ?? 'Erro inesperado')
  }

  return dados as T
}
