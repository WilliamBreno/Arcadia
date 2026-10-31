import { useEffect, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import { baseChat, enviarChat, fotoChat, papeisChat, type ChatMensagem, type ChatPagina } from '@/lib/chat-evento'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { VoltarLink } from '@/components/voltar-link'

function FotoPrivada({ eventoId, mensagemId }: { eventoId: number; mensagemId: number }) {
  const { usuario } = useAuth()
  const [src, setSrc] = useState(''), [erro, setErro] = useState(false)
  useEffect(() => {
    const abort = new AbortController(); let url = ''
    fotoChat(eventoId, mensagemId, abort.signal).then(blob => { if (!abort.signal.aborted) { url = URL.createObjectURL(blob); setSrc(url) } }).catch(() => { if (!abort.signal.aborted) setErro(true) })
    return () => { abort.abort(); if (url) URL.revokeObjectURL(url) }
  }, [eventoId, mensagemId, usuario?.id])
  if (erro) return <p className="text-xs text-muted-foreground">Foto indisponível. Atualize a conversa para tentar novamente.</p>
  return src ? <a href={src} target="_blank" rel="noreferrer"><img src={src} alt="Foto enviada à equipe" className="mt-2 max-h-64 max-w-full rounded-lg object-contain" /></a> : <p className="text-xs text-muted-foreground">Carregando foto…</p>
}

export default function ChatEvento() { const { id } = useParams(); const { usuario } = useAuth(); return <Conversa key={`${id}-${usuario?.id}`} id={Number(id)} /> }
function Conversa({ id }: { id: number }) {
  const { usuario } = useAuth(), qc = useQueryClient()
  const chave = ['chat-evento', id, usuario?.id]
  const { data, isLoading, error } = useQuery({ queryKey: chave, queryFn: async () => {
    const atual = await api<ChatPagina>(baseChat(id))
    const cache = qc.getQueryData<ChatPagina & { pode_carregar: boolean }>(chave)
    const novas = [...atual.mensagens]
    // Preserva o histórico já carregado e busca mensagens que chegaram entre consultas.
    let lote = atual.mensagens
    const anterior = cache?.mensagens[0]?.id
    while (anterior && lote.length === 50 && lote.every(m => m.id > anterior)) {
      const pagina = await api<ChatPagina>(`${baseChat(id)}?antes=${lote[lote.length - 1].id}`)
      lote = pagina.mensagens; novas.push(...lote)
    }
    return { ...atual, pode_carregar: cache?.pode_carregar ?? atual.mensagens.length === 50, mensagens: [...new Map([...(cache?.mensagens ?? []), ...novas].map(m => [m.id, m])).values()].sort((a, b) => b.id - a.id) }
  }, refetchInterval: 5000, retry: false })
  const [anteriores, setAnteriores] = useState<ChatMensagem[]>([]), [texto, setTexto] = useState(''), [foto, setFoto] = useState<File | null>(null)
  const [erro, setErro] = useState(''), [ocupado, setOcupado] = useState(false), [carregandoAntigas, setCarregandoAntigas] = useState(false), [fimHistorico, setFimHistorico] = useState(false)
  const [pertoFim, setPertoFim] = useState(true)
  const lista = useRef<HTMLDivElement>(null), arquivo = useRef<HTMLInputElement>(null), tokenEnvio = useRef(crypto.randomUUID())
  const ultimoLido = useRef(0)
  const ultimo = data?.mensagens[0]?.id ?? 0
  useEffect(() => {
    if (!data || error || !pertoFim) return
    const marcar = () => {
      if (document.visibilityState !== 'visible' || !document.hasFocus() || ultimo <= Math.max(data.lido_ate, ultimoLido.current)) return
      ultimoLido.current = ultimo
      api(`${baseChat(id)}/lido`, { method: 'POST', body: { ate: ultimo } }).catch(() => { ultimoLido.current = 0 })
    }
    marcar(); window.addEventListener('focus', marcar); document.addEventListener('visibilitychange', marcar)
    return () => { window.removeEventListener('focus', marcar); document.removeEventListener('visibilitychange', marcar) }
  }, [data, error, id, pertoFim, ultimo])
  useEffect(() => { if (pertoFim && lista.current) lista.current.scrollTop = lista.current.scrollHeight }, [ultimo, pertoFim])
  const executar = async (f: () => Promise<unknown>) => {
    setErro(''); setOcupado(true)
    try { await f(); await qc.invalidateQueries({ queryKey: chave }); qc.invalidateQueries({ queryKey: ['chats-eventos', usuario?.id] }) }
    catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir. Tente novamente.') }
    finally { setOcupado(false) }
  }
  const enviar = (e: React.FormEvent) => {
    e.preventDefault()
    executar(async () => { await enviarChat(id, texto, tokenEnvio.current, foto); setTexto(''); setFoto(null); if (arquivo.current) arquivo.current.value = ''; tokenEnvio.current = crypto.randomUUID(); setPertoFim(true) })
  }
  const antigas = async () => {
    const minimo = Math.min(...(anteriores.length ? anteriores : data!.mensagens).map(m => m.id))
    setCarregandoAntigas(true); setErro('')
    try { const r = await api<ChatPagina>(`${baseChat(id)}?antes=${minimo}`); setAnteriores(prev => [...prev, ...r.mensagens]); setFimHistorico(r.mensagens.length < 50) }
    catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível carregar o histórico.') }
    finally { setCarregandoAntigas(false) }
  }
  if (isLoading) return <p className="p-8" role="status">Carregando conversa…</p>
  if (error || !data) return <main className="mx-auto max-w-2xl px-4 py-10"><VoltarLink to="/conversas-equipe">Conversas da equipe</VoltarLink><p role="alert">{error instanceof ApiError ? error.message : 'Não foi possível carregar a conversa.'}</p></main>
  const mensagens = [...new Map([...anteriores, ...data.mensagens].map(m => [m.id, m])).values()].sort((a, b) => a.id - b.id)
  const fixadas = new Set(data.fixadas.map(m => m.id))
  return <main className="mx-auto flex max-w-3xl flex-col gap-5 px-4 py-8">
    <VoltarLink to="/conversas-equipe">Conversas da equipe</VoltarLink>
    <header><h1 className="text-2xl font-semibold">{data.titulo}</h1><p className="mt-1 text-sm text-muted-foreground">Chat privado da equipe. Atualiza automaticamente enquanto estiver aberto.</p></header>
    <details className="rounded-xl border border-border p-3"><summary className="cursor-pointer text-sm font-medium">Quem participa ({data.participantes.length})</summary><ul className="mt-3 flex flex-col gap-2 text-sm">{data.participantes.map(p => <li key={p.usuario_id}>{p.nome} <span className="text-muted-foreground">· {papeisChat[p.papel] ?? p.papel}</span></li>)}</ul></details>
    {data.fixadas.length > 0 && <section aria-label="Avisos fixados" className="rounded-xl border border-primary/30 bg-primary/5 p-4"><h2 className="mb-2 font-semibold">Avisos fixados</h2><ul className="flex flex-col gap-3">{data.fixadas.map(m => <li key={m.id} className="min-w-0"><p className="whitespace-pre-wrap break-words text-sm">{m.texto || 'Foto fixada na conversa'}</p>{m.tem_foto && <FotoPrivada key={`${m.id}-${usuario?.id}`} eventoId={id} mensagemId={m.id} />}<p className="text-xs text-muted-foreground">{m.nome}</p>{data.pode_fixar && <Button type="button" size="sm" variant="ghost" disabled={ocupado} onClick={() => executar(() => api(`${baseChat(id)}/mensagens/${m.id}/fixada`, { method: 'PUT', body: { fixada: false } }))}>Desafixar</Button>}</li>)}</ul></section>}
    {data.atencoes.length > 0 && <section aria-label="Pedidos de atenção" className="rounded-xl border border-amber-500/40 bg-amber-500/10 p-4"><h2 className="font-semibold">Jurados precisam de atenção</h2><ul className="mt-2 flex flex-col gap-3">{data.atencoes.map(a => <li key={a.id} className="flex flex-wrap items-center justify-between gap-2 text-sm"><span>{a.nome} · {new Date(a.criado_em).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}</span><Button type="button" size="sm" disabled={ocupado} onClick={() => executar(() => api(`${baseChat(id)}/atencoes/${a.id}/atender`, { method: 'POST' }))}>Confirmar atendimento</Button></li>)}</ul></section>}
    <section aria-label="Mensagens" className="rounded-xl border border-border">
      {!fimHistorico && data.pode_carregar && <Button type="button" variant="ghost" className="m-2" disabled={carregandoAntigas} onClick={antigas}>{carregandoAntigas ? 'Carregando…' : 'Carregar anteriores'}</Button>}
      <div ref={lista} onScroll={e => setPertoFim(e.currentTarget.scrollHeight - e.currentTarget.scrollTop - e.currentTarget.clientHeight < 60)} className="flex max-h-96 flex-col gap-4 overflow-y-auto p-4" tabIndex={0}>
        {mensagens.length === 0 && <p className="py-6 text-center text-sm text-muted-foreground">Comece a conversa com a equipe.</p>}
        {mensagens.map(m => { const leitores = data.participantes.filter(p => p.usuario_id !== m.usuario_id && p.lido_ate >= m.id); return <article key={m.id} className={`min-w-0 rounded-xl p-3 ${m.usuario_id === usuario?.id ? 'ml-4 bg-primary/10' : 'mr-4 bg-muted'}`}>
          <div className="mb-1 flex flex-wrap justify-between gap-2 text-xs text-muted-foreground"><span className="font-medium text-foreground">{m.nome}</span><time dateTime={m.criado_em}>{new Date(m.criado_em).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}</time></div>
          <p className="whitespace-pre-wrap break-words text-sm">{m.texto}</p>{m.tem_foto && <FotoPrivada key={`${m.id}-${usuario?.id}`} eventoId={id} mensagemId={m.id} />}
          <div className="mt-2 flex flex-wrap items-center justify-between gap-2"><details className="text-xs text-muted-foreground"><summary className="cursor-pointer">Lido por {leitores.length}</summary><p className="mt-1">{leitores.map(p => p.nome).join(', ') || 'Ainda sem confirmação de leitura.'}</p></details>{data.pode_fixar && <Button type="button" size="sm" variant="ghost" disabled={ocupado} onClick={() => executar(() => api(`${baseChat(id)}/mensagens/${m.id}/fixada`, { method: 'PUT', body: { fixada: !fixadas.has(m.id) } }))}>{fixadas.has(m.id) ? 'Desafixar' : 'Fixar aviso'}</Button>}</div>
        </article> })}
      </div>
      {!pertoFim && <Button type="button" variant="ghost" className="m-2" onClick={() => { setPertoFim(true); if (lista.current) lista.current.scrollTop = lista.current.scrollHeight }}>Ir para as últimas mensagens</Button>}
    </section>
    <form onSubmit={enviar} className="flex flex-col gap-3">
      <Label htmlFor="chat-texto">Mensagem para a equipe</Label><Textarea id="chat-texto" rows={3} maxLength={3000} value={texto} disabled={ocupado} onChange={e => { setTexto(e.target.value); tokenEnvio.current = crypto.randomUUID() }} placeholder="Combine os detalhes do evento…" />
      <div className="flex flex-wrap items-center gap-3"><label className="cursor-pointer rounded-lg border border-border px-3 py-2 text-sm">{foto ? 'Trocar foto' : 'Anexar foto'}<input ref={arquivo} type="file" accept="image/jpeg,image/png" className="sr-only" disabled={ocupado} onChange={e => { const f = e.target.files?.[0] ?? null; if (f && f.size > 5 * 1024 * 1024) { setErro('A foto deve ter até 5 MB.'); e.target.value = ''; return } setFoto(f); tokenEnvio.current = crypto.randomUUID() }} /></label>{foto && <><span className="min-w-0 break-all text-xs">{foto.name}</span><Button type="button" size="sm" variant="ghost" disabled={ocupado} onClick={() => { setFoto(null); tokenEnvio.current = crypto.randomUUID(); if (arquivo.current) arquivo.current.value = '' }}>Remover foto</Button></>}<Button type="submit" disabled={ocupado || (!texto.trim() && !foto)}>{ocupado ? 'Enviando…' : 'Enviar mensagem'}</Button></div>
      <p className="text-xs text-muted-foreground">Foto JPEG ou PNG, até 5 MB. Visível somente para quem tem acesso ao chat.</p>
    </form>
    {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
    <label className="flex items-start gap-2 text-sm text-muted-foreground"><input type="checkbox" checked={data.email_ativo} disabled={ocupado} onChange={e => executar(() => api(`${baseChat(id)}/lido`, { method: 'POST', body: { ate: data.lido_ate, email_ativo: e.target.checked } }))} />Receber resumo por e-mail quando houver mensagens não lidas (no máximo uma vez por hora).</label>
  </main>
}
