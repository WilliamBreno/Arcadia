import { useEffect, useRef, useState } from 'react'

// Entrada suave quando o bloco aparece na tela (sobe 24px e ganha opacidade,
// uma vez). Sem IntersectionObserver, ou com "reduzir movimento", o conteúdo
// aparece direto — nunca fica escondido.
export function Surgir({ children }: { children: React.ReactNode }) {
  const ref = useRef<HTMLDivElement>(null)
  const [visivel, setVisivel] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el || typeof IntersectionObserver === 'undefined') {
      setVisivel(true)
      return
    }
    const obs = new IntersectionObserver(
      ([entrada]) => {
        if (entrada.isIntersecting) {
          setVisivel(true)
          obs.disconnect()
        }
      },
      { rootMargin: '0px 0px -10% 0px' },
    )
    obs.observe(el)
    return () => obs.disconnect()
  }, [])

  return (
    <div ref={ref} className={visivel ? 'surgir-visivel' : 'surgir-oculto'}>
      {children}
    </div>
  )
}
