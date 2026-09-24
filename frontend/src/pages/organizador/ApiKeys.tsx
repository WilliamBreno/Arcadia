import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

type Chave = { id: number; nome: string; prefixo: string; ativo: boolean; criado_em: string; ultimo_uso_em: string | null }

export default function ApiKeys() {
  const queryClient = useQueryClient()
  const [nome, setNome] = useState('')
  const [novaChave, setNovaChave] = useState<string | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const { data, error } = useQuery({ queryKey: ['org-api-keys'], queryFn: () => api<Chave[]>('/org/api-keys') })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-api-keys'] })

  const criar = async () => {
    setErro(null)
    try {
      const r = await api<{ chave: string }>('/org/api-keys', { method: 'POST', body: { nome } })
      setNovaChave(r.chave)
      setNome('')
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar chave')
    }
  }

  const revogar = async (id: number) => {
    if (!confirm('Revogar esta chave? Integrações que a usam vão parar de funcionar.')) return
    await api(`/org/api-keys/${id}`, { method: 'DELETE' })
    invalidar()
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-muted-foreground">Só o dono do perfil de organizador gerencia chaves de API.</p>
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <h1 className="mb-2 text-2xl font-semibold text-foreground">API para integrações</h1>
      <p className="mb-4 text-sm text-muted-foreground">
        Somente leitura, escopo dos seus eventos. Use a chave só em servidores (nunca em site ou app). Envie no header <code>X-API-Key</code>.
      </p>
      <pre className="mb-6 overflow-x-auto rounded-lg border border-border p-3 text-xs text-foreground">
        {`GET /api/public/v1/eventos
GET /api/public/v1/eventos/:id
GET /api/public/v1/eventos/:id/ingressos     # tipos, ocupados, disponiveis
GET /api/public/v1/eventos/:id/participantes # nome, e-mail, codigo, status (dados pessoais)
GET /api/public/v1/eventos/:id/resumo        # contadores de check-in`}
      </pre>

      {novaChave && (
        <Card className="mb-4 border-primary">
          <CardContent className="py-4">
            <p className="text-sm font-medium text-foreground">Copie agora — esta chave não será exibida de novo:</p>
            <p className="mt-1 font-mono text-xs break-all text-foreground">{novaChave}</p>
          </CardContent>
        </Card>
      )}

      <div className="flex flex-col gap-2">
        {data?.length === 0 && <p className="text-sm text-muted-foreground">Nenhuma chave criada.</p>}
        {data?.map((k) => (
          <Card key={k.id}>
            <CardContent className="flex items-center justify-between py-3">
              <div>
                <p className="text-sm font-medium text-foreground">
                  {k.nome} {!k.ativo && <span className="text-xs text-muted-foreground">(revogada)</span>}
                </p>
                <p className="font-mono text-xs text-muted-foreground">
                  {k.prefixo}… · último uso: {k.ultimo_uso_em ? new Date(k.ultimo_uso_em).toLocaleString('pt-BR') : 'nunca'}
                </p>
              </div>
              {k.ativo && (
                <Button variant="destructive" size="sm" onClick={() => revogar(k.id)}>
                  Revogar
                </Button>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="mt-6 flex gap-2">
        <Input placeholder="Nome da chave (ex.: ERP)" value={nome} onChange={(e) => setNome(e.target.value)} />
        <Button onClick={criar} disabled={!nome}>
          Criar chave
        </Button>
      </div>
      {erro && <p className="mt-2 text-sm text-destructive">{erro}</p>}
    </main>
  )
}
