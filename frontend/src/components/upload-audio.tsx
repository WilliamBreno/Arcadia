import { useState } from 'react'

import { enviarArquivo } from '@/lib/upload'
import { Label } from '@/components/ui/label'

export function UploadAudio({
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
      onEnviado(await enviarArquivo(arquivo))
    } catch (err) {
      setErro(err instanceof Error ? err.message : 'Erro ao enviar')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{rotulo}</Label>
      {valor && <audio controls src={valor} className="w-full" />}
      <input id={id} type="file" accept="audio/mpeg,audio/wav,audio/mp4" onChange={aoSelecionar} disabled={enviando} className="text-sm" />
      <p className="text-xs text-muted-foreground">MP3, WAV ou M4A, até 10 MB.</p>
      {enviando && <p className="text-xs text-muted-foreground">Enviando…</p>}
      {erro && <p className="text-xs text-destructive">{erro}</p>}
    </div>
  )
}
