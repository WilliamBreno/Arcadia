import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import type { Local } from '@/lib/organizador'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export default function Locais() {
  const queryClient = useQueryClient()
  const { data: locais, isLoading, error } = useQuery({
    queryKey: ['org-locais'],
    queryFn: () => api<Local[]>('/org/locais'),
  })

  const excluir = async (id: number) => {
    if (!confirm('Excluir este local?')) return
    await api(`/org/locais/${id}`, { method: 'DELETE' })
    queryClient.invalidateQueries({ queryKey: ['org-locais'] })
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Locais</h1>
        <Link to="/organizador/locais/novo" className={buttonVariants()}>
          Novo local
        </Link>
      </div>

      {isLoading && <p className="text-muted-foreground">Carregando…</p>}
      {error instanceof ApiError && error.status === 404 && (
        <p className="text-muted-foreground">
          Crie seu <Link to="/organizador/perfil" className="underline">perfil de organizador</Link> primeiro.
        </p>
      )}
      {locais?.length === 0 && <p className="text-muted-foreground">Nenhum local cadastrado ainda.</p>}

      <div className="flex flex-col gap-3">
        {locais?.map((local) => (
          <Card key={local.id}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <p className="font-medium text-foreground">{local.nome}</p>
                <p className="text-sm text-muted-foreground">
                  {local.cidade} — {local.uf}
                  {local.capacidade ? ` · capacidade ${local.capacidade}` : ''}
                </p>
              </div>
              <div className="flex gap-2">
                <Link
                  to={`/organizador/locais/${local.id}`}
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                >
                  Editar
                </Link>
                <Button variant="destructive" size="sm" onClick={() => excluir(local.id)}>
                  Excluir
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
