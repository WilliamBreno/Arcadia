import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { diaOrcamento, lerCentavos, statusOrcamento, type ConversaArtista } from '@/lib/contratacao'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Carregando } from '@/components/carregando'
import { useAuth } from '@/hooks/use-auth'

export default function OrcamentoArtista() {
  const { id } = useParams(), qc = useQueryClient()
  const { usuario } = useAuth()
  const chave = ['conversa-artista', id, usuario?.id]
  const { data: paginas, error, isLoading, hasNextPage, fetchNextPage, isFetchingNextPage } = useInfiniteQuery({ queryKey: chave, initialPageParam: 0, queryFn: ({ pageParam }) => api<ConversaArtista>(`/me/artistas-orcamentos/${id}?antes=${pageParam}`), getNextPageParam: (ultima) => ultima.mensagens.length === 200 ? ultima.mensagens[0].id : undefined, refetchInterval: 5000, retry: false })
  const data = paginas?.pages[0]
  const mensagens = paginas?.pages.flatMap(p => p.mensagens).sort((a, b) => a.id - b.id) ?? []
  const [texto, setTexto] = useState(''), [valor, setValor] = useState(''), [condicoes, setCondicoes] = useState(''), [motivo, setMotivo] = useState('')
  const [erro, setErro] = useState(''), [ocupado, setOcupado] = useState(false)
  const atualizar = async () => { await qc.invalidateQueries({ queryKey: chave }); qc.invalidateQueries({ queryKey: ['orcamentos-artista'] }); qc.invalidateQueries({ queryKey: ['agenda-artista'] }) }
  const enviar = async (e: React.FormEvent) => {
    e.preventDefault(); setErro(''); setOcupado(true)
    try { await api(`/me/artistas-orcamentos/${id}/mensagens`, { method: 'POST', body: { texto } }); setTexto(''); await atualizar() }
    catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível enviar.') } finally { setOcupado(false) }
  }
  const agir = async (acao: string) => {
    setErro('')
    const centavos = acao === 'propor' ? lerCentavos(valor) : null
    if (acao === 'propor' && (centavos === null || !condicoes.trim())) { setErro('Informe o valor em reais e as condições da proposta.'); return }
    if ((acao === 'cancelar' || acao === 'recusar') && !motivo.trim()) { setErro('Informe o motivo.'); return }
    if (acao === 'aceitar' && !window.confirm('Aceitar esta proposta e reservar a data? O pagamento será combinado diretamente com o artista.')) return
    setOcupado(true)
    try { await api(`/me/artistas-orcamentos/${id}/acoes`, { method: 'POST', body: { acao, valor_centavos: centavos, texto: condicoes, motivo, versao: data?.pedido.versao } }); await atualizar() }
    catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir.') } finally { setOcupado(false) }
  }
  if (isLoading) return <Carregando />
  if (error || !data) return <p role="alert" className="p-8 text-center">{error instanceof ApiError ? error.message : 'Pedido indisponível.'}</p>
  const p = data.pedido, fechado = ['cancelado', 'recusado', 'realizado'].includes(p.status)
  return <main className="mx-auto flex max-w-3xl flex-col gap-5 px-4 py-10">
    <Link to={p.sou_cliente ? '/meus-orcamentos' : `/meus-artistas/${p.grupo_id}`} className="text-sm underline">Voltar aos pedidos</Link>
    <header><h1 className="text-2xl font-bold">{p.grupo_nome}</h1><p className="mt-2 text-sm text-muted-foreground">Pedido #{p.id} · {statusOrcamento[p.status]}</p></header>
    <section className="rounded-xl border border-border p-4 text-sm">
      <h2 className="mb-2 font-semibold">{p.tipo_evento} · {p.cliente_nome}</h2>
      <p>{diaOrcamento(p.inicio)} a {diaOrcamento(p.fim)} · {p.destino?.nome}/{p.destino?.uf} · {p.duracao_minutos} minutos</p>
      {p.observacoes && <p className="mt-3 whitespace-pre-wrap break-words">{p.observacoes}</p>}
      <p className="mt-3 text-muted-foreground">A Evve não recebe o pagamento desta contratação. Combine passagem, hospedagem, alimentação e pagamento com a outra parte.</p>
    </section>
    {p.proposta_centavos !== null && <section className="rounded-xl bg-muted p-4">
      <h2 className="font-semibold">Proposta {p.versao} · {formatarCentavos(p.proposta_centavos)}</h2>
      <p className="mt-2 whitespace-pre-wrap break-words text-sm">{p.proposta_texto}</p>
      {p.sou_cliente && p.status === 'proposta_enviada' && <Button className="mt-4" disabled={ocupado} onClick={() => agir('aceitar')}>Aceitar proposta e reservar data</Button>}
    </section>}
    {p.motivo && <p className="rounded-xl bg-muted p-4 text-sm">Motivo: {p.motivo}</p>}
    {!p.sou_cliente && !fechado && p.status !== 'aceito' && <form onSubmit={e => { e.preventDefault(); agir('propor') }} className="flex flex-col gap-3 rounded-xl border border-border p-4">
      <h2 className="font-semibold">{p.status === 'proposta_enviada' ? 'Substituir proposta' : 'Enviar proposta'}</h2>
      <Label htmlFor="proposta-valor">Valor total em reais</Label><Input id="proposta-valor" inputMode="decimal" required value={valor} onChange={e => setValor(e.target.value)} placeholder="1500,00" />
      <Label htmlFor="proposta-condicoes">Condições, inclusões e pagamento</Label><Textarea id="proposta-condicoes" maxLength={3000} required value={condicoes} onChange={e => setCondicoes(e.target.value)} />
      <Button type="submit" className="self-start" disabled={ocupado}>Enviar proposta</Button>
    </form>}
    <section aria-label="Conversa" className="flex flex-col gap-3">
      <h2 className="font-semibold">Mensagens</h2>
      {hasNextPage && <Button type="button" variant="outline" disabled={isFetchingNextPage} onClick={() => fetchNextPage()}>Carregar mensagens anteriores</Button>}
      {mensagens.length === 0 && <p className="text-sm text-muted-foreground">Envie uma mensagem para combinar os detalhes.</p>}
      <ul className="flex flex-col gap-3">{mensagens.map(m => <li key={m.id} className={`max-w-full rounded-xl border border-border p-3 sm:max-w-xl ${m.minha ? 'self-end bg-muted' : 'self-start'}`}>
        <p className="text-xs text-muted-foreground">{m.autor}{m.minha ? ' (você)' : ''} · {new Date(m.criado_em).toLocaleString('pt-BR')}</p><p className="mt-1 whitespace-pre-wrap break-words text-sm">{m.texto}</p>
      </li>)}</ul>
      {!fechado && <form onSubmit={enviar} className="flex flex-col gap-2"><Label htmlFor="mensagem-artista">Sua mensagem</Label><Textarea id="mensagem-artista" maxLength={3000} required value={texto} onChange={e => setTexto(e.target.value)} /><Button type="submit" disabled={ocupado || !texto.trim()} className="self-start">Enviar mensagem</Button></form>}
    </section>
    {!fechado && <section className="flex flex-col gap-3 rounded-xl border border-border p-4">
      <h2 className="font-semibold">Gerenciar pedido</h2><Label htmlFor="motivo-pedido">Motivo para cancelar ou recusar</Label><Textarea id="motivo-pedido" maxLength={1000} value={motivo} onChange={e => setMotivo(e.target.value)} />
      <div className="evve-actions flex flex-wrap gap-2"><Button variant="outline" disabled={ocupado} onClick={() => agir('cancelar')}>Cancelar pedido</Button>
        {!p.sou_cliente && p.status !== 'aceito' && <Button variant="outline" disabled={ocupado} onClick={() => agir('recusar')}>Recusar pedido</Button>}
        {!p.sou_cliente && p.status === 'aceito' && <Button variant="outline" disabled={ocupado} onClick={() => agir('realizado')}>Marcar como realizado</Button>}
      </div><p className="text-xs text-muted-foreground">Cancelar uma contratação confirmada libera a data reservada. Acertos de pagamento são feitos diretamente entre as partes.</p>
    </section>}
    {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
  </main>
}
