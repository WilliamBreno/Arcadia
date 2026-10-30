import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { VoltarLink } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Membro = { usuario_id: number; nome: string; email: string; papel: string }

// Item 9.8: funções da equipe.
const funcoes = [
  { valor: 'gestor', rotulo: 'Gestor', descricao: 'Cria e edita eventos, ingressos, cupons, participantes e faz check-in. Não vê dinheiro.' },
  { valor: 'financeiro', rotulo: 'Financeiro', descricao: 'Vê vendas, financeiro dos eventos, repasses e relatórios. Não edita nada.' },
  { valor: 'leitura', rotulo: 'Leitura', descricao: 'Só consulta os eventos. Não edita e não vê dinheiro.' },
  { valor: 'checkin', rotulo: 'Check-in', descricao: 'Só a portaria e a venda na porta de todos os eventos.' },
]
const rotuloFuncao = (f: string) => funcoes.find((x) => x.valor === f)?.rotulo ?? f

export default function Equipe() {
  const queryClient = useQueryClient()
  const [email, setEmail] = useState('')
  const [funcao, setFuncao] = useState('gestor')
  const [erro, setErro] = useState<string | null>(null)
  const { data, error } = useQuery({ queryKey: ['org-equipe'], queryFn: () => api<Membro[]>('/org/equipe') })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-equipe'] })

  const executar = async (acao: () => Promise<unknown>) => {
    setErro(null)
    try {
      await acao()
      invalidar()
      return true
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
      return false
    }
  }

  const adicionar = async () => {
    if (await executar(() => api('/org/equipe', { method: 'POST', body: { email, funcao } }))) setEmail('')
  }

  const remover = (id: number) => {
    if (!confirm('Remover esta pessoa da equipe?')) return
    executar(() => api(`/org/equipe/${id}`, { method: 'DELETE' }))
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-muted-foreground">Só o dono do perfil de organizador gerencia a equipe.</p>
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar ao painel</VoltarLink>
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Equipe</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Quem entra precisa ter conta na plataforma. Só você, o dono, mexe em perfil e Pix, na equipe, em coprodução e no cancelamento de eventos.
      </p>
      <dl className="mb-6 grid gap-2 rounded-lg border border-border p-3 text-sm sm:grid-cols-2">
        {funcoes.map((f) => (
          <div key={f.valor}>
            <dt className="font-medium text-foreground">{f.rotulo}</dt>
            <dd className="text-muted-foreground">{f.descricao}</dd>
          </div>
        ))}
      </dl>
      <div className="flex flex-col gap-2">
        {data?.length === 0 && <p className="text-sm text-muted-foreground">Ninguém na equipe ainda.</p>}
        {data?.map((m) => (
          <Card key={m.usuario_id}>
            <CardContent className="flex flex-wrap items-center justify-between gap-3 py-3">
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground">{m.nome}</p>
                <p className="truncate text-xs text-muted-foreground">{m.email}</p>
              </div>
              <div className="flex items-center gap-2">
                <select
                  aria-label={`Função de ${m.nome}`}
                  className="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground"
                  value={m.papel}
                  onChange={(e) => executar(() => api(`/org/equipe/${m.usuario_id}`, { method: 'PUT', body: { funcao: e.target.value } }))}
                >
                  {funcoes.map((f) => (
                    <option key={f.valor} value={f.valor}>
                      {f.rotulo}
                    </option>
                  ))}
                </select>
                <Button variant="destructive" size="sm" onClick={() => remover(m.usuario_id)}>
                  Remover
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:items-end">
        <div className="flex flex-1 flex-col gap-1.5">
          <Label htmlFor="email-membro">E-mail da pessoa</Label>
          <Input id="email-membro" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="funcao-membro">Função</Label>
          <select
            id="funcao-membro"
            className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground"
            value={funcao}
            onChange={(e) => setFuncao(e.target.value)}
          >
            {funcoes.map((f) => (
              <option key={f.valor} value={f.valor}>
                {rotuloFuncao(f.valor)}
              </option>
            ))}
          </select>
        </div>
        <Button onClick={adicionar} disabled={!email}>
          Adicionar
        </Button>
      </div>
      {erro && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {erro}
        </p>
      )}
    </main>
  )
}
