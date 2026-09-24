import { useEffect, useState } from 'react'

import { api } from '@/lib/api'

type RespostaQR = { payload: string; rotativo: boolean; renova_em_segundos: number }

// useQR busca o texto do QR no servidor e, quando o evento usa QR rotativo,
// renova pouco antes de a janela virar (itens 4.x — anti-print).
export function useQR(caminho: string, semAuth = false) {
  const [qr, setQr] = useState<RespostaQR | null>(null)
  const [erro, setErro] = useState(false)

  useEffect(() => {
    let cancelado = false
    let timer: ReturnType<typeof setTimeout>

    const buscar = async () => {
      try {
        const r = await api<RespostaQR>(caminho, { semAuth })
        if (cancelado) return
        setQr(r)
        setErro(false)
        if (r.rotativo) timer = setTimeout(buscar, Math.max(3, r.renova_em_segundos - 3) * 1000)
      } catch {
        if (!cancelado) {
          setErro(true)
          timer = setTimeout(buscar, 10000)
        }
      }
    }
    buscar()
    return () => {
      cancelado = true
      clearTimeout(timer)
    }
  }, [caminho, semAuth])

  return { qr, erro }
}
