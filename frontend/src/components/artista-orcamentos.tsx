import { useInfiniteQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { listarOrcamentos, statusOrcamento, diaOrcamento } from '@/lib/contratacao'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/use-auth'

export function ArtistaOrcamentos({ id }: { id?: number }) {
  const { usuario } = useAuth()
  const { data: paginas, error, isLoading, hasNextPage, fetchNextPage, isFetchingNextPage } = useInfiniteQuery({ queryKey: ['orcamentos-artista', id ?? 'cliente', usuario?.id], initialPageParam: 1, queryFn: ({ pageParam }) => listarOrcamentos(id, pageParam), getNextPageParam: (ultima, todas) => ultima.length === 200 ? todas.length + 1 : undefined, refetchInterval: 15000 })
  const data = paginas?.pages.flat()
  return <Card><CardHeader><CardTitle>{id ? 'Pedidos de contratação' : 'Meus pedidos de orçamento'}</CardTitle></CardHeader><CardContent>
    {isLoading && <p role="status">Carregando…</p>}{error && <p role="alert">Não foi possível carregar os pedidos.</p>}
    {data?.length === 0 && <p className="text-sm text-muted-foreground">Nenhum pedido por enquanto.</p>}
    <ul className="flex flex-col divide-y divide-border">{data?.map(p => <li key={p.id}>
      <Link to={`/orcamento-artista/${p.id}`} className="flex flex-col gap-1 py-3 hover:underline">
        <span className="font-medium">{id ? `${p.cliente_nome} · ${p.tipo_evento}` : p.grupo_nome}</span>
        <span className="text-sm text-muted-foreground">{diaOrcamento(p.inicio)} a {diaOrcamento(p.fim)} · {p.destino?.nome}/{p.destino?.uf}</span>
        <span className="text-xs font-medium">{statusOrcamento[p.status] ?? p.status}</span>
      </Link>
    </li>)}</ul>
    {hasNextPage && <Button type="button" variant="outline" disabled={isFetchingNextPage} onClick={() => fetchNextPage()}>Carregar mais pedidos</Button>}
  </CardContent></Card>
}
