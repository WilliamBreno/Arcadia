import { useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useLocation, useParams } from 'react-router-dom'
import { Heart, Flag } from 'lucide-react'
import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import { baseMural, enviarMural, rotulosMural, type EuMural, type PaginaMural, type PostMural } from '@/lib/mural'
import { ConteudoMural } from '@/components/mural-post'
import { Button, buttonVariants } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { VoltarLink } from '@/components/voltar-link'

function Publicacao({ slug, post, curtida, aoMudar }: { slug: string; post: PostMural; curtida: boolean; aoMudar: () => Promise<unknown> }) {
 const { usuario } = useAuth(), location = useLocation()
 const [denuncia, setDenuncia] = useState(false), [motivo, setMotivo] = useState(''), [erro, setErro] = useState(''), [ok, setOk] = useState(''), [ocupado, setOcupado] = useState(false)
 const agir = async (f: () => Promise<unknown>, aviso = '') => { setErro(''); setOk(''); setOcupado(true); try { await f(); await aoMudar(); setOk(aviso) } catch(e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir.') } finally { setOcupado(false) } }
 return <article className="flex min-w-0 flex-col gap-3 rounded-xl border border-border p-4">
   {post.destaque && <span className="text-xs font-medium text-primary">Melhor momento</span>}
   <ConteudoMural slug={slug} post={post} />
   {usuario ? <div className="flex flex-wrap items-center gap-2"><Button size="sm" variant={curtida ? 'secondary' : 'ghost'} disabled={ocupado} aria-pressed={curtida} onClick={() => agir(() => api(`${baseMural(slug)}/${post.id}/curtida`, { method: 'PUT', body: { curtida: !curtida } }))}><Heart className="size-4" />{curtida ? 'Curtido' : 'Curtir'} · {post.curtidas}</Button><Button size="sm" variant="ghost" onClick={() => setDenuncia(!denuncia)}><Flag className="size-4" />Denunciar</Button></div> : <p className="text-xs text-muted-foreground">{post.curtidas} curtida(s) · <Link to="/login" state={{ de: location.pathname }} className="underline">Entre para curtir ou denunciar</Link></p>}
   {denuncia && <div className="flex flex-col gap-2 rounded-lg bg-muted p-3"><Label htmlFor={`denuncia-${post.id}`}>O que a organização precisa revisar?</Label><Textarea id={`denuncia-${post.id}`} maxLength={500} value={motivo} onChange={e => setMotivo(e.target.value)} placeholder="Conteúdo sensível, dados pessoais, direito de imagem…" /><Button size="sm" disabled={ocupado || motivo.trim().length < 5} onClick={() => agir(async () => { await api(`${baseMural(slug)}/${post.id}/denunciar`, { method: 'POST', body: { motivo } }); setDenuncia(false) }, 'Denúncia enviada. A organização vai avaliar esta publicação.')}>Enviar denúncia</Button></div>}
   {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}{ok && <p role="status" className="text-sm text-muted-foreground">{ok}</p>}
 </article>
}

export default function MuralEvento() { const { slug = '' } = useParams(); const { usuario } = useAuth(); return <><meta name="robots" content="noindex, nofollow" /><Mural key={`${slug}-${usuario?.id}`} slug={slug} /></> }
function Mural({ slug }: { slug: string }) {
 const { usuario } = useAuth(), location = useLocation(), qc = useQueryClient()
 const [antes, setAntes] = useState(0)
 const publico = useQuery({ queryKey: ['mural-publico', slug, antes], queryFn: () => api<PaginaMural>(`${baseMural(slug)}${antes ? `?antes=${antes}` : ''}`, { semAuth: true }), refetchInterval: 30000, retry: false })
 const eu = useQuery({ queryKey: ['mural-eu', slug, usuario?.id], queryFn: () => api<EuMural>(`${baseMural(slug)}/eu`), enabled: !!usuario, refetchInterval: 15000, retry: false })
 const [texto, setTexto] = useState(''), [fotos, setFotos] = useState<File[]>([]), [consentiu, setConsentiu] = useState(false), [participando, setParticipando] = useState(false)
 const [erro, setErro] = useState(''), [aviso, setAviso] = useState(''), [ocupado, setOcupado] = useState(false)
 const token = useRef(crypto.randomUUID()), arquivo = useRef<HTMLInputElement>(null)
 const mudar = () => Promise.all([qc.invalidateQueries({ queryKey: ['mural-publico', slug] }), qc.invalidateQueries({ queryKey: ['mural-eu', slug, usuario?.id] })])
 const enviar = async (e: React.FormEvent) => {
   e.preventDefault(); setErro(''); setAviso(''); setOcupado(true)
   try { await enviarMural(slug, texto, token.current, fotos, participando); setTexto(''); setFotos([]); setConsentiu(false); if (arquivo.current) arquivo.current.value = ''; token.current = crypto.randomUUID(); setAviso('Publicação enviada para aprovação. Ela só aparece no mural depois que a organização aprovar.'); await mudar() }
   catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível enviar. Tente novamente.') }
   finally { setOcupado(false) }
 }
 const remover = async (id: number) => {
   if (!confirm('Remover esta publicação e suas fotos?')) return
   setOcupado(true); setErro('')
   try { await api(`${baseMural(slug)}/${id}`, { method: 'DELETE' }); await mudar() } catch(e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível remover.') } finally { setOcupado(false) }
 }
 if (publico.isLoading) return <p className="p-8" role="status">Carregando mural…</p>
 if (publico.error || !publico.data) return <main className="mx-auto max-w-2xl px-4 py-10"><VoltarLink to={`/e/${slug}`}>Voltar ao evento</VoltarLink><p role="alert">{publico.error instanceof ApiError ? publico.error.message : 'Não foi possível carregar o mural.'}</p></main>
 const data = publico.data, livre = !!eu.data?.entrada_livre && eu.data.motivo.startsWith('confirme que está participando')
 const pode = !eu.error && (eu.data?.pode_postar || livre)
 return <main className="mx-auto flex max-w-3xl flex-col gap-6 px-4 py-10">
   <VoltarLink to={`/e/${slug}`}>Voltar ao evento</VoltarLink>
   <header><p className="text-sm text-muted-foreground">{data.titulo}</p><h1 className="mt-1 text-3xl font-semibold">Mural do evento</h1><p className="mt-2 text-sm text-muted-foreground">Momentos de quem viveu o evento. Toda publicação passa pela aprovação da organização.</p></header>
   {!eu.error && eu.data?.moderador && <Link to={`/e/${slug}/mural/moderacao`} className={buttonVariants({ variant: 'outline', className: 'self-start' })}>Revisar publicações e gerar Stories</Link>}
   {!usuario && <p className="rounded-xl bg-muted p-4 text-sm"><Link to="/login" state={{ de: location.pathname }} className="font-medium underline">Entre na sua conta</Link> para enviar um momento do evento.</p>}
   {usuario && eu.isLoading && <p role="status" className="text-sm">Conferindo participação…</p>}
   {eu.error && <p role="alert" className="text-sm text-destructive">Não foi possível conferir sua participação. Atualize a página para tentar novamente.</p>}
   {pode ? <form onSubmit={enviar} className="flex flex-col gap-3 rounded-xl border border-border p-4">
     <h2 className="font-semibold">Compartilhe um momento</h2><Label htmlFor="mural-texto">Seu comentário</Label><Textarea id="mural-texto" maxLength={2000} rows={3} value={texto} disabled={ocupado} onChange={e => { setTexto(e.target.value); token.current = crypto.randomUUID() }} placeholder="Como está sendo o evento?" />
     <label className="self-start rounded-lg border border-border px-3 py-2 text-sm">Adicionar até cinco fotos<input ref={arquivo} type="file" accept="image/jpeg,image/png" multiple className="mt-2 block w-full max-w-xs text-sm" disabled={ocupado} onChange={e => { const fs = [...(e.target.files ?? [])]; if (fs.length > 5 || fs.some(f => f.size > 5 * 1024 * 1024)) { setErro('Selecione até cinco fotos JPEG/PNG, de até 5 MB cada.'); e.target.value = ''; return } setFotos(fs); token.current = crypto.randomUUID(); setErro('') }} /></label>
     {fotos.length > 0 && <ul className="flex flex-col gap-2 text-xs">{fotos.map((f,i) => <li key={i} className="flex items-center justify-between gap-2"><span className="min-w-0 break-all">{f.name}</span><Button type="button" size="sm" variant="ghost" disabled={ocupado} onClick={() => { setFotos(fotos.filter((_,j) => i !== j)); token.current = crypto.randomUUID() }}>Remover foto</Button></li>)}</ul>}
     {livre && <label className="flex items-start gap-2 text-sm"><input type="checkbox" checked={participando} onChange={e => setParticipando(e.target.checked)} />Confirmo que estou participando deste evento de entrada livre.</label>}
     <label className="flex items-start gap-2 text-sm text-muted-foreground"><input type="checkbox" checked={consentiu} onChange={e => setConsentiu(e.target.checked)} />Tenho autorização para compartilhar este conteúdo e as imagens das pessoas retratadas. Entendo que texto e fotos podem ser analisados por IA e ficarão públicos após aprovação.</label>
     <p className="text-xs text-muted-foreground">Não inclua telefone, endereço ou dados pessoais. JPEG/PNG, até 5 MB e 20 megapixels por foto. Novos posts até sete dias após o término.</p>
     <Button type="submit" disabled={ocupado || !consentiu || (livre && !participando) || (!texto.trim() && !fotos.length)} className="self-start">{ocupado ? 'Enviando para análise…' : 'Enviar para aprovação'}</Button>
   </form> : !eu.error && eu.data && <p className="rounded-xl bg-muted p-4 text-sm">{eu.data.motivo}</p>}
   {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}{aviso && <p role="status" className="text-sm">{aviso}</p>}
   {!eu.error && !!eu.data?.meus.length && <details className="rounded-xl border border-border p-4"><summary className="cursor-pointer font-medium">Minhas publicações ({eu.data.meus.length})</summary><div className="mt-4 flex flex-col gap-3">{eu.data.meus.map(p => <article key={p.id} className="flex flex-col gap-3 rounded-lg bg-muted p-3"><p className="text-xs font-medium">{rotulosMural[p.status]}</p><ConteudoMural slug={slug} post={p} />{p.motivo && <p className="text-sm">Orientação da organização: {p.motivo}</p>}{p.status !== 'removido' && <Button type="button" size="sm" variant="ghost" className="self-start" disabled={ocupado} onClick={() => remover(p.id)}>Remover minha publicação</Button>}</article>)}</div></details>}
   {data.destaques.length > 0 && <section className="rounded-xl bg-primary/5 p-4"><h2 className="mb-3 text-lg font-semibold">Melhores momentos</h2><div className="flex flex-col gap-3">{data.destaques.slice(0,4).map(p => <Publicacao key={p.id} slug={slug} post={p} curtida={eu.data?.curtidas.includes(p.id) ?? false} aoMudar={mudar} />)}</div></section>}
   <section aria-label="Publicações do mural" className="flex flex-col gap-4"><h2 className="text-lg font-semibold">Últimas publicações</h2>{data.posts.length === 0 && <p className="text-sm text-muted-foreground">Ainda não há publicações aprovadas. Os momentos compartilhados aparecem aqui após a revisão da organização.</p>}{data.posts.map(p => <Publicacao key={p.id} slug={slug} post={p} curtida={eu.data?.curtidas.includes(p.id) ?? false} aoMudar={mudar} />)}</section>
   <div className="flex flex-wrap gap-2">{antes > 0 && <Button variant="outline" onClick={() => setAntes(0)}>Voltar às mais recentes</Button>}{data.posts.length === 20 && <Button variant="outline" onClick={() => setAntes(data.posts[data.posts.length-1].id)}>Ver publicações anteriores</Button>}</div>
 </main>
}
