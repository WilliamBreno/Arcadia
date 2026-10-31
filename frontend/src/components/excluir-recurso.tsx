import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'

export function ExcluirRecurso({ url, titulo, descricao, destino }: { url: string; titulo: string; descricao: string; destino: string }) {
  const [confirmando, setConfirmando] = useState(false)
  const [ocupado, setOcupado] = useState(false)
  const [erro, setErro] = useState('')
  const navigate = useNavigate(), queryClient = useQueryClient()
  const excluir = async () => {
    setOcupado(true); setErro('')
    try {
      await api(url, { method: 'DELETE' })
      // Desmonta a tela antes de invalidar consultas do recurso removido.
      navigate(destino, { replace: true })
      queryClient.invalidateQueries({ queryKey: ['org-eventos'] })
      queryClient.invalidateQueries({ queryKey: ['meus-artistas'] })
      queryClient.invalidateQueries({ queryKey: ['artistas'] })
      queryClient.invalidateQueries({ queryKey: ['artista'] })
    } catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível excluir. Tente novamente.') }
    finally { setOcupado(false) }
  }
  return <section className="mt-6 flex flex-col items-start gap-3 rounded-xl border border-border p-4">
    <h2 className="font-semibold">{titulo}</h2>
    <p className="text-sm text-muted-foreground">{descricao}</p>
    {confirmando ? <><p className="text-sm">Confirma a exclusão? Esta ação não pode ser desfeita.</p><div className="flex flex-wrap gap-2"><Button type="button" variant="destructive" disabled={ocupado} onClick={excluir}>{ocupado ? 'Excluindo…' : 'Confirmar exclusão'}</Button><Button type="button" variant="outline" disabled={ocupado} onClick={() => { setConfirmando(false); setErro('') }}>Manter</Button></div></> : <Button type="button" variant="destructive" onClick={() => setConfirmando(true)}>{titulo}</Button>}
    {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
  </section>
}
