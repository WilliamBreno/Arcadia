import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { AdminNav } from '@/components/admin-nav'
import { SeloVerificado } from '@/components/selo-verificado'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

type OrganizadorAdmin = {
  id: number
  nome_publico: string
  slug: string
  usuario_email: string
  tipo_pessoa: 'pf' | 'pj'
  documento: string
  chave_pix: string
  tipo_chave_pix: string
  eventos: number
  repasses_pagos: number
  verificado_em: string | null
}

// Verificação de organizadores (item 5.4): o admin confere o documento e
// se a chave Pix é do mesmo titular antes de marcar como verificado.
export default function AdminOrganizadores() {
  const queryClient = useQueryClient()
  const [pendentes, setPendentes] = useState(true)
  const [erro, setErro] = useState<string | null>(null)

  const { data, error } = useQuery({
    queryKey: ['admin-organizadores', pendentes],
    queryFn: () => api<OrganizadorAdmin[]>(`/admin/organizadores${pendentes ? '?pendentes=1' : ''}`),
  })

  const definir = async (id: number, verificado: boolean) => {
    setErro(null)
    try {
      await api(`/admin/organizadores/${id}/verificacao`, { method: 'POST', body: { verificado } })
      queryClient.invalidateQueries({ queryKey: ['admin-organizadores'] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao salvar')
    }
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-destructive">Acesso restrito ao admin da plataforma.</p>
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <AdminNav />
      <div className="mb-2 flex items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold text-foreground">Organizadores</h1>
        <select
          className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
          value={pendentes ? 'pendentes' : 'todos'}
          onChange={(e) => setPendentes(e.target.value === 'pendentes')}
        >
          <option value="pendentes">Aguardando verificação</option>
          <option value="todos">Todos</option>
        </select>
      </div>
      <p className="mb-6 text-sm text-muted-foreground">
        Confira se o CPF/CNPJ é válido e se a chave Pix está no mesmo nome. Organizador sem verificação, ou com menos de 3
        repasses pagos, tem 10% das vendas no cartão retidas por 60 dias. Trocar a chave Pix tira a verificação.
      </p>
      {erro && <p className="mb-3 text-sm text-destructive">{erro}</p>}
      {data?.length === 0 && <p className="text-muted-foreground">Nenhum organizador aguardando verificação.</p>}
      <div className="flex flex-col gap-3">
        {data?.map((o) => (
          <Card key={o.id}>
            <CardContent className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <p className="flex flex-wrap items-center gap-2 font-medium text-foreground">
                  {o.nome_publico}
                  {o.verificado_em && <SeloVerificado />}
                </p>
                <p className="text-xs text-muted-foreground">{o.usuario_email}</p>
                <p className="text-xs text-muted-foreground">
                  {o.tipo_pessoa === 'pj' ? 'CNPJ' : 'CPF'} <span className="font-mono">{o.documento}</span> · Pix ({o.tipo_chave_pix || '—'}):{' '}
                  <span className="font-mono break-all">{o.chave_pix || 'não cadastrada'}</span>
                </p>
                <p className="text-xs text-muted-foreground">
                  {o.eventos} evento(s) · {o.repasses_pagos} repasse(s) pago(s)
                  {o.verificado_em && ` · verificado em ${new Date(o.verificado_em).toLocaleDateString('pt-BR')}`}
                </p>
              </div>
              {o.verificado_em ? (
                <Button size="sm" variant="outline" onClick={() => definir(o.id, false)}>
                  Remover verificação
                </Button>
              ) : (
                <Button size="sm" onClick={() => definir(o.id, true)} disabled={!o.chave_pix}>
                  Marcar como verificado
                </Button>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
