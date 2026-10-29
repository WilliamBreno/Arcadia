import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { AdminNav } from '@/components/admin-nav'
import { Estrelas } from '@/components/estrelas'
import { Button } from '@/components/ui/button'

type AvaliacaoAdmin = {
  id: number
  evento_titulo: string
  evento_slug: string
  usuario_email: string
  nota: number
  comentario: string
  ocultada: boolean
  criado_em: string
}

// Moderação das avaliações do público (item 6.5). O admin oculta o que fere
// as regras (ofensa, dado pessoal, propaganda); nada é apagado.
export default function AdminAvaliacoes() {
  const queryClient = useQueryClient()
  const [erro, setErro] = useState<string | null>(null)
  const [soComentario, setSoComentario] = useState(true)
  const { data, error, isLoading } = useQuery({
    queryKey: ['admin-avaliacoes'],
    queryFn: () => api<AvaliacaoAdmin[]>('/admin/avaliacoes'),
  })

  const alternar = async (a: AvaliacaoAdmin) => {
    setErro(null)
    try {
      await api(`/admin/avaliacoes/${a.id}/ocultar`, { method: 'POST', body: { ocultar: !a.ocultada } })
      queryClient.invalidateQueries({ queryKey: ['admin-avaliacoes'] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao salvar')
    }
  }

  const lista = (data ?? []).filter((a) => !soComentario || a.comentario)

  return (
    <main className="mx-auto max-w-5xl px-4 py-10">
      <AdminNav />
      <h1 className="text-2xl font-semibold text-foreground">Avaliações</h1>
      <p className="mt-1 text-sm text-muted-foreground">
        Oculte avaliações com ofensa, dado pessoal ou propaganda. Ocultar tira da página e da média; dá para reexibir.
      </p>
      <label className="mt-4 flex items-center gap-2 text-sm text-foreground">
        <input type="checkbox" checked={soComentario} onChange={(e) => setSoComentario(e.target.checked)} />
        Só as que têm comentário
      </label>
      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}
      {error && <p className="mt-4 text-destructive">Não foi possível carregar.</p>}
      {isLoading && <p className="mt-4 text-muted-foreground">Carregando…</p>}
      {data && lista.length === 0 && <p className="mt-6 rounded-xl border border-border p-6 text-muted-foreground">Nenhuma avaliação.</p>}
      <ul className="mt-4 flex flex-col divide-y divide-border">
        {lista.map((a) => (
          <li key={a.id} className={`flex flex-col gap-2 py-4 sm:flex-row sm:items-start sm:justify-between ${a.ocultada ? 'opacity-60' : ''}`}>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2 text-sm">
                <Estrelas nota={a.nota} tamanho={14} />
                <Link to={`/e/${a.evento_slug}`} className="font-medium text-foreground hover:underline">
                  {a.evento_titulo}
                </Link>
                {a.ocultada && <span className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">ocultada</span>}
              </div>
              <p className="mt-0.5 break-all text-xs text-muted-foreground">
                {a.usuario_email} · {new Date(a.criado_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })}
              </p>
              {a.comentario && <p className="mt-1 whitespace-pre-line text-pretty text-foreground">{a.comentario}</p>}
            </div>
            <Button type="button" variant="outline" size="sm" className="shrink-0 self-start" onClick={() => alternar(a)}>
              {a.ocultada ? 'Reexibir' : 'Ocultar'}
            </Button>
          </li>
        ))}
      </ul>
    </main>
  )
}
