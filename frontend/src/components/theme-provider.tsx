import { createContext, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'

type Tema = 'light' | 'dark'

type ThemeProviderState = {
  tema: Tema
  alternarTema: () => void
}

const CHAVE_LOCALSTORAGE = 'arcadia-tema'

const ThemeProviderContext = createContext<ThemeProviderState | undefined>(undefined)

function temaInicial(): Tema {
  if (typeof window === 'undefined') return 'light'
  try {
    const salvo = window.localStorage.getItem(CHAVE_LOCALSTORAGE)
    if (salvo === 'light' || salvo === 'dark') return salvo
  } catch {
    // localStorage indisponível (modo privado, etc.) — segue com o padrão.
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [tema, setTema] = useState<Tema>(temaInicial)

  useEffect(() => {
    const root = document.documentElement
    root.classList.toggle('dark', tema === 'dark')
    try {
      window.localStorage.setItem(CHAVE_LOCALSTORAGE, tema)
    } catch {
      // per-viewer apenas; sem problema se não persistir.
    }
  }, [tema])

  const alternarTema = () => setTema((atual) => (atual === 'dark' ? 'light' : 'dark'))

  return (
    <ThemeProviderContext.Provider value={{ tema, alternarTema }}>
      {children}
    </ThemeProviderContext.Provider>
  )
}

export function useTheme() {
  const contexto = useContext(ThemeProviderContext)
  if (!contexto) throw new Error('useTheme precisa estar dentro de <ThemeProvider>')
  return contexto
}
