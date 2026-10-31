import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { ApiError } from '@/lib/api'
import { decidirSelo, listarPedidosSelo } from '@/lib/artistas'
import { AdminNav } from '@/components/admin-nav'
import { ArtistaViagemConfig } from '@/components/artista-viagem-config'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

export default function AdminArtistas() {
  const queryClient = useQueryClient()
  const chave = ['admin-artistas-selo']
  const { data, isLoading, error } = useQuery({ queryKey: chave, queryFn: listarPedidosSelo, retry: false })
  const [enviando, setEnviando] = useState<number | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)

  const decidir = async (grupoId: number, aprovar: boolean, instagram: string) => {
    setEnviando(grupoId)
    setErro(null)
    setAviso(null)
    try {
      await decidirSelo(grupoId, aprovar, instagram)
      setAviso(aprovar ? 'Selo aprovado.' : 'Pedido recusado.')
      await queryClient.invalidateQueries({ queryKey: chave })
    } catch (e) {
      setErro(e instanceof ApiError
        ? e.status === 403 ? 'Acesso restrito ao admin da plataforma.' : e.message
        : 'Não foi possível decidir o pedido.')
    } finally {
      setEnviando(null)
    }
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <AdminNav />
      <ArtistaViagemConfig />
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Selo do Instagram</h1>
      <p className="mb-6 text-sm text-muted-foreground">Abra o perfil no Instagram, confira se o código está na bio e aprove o selo.</p>
      {isLoading && <p role="status" className="text-muted-foreground">Carregando pedidos…</p>}
      {error && (
        <p role="alert" className="text-destructive">
          {error instanceof ApiError
            ? error.status === 403 ? 'Acesso restrito ao admin da plataforma.' : error.message
            : 'Não foi possível carregar os pedidos.'}
        </p>
      )}
      {erro && <p role="alert" className="mb-4 text-sm text-destructive">{erro}</p>}
      {aviso && <p role="status" className="mb-4 text-sm text-muted-foreground">{aviso}</p>}
      {!error && data?.length === 0 && <p className="text-muted-foreground">Nenhum pedido agora.</p>}
      {!error && (
        <div className="flex flex-col gap-3">
          {data?.map((pedido) => (
            <Card key={pedido.grupo_id}>
              <CardContent className="flex flex-col gap-3 py-4">
                <Link to={`/artista/${pedido.slug}`} className="break-words font-medium text-foreground underline-offset-4 hover:underline">
                  {pedido.nome}
                </Link>
                <a href={`https://www.instagram.com/${pedido.instagram.replace(/^@/, '')}/`} target="_blank" rel="noopener noreferrer" className="break-all text-sm text-primary underline-offset-4 hover:underline">
                  @{pedido.instagram.replace(/^@/, '')}
                </a>
                <div className="rounded-lg bg-muted p-3 text-foreground">
                  <p className="text-xs text-muted-foreground">Código para conferir na bio</p>
                  <code className="break-all font-mono text-xl font-bold tracking-wider">{pedido.codigo}</code>
                </div>
                <p className="text-sm text-muted-foreground">
                  Pedido em <time dateTime={pedido.pedido_em}>{new Date(pedido.pedido_em).toLocaleString('pt-BR')}</time>
                </p>
                <div className="flex flex-wrap gap-2">
                  <Button disabled={enviando !== null} onClick={() => decidir(pedido.grupo_id, true, pedido.instagram)}>Aprovar selo</Button>
                  <Button variant="outline" disabled={enviando !== null} onClick={() => decidir(pedido.grupo_id, false, pedido.instagram)}>Recusar</Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </main>
  )
}
