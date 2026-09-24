import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

// Troca o titular nominal do ingresso. O QR atual deixa de valer: o novo
// titular recebe um novo QR por e-mail (item 2.6).
export function TransferirIngresso({ itemId, onConcluido }: { itemId: number; onConcluido: () => void }) {
  const queryClient = useQueryClient()
  const [aberto, setAberto] = useState(false)
  const [nome, setNome] = useState('')
  const [email, setEmail] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

  if (!aberto) {
    return (
      <Button variant="outline" size="sm" onClick={() => setAberto(true)}>
        Transferir ingresso
      </Button>
    )
  }

  const transferir = async () => {
    if (!confirm('O QR atual vai deixar de valer e o novo titular receberá outro por e-mail. Continuar?')) return
    setErro(null)
    setEnviando(true)
    try {
      await api(`/itens/${itemId}/transferir`, { method: 'POST', body: { nome, email } })
      await queryClient.invalidateQueries({ queryKey: ['meus-ingressos'] })
      onConcluido()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao transferir')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <div className="flex w-full flex-col gap-2 border-t border-border pt-3 text-left">
      <p className="text-xs text-muted-foreground">Novo titular (a pessoa recebe o QR por e-mail):</p>
      <Input placeholder="Nome" value={nome} onChange={(e) => setNome(e.target.value)} />
      <Input type="email" placeholder="E-mail" value={email} onChange={(e) => setEmail(e.target.value)} />
      {erro && <p className="text-xs text-destructive">{erro}</p>}
      <Button size="sm" onClick={transferir} disabled={enviando || !nome || !email}>
        {enviando ? 'Transferindo…' : 'Confirmar transferência'}
      </Button>
    </div>
  )
}
