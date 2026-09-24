import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

type Membro = { usuario_id: number; nome: string; email: string; papel: string }

export default function Equipe() {
  const queryClient = useQueryClient()
  const [email, setEmail] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const { data, error } = useQuery({ queryKey: ['org-equipe'], queryFn: () => api<Membro[]>('/org/equipe') })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-equipe'] })

  const adicionar = async () => {
    setErro(null)
    try {
      await api('/org/equipe', { method: 'POST', body: { email } })
      setEmail('')
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao adicionar')
    }
  }

  const remover = async (id: number) => {
    if (!confirm('Remover esta pessoa da equipe?')) return
    await api(`/org/equipe/${id}`, { method: 'DELETE' })
    invalidar()
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-muted-foreground">Só o dono do perfil de organizador gerencia a equipe.</p>
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Equipe</h1>
      <p className="mb-6 text-sm text-muted-foreground">
        Gestores (precisam ter conta na plataforma) criam e editam eventos, ingressos, cupons, participantes, convites, staff e fazem check-in.
        Eles <strong>não</strong> veem perfil/Pix, financeiro, repasses nem relatórios de receita, e não podem cancelar evento.
      </p>
      <div className="flex flex-col gap-2">
        {data?.length === 0 && <p className="text-sm text-muted-foreground">Ninguém na equipe ainda.</p>}
        {data?.map((m) => (
          <Card key={m.usuario_id}>
            <CardContent className="flex items-center justify-between py-3">
              <div>
                <p className="text-sm font-medium text-foreground">{m.nome}</p>
                <p className="text-xs text-muted-foreground">
                  {m.email} · {m.papel}
                </p>
              </div>
              <Button variant="destructive" size="sm" onClick={() => remover(m.usuario_id)}>
                Remover
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="mt-6 flex gap-2">
        <Input type="email" placeholder="E-mail da pessoa" value={email} onChange={(e) => setEmail(e.target.value)} />
        <Button onClick={adicionar} disabled={!email}>
          Adicionar
        </Button>
      </div>
      {erro && <p className="mt-2 text-sm text-destructive">{erro}</p>}
    </main>
  )
}
