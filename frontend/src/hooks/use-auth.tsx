import { createContext, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'

import { api, definirAccessToken } from '@/lib/api'

export type Usuario = {
  id: number
  nome: string
  email: string
  avatar_url: string
  email_verificado: boolean
  papel_plataforma: string
}

type RespostaAuth = { access_token: string; usuario: Usuario }

type AuthContextState = {
  usuario: Usuario | null
  carregando: boolean
  login: (email: string, senha: string) => Promise<void>
  cadastrar: (nome: string, email: string, senha: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextState | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [usuario, setUsuario] = useState<Usuario | null>(null)
  const [carregando, setCarregando] = useState(true)

  // Ao carregar a página, tenta renovar a sessão usando o refresh token
  // (cookie httpOnly) — assim o access token (só em memória) sobrevive
  // a um F5 sem precisar guardar nada sensível em localStorage.
  useEffect(() => {
    api<RespostaAuth>('/auth/refresh', { method: 'POST', semAuth: true })
      .then((resposta) => {
        definirAccessToken(resposta.access_token)
        setUsuario(resposta.usuario)
      })
      .catch(() => {
        definirAccessToken(null)
        setUsuario(null)
      })
      .finally(() => setCarregando(false))
  }, [])

  const login = async (email: string, senha: string) => {
    const resposta = await api<RespostaAuth>('/auth/login', {
      method: 'POST',
      body: { email, senha },
      semAuth: true,
    })
    definirAccessToken(resposta.access_token)
    setUsuario(resposta.usuario)
  }

  const cadastrar = async (nome: string, email: string, senha: string) => {
    const resposta = await api<RespostaAuth>('/auth/cadastro', {
      method: 'POST',
      body: { nome, email, senha },
      semAuth: true,
    })
    definirAccessToken(resposta.access_token)
    setUsuario(resposta.usuario)
  }

  const logout = async () => {
    await api('/auth/logout', { method: 'POST' }).catch(() => undefined)
    definirAccessToken(null)
    setUsuario(null)
  }

  return (
    <AuthContext.Provider value={{ usuario, carregando, login, cadastrar, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const contexto = useContext(AuthContext)
  if (!contexto) throw new Error('useAuth precisa estar dentro de <AuthProvider>')
  return contexto
}
