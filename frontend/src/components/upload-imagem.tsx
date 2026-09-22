import { useState } from 'react'

import { enviarArquivo } from '@/lib/upload'
import { Label } from '@/components/ui/label'

export function UploadImagem({
  id,
  rotulo,
  valor,
  onEnviado,
}: {
  id: string
  rotulo: string
  valor: string
  onEnviado: (url: string) => void
}) {
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  const aoSelecionar = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const arquivo = e.target.files?.[0]
    if (!arquivo) return

    setErro(null)
    setEnviando(true)
    try {
      const url = await enviarArquivo(arquivo)
      onEnviado(url)
    } catch (err) {
      setErro(err instanceof Error ? err.message : 'Erro ao enviar')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{rotulo}</Label>
      {valor && <img src={valor} alt="" className="size-20 rounded-lg object-cover" />}
      <input id={id} type="file" accept="image/jpeg,image/png" onChange={aoSelecionar} disabled={enviando} className="text-sm" />
      {enviando && <p className="text-xs text-muted-foreground">Enviando…</p>}
      {erro && <p className="text-xs text-destructive">{erro}</p>}
    </div>
  )
}
