import type { CSSProperties } from 'react'

// estiloTema aplica a cor do organizador (#RRGGBB validada no backend) só
// nas variáveis do shadcn dentro da página do evento, com texto legível.
export function estiloTema(cor: string | undefined | null): CSSProperties | undefined {
  if (!cor || !/^#[0-9a-fA-F]{6}$/.test(cor)) return undefined
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(cor.slice(i, i + 2), 16))
  const luminancia = (0.299 * r + 0.587 * g + 0.114 * b) / 255
  return {
    '--primary': cor,
    '--primary-foreground': luminancia > 0.6 ? '#111111' : '#ffffff',
    '--ring': cor,
  } as CSSProperties
}
