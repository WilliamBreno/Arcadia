import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import { papeisChat, type ChatAtencao, type ChatParticipante } from '@/lib/chat-evento'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'

export function ApoioJurado({ slug }: { slug: string }) {
  const { usuario } = useAuth(), qc = useQueryClient()
  const chave = ['apoio-jurado', slug, usuario?.id]
  const { data, error } = useQuery({ queryKey: chave, queryFn: () => api<{ responsaveis: ChatParticipante[]; pedido: ChatAtencao | null }>(`/eventos/${slug}/apoio-jurado`), refetchInterval: 10000, retry: false })
  const [texto, setTexto] = useState(''), [erro, setErro] = useState(''), [ocupado, setOcupado] = useState(false)
  const pedir = async () => {
    setOcupado(true); setErro('')
    try { await api(`/eventos/${slug}/apoio-jurado`, { method: 'POST', body: { texto } }); setTexto(''); await qc.invalidateQueries({ queryKey: chave }) }
    catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível avisar a equipe.') }
    finally { setOcupado(false) }
  }
  if (error) return <p role="alert" className="mb-6 text-sm text-destructive">Não foi possível carregar o apoio da equipe.</p>
  if (!data) return null
  const pendente = data.pedido && !data.pedido.atendido_em
  return <section className="mb-6 flex flex-col gap-3 rounded-xl border border-border p-4" aria-label="Apoio ao jurado">
    <h2 className="font-semibold">Responsáveis do evento</h2>
    <ul className="flex flex-wrap gap-x-4 gap-y-2 text-sm">{data.responsaveis.map(p => <li key={p.usuario_id}>{p.nome} <span className="text-muted-foreground">· {papeisChat[p.papel] ?? p.papel}</span></li>)}</ul>
    {data.responsaveis.length === 0 && <p className="text-sm text-muted-foreground">Nenhum responsável disponível no momento.</p>}
    {pendente ? <p role="status" className="rounded-lg bg-muted p-3 text-sm">Seu pedido de atenção foi enviado. Aguarde o atendimento da equipe.</p> : <>
      {data.pedido?.atendido_em && <p role="status" className="text-sm">A equipe confirmou o atendimento do seu último pedido.</p>}
      <Label htmlFor="apoio-motivo">Como a equipe pode ajudar? (opcional)</Label>
      <Textarea id="apoio-motivo" maxLength={500} value={texto} onChange={e => setTexto(e.target.value)} rows={2} />
      <Button type="button" disabled={ocupado} onClick={pedir} className="self-start">{ocupado ? 'Avisando…' : 'Preciso de atenção'}</Button>
    </>}
    {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
  </section>
}
