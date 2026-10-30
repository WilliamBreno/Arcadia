// Consentimento de cookies de marketing (item 7.5, LGPD + guia de cookies da
// ANPD). Sem escolha registrada, nada de marketing carrega. A escolha fica só
// neste navegador e pode ser mudada a qualquer momento em /cookies.
const CHAVE = 'evve-cookies-v1'
const EVENTO = 'evve-consentimento'

export type Escolha = { marketing: boolean; em: string }

export function lerEscolha(): Escolha | null {
  try {
    const bruto = localStorage.getItem(CHAVE)
    return bruto ? (JSON.parse(bruto) as Escolha) : null
  } catch {
    return null
  }
}

export function salvarEscolha(marketing: boolean) {
  const escolha: Escolha = { marketing, em: new Date().toISOString() }
  try {
    localStorage.setItem(CHAVE, JSON.stringify(escolha))
  } catch {
    // sem localStorage: vale só nesta página
  }
  window.dispatchEvent(new CustomEvent(EVENTO, { detail: escolha }))
  // Quem recusa depois de ter aceitado: recarrega para descarregar os scripts.
  if (!marketing && (window as unknown as { __evvePixels?: boolean }).__evvePixels) window.location.reload()
}

export const aceitouMarketing = () => lerEscolha()?.marketing === true

export function aoMudarEscolha(fn: (e: Escolha) => void) {
  const h = (ev: Event) => fn((ev as CustomEvent<Escolha>).detail)
  window.addEventListener(EVENTO, h)
  return () => window.removeEventListener(EVENTO, h)
}
