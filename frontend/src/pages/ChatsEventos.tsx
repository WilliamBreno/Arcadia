import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, ApiError } from '@/lib/api'
import type { ChatCanal } from '@/lib/chat-evento'
import { useAuth } from '@/hooks/use-auth'
import { VoltarLink } from '@/components/voltar-link'

export default function ChatsEventos() {
  const { usuario } = useAuth()
  const { data, isLoading, error } = useQuery({ queryKey: ['chats-eventos', usuario?.id], queryFn: () => api<ChatCanal[]>('/me/chats-eventos'), refetchInterval: 15000, retry: false })
  return <main className="mx-auto max-w-2xl px-4 py-10">
    <VoltarLink to="/meus-eventos">Meus eventos</VoltarLink>
    <h1 className="mb-2 text-2xl font-semibold">Conversas da equipe</h1>
    <p className="mb-6 text-sm text-muted-foreground">Um chat privado por evento, para organização, equipe e coprodutores. Jurados pedem apoio pela área de avaliação.</p>
    {isLoading && <p role="status">Carregando conversas…</p>}
    {error && <p role="alert" className="text-destructive">{error instanceof ApiError ? error.message : 'Não foi possível carregar as conversas.'}</p>}
    {!error && data?.length === 0 && <p className="text-muted-foreground">Você ainda não faz parte da equipe de nenhum evento.</p>}
    <ul className="flex flex-col gap-3">{!error && data?.map(c => <li key={c.evento_id}><Link to={`/chat-evento/${c.evento_id}`} className="flex items-center justify-between gap-3 rounded-xl border border-border p-4 hover:bg-muted"><span className="min-w-0 break-words font-medium">{c.titulo}</span>{c.nao_lidas > 0 && <span className="shrink-0 rounded-full bg-primary px-2 py-1 text-xs text-primary-foreground">{c.nao_lidas} não lidas</span>}</Link></li>)}</ul>
  </main>
}
