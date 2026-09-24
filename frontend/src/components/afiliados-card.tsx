import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

type Afiliado = { id: number; nome: string; codigo: string; ativo: boolean; pedidos: number; itens: number; receita_centavos: number }

export function AfiliadosCard({ eventoId, slug }: { eventoId: number; slug: string }) {
  const queryClient = useQueryClient()
  const [nome, setNome] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const { data } = useQuery({ queryKey: ['org-afiliados', eventoId], queryFn: () => api<Afiliado[]>(`/org/eventos/${eventoId}/afiliados`) })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-afiliados', eventoId] })

  const criar = async () => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/afiliados`, { method: 'POST', body: { nome } })
      setNome('')
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar divulgador')
    }
  }

  const desativar = async (id: number) => {
    await api(`/org/eventos/${eventoId}/afiliados/${id}`, { method: 'DELETE' })
    invalidar()
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Divulgadores (links de afiliado)</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-xs text-muted-foreground">
          Cada divulgador ganha um link; as vendas pagas feitas por ele aparecem aqui. Só atribuição e estatística — comissão não é calculada nem paga
          pela plataforma.
        </p>
        {data?.map((a) => (
          <div key={a.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
            <div className="min-w-0">
              <p className="font-medium text-foreground">
                {a.nome} {!a.ativo && <span className="text-xs text-muted-foreground">(desativado)</span>}
              </p>
              <p className="truncate font-mono text-xs text-muted-foreground">{`${window.location.origin}/e/${slug}?ref=${a.codigo}`}</p>
              <p className="text-xs text-muted-foreground">
                {a.pedidos} pedido(s) · {a.itens} ingresso(s) · {formatarCentavos(a.receita_centavos)}
              </p>
            </div>
            {a.ativo && (
              <Button variant="destructive" size="sm" onClick={() => desativar(a.id)}>
                Desativar
              </Button>
            )}
          </div>
        ))}
        <div className="flex gap-2 border-t border-border pt-3">
          <Input placeholder="Nome do divulgador" value={nome} onChange={(e) => setNome(e.target.value)} />
          <Button onClick={criar} disabled={!nome}>
            Criar link
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}
