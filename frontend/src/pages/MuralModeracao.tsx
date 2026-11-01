import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import { baseMural, rotulosMural, type PostMural } from '@/lib/mural'
import { ConteudoMural } from '@/components/mural-post'
import { MuralStories } from '@/components/mural-stories'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { VoltarLink } from '@/components/voltar-link'

function Revisao({ slug, post, atualizar }: { slug: string; post: PostMural; atualizar: () => Promise<unknown> }) {
 const { usuario } = useAuth()
 const [motivo,setMotivo] = useState(''), [ciente,setCiente] = useState(false), [erro,setErro] = useState(''), [ocupado,setOcupado] = useState(false), [verDenuncias,setVerDenuncias] = useState(false)
 const denuncias = useQuery({ queryKey: ['mural-denuncias',slug,post.id,usuario?.id], queryFn: () => api<{ id: number; motivo: string }[]>(`${baseMural(slug)}/${post.id}/denuncias`), enabled: verDenuncias, retry: false })
 const decidir = async (acao: string) => {
   if (acao === 'remover' && !confirm('Remover esta publicação e apagar suas fotos?')) return
   setOcupado(true);setErro('')
   try { await api(`${baseMural(slug)}/${post.id}/moderar`, { method: 'POST', body: { acao, motivo, ciente_alertas: ciente } }); await atualizar() }
   catch(e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível registrar a decisão.') }
   finally { setOcupado(false) }
 }
 const alerta = post.ia_alerta || post.ia_status === 'indisponivel', analisando = post.ia_status === 'pendente' || post.ia_status === 'analisando'
 return <article className="flex min-w-0 flex-col gap-3 rounded-xl border border-border p-4">
   <p className="text-xs font-medium">{rotulosMural[post.status]}{post.destaque && ' · Melhor momento'}</p><ConteudoMural slug={slug} post={post} />
   {post.status !== 'removido' && <>
     <div className={`rounded-lg p-3 text-sm ${alerta ? 'border border-amber-500/40 bg-amber-500/10' : 'bg-muted'}`}>
       <p className="font-medium">{analisando ? 'Análise automática em andamento' : alerta ? 'Revise os alertas antes de decidir' : 'Análise automática sem alertas'}</p>
       <p className="mt-1 text-xs text-muted-foreground">A IA pode errar. A decisão é sua: confira o comentário e todas as fotos, inclusive direitos de imagem e exposição de menores.</p>
       {!!post.ia_motivos?.length && <ul className="mt-2 list-disc pl-5">{post.ia_motivos.map((m,i) => <li key={i}>{m}</li>)}</ul>}
       {alerta && <label className="mt-3 flex items-start gap-2"><input type="checkbox" checked={ciente} onChange={e => setCiente(e.target.checked)} />Li os alertas e revisei manualmente o texto e todas as fotos.</label>}
     </div>
     {!!post.denuncias && <div className="flex flex-col gap-2"><Button type="button" variant="outline" size="sm" className="self-start" onClick={() => setVerDenuncias(!verDenuncias)}>Ver denúncias ({post.denuncias})</Button>{verDenuncias && <ul className="list-disc pl-5 text-sm">{denuncias.data?.map(d => <li key={d.id}>{d.motivo}</li>)}</ul>}{denuncias.error && <p role="alert" className="text-sm text-destructive">Não foi possível carregar as denúncias.</p>}</div>}
     <Label htmlFor={`motivo-${post.id}`}>Orientação para o autor (obrigatória para rejeitar)</Label><Textarea id={`motivo-${post.id}`} maxLength={500} rows={2} value={motivo} onChange={e => setMotivo(e.target.value)} />
     <div className="evve-actions flex flex-wrap gap-2">
       {post.status !== 'aprovado' && <Button type="button" disabled={ocupado || analisando || (!!alerta && !ciente)} onClick={() => decidir('aprovar')}>Aprovar publicação</Button>}
       {post.status !== 'rejeitado' && <Button type="button" variant="outline" disabled={ocupado || !motivo.trim()} onClick={() => decidir('rejeitar')}>Rejeitar</Button>}
       {post.status === 'aprovado' && <><Button type="button" variant="outline" disabled={ocupado} onClick={() => decidir(post.destaque ? 'desdestacar' : 'destacar')}>{post.destaque ? 'Retirar dos melhores momentos' : 'Destacar melhor momento'}</Button><Button type="button" variant="outline" disabled={ocupado} onClick={() => decidir('ocultar')}>Ocultar para revisar</Button></>}
       <Button type="button" variant="destructive" disabled={ocupado} onClick={() => decidir('remover')}>Remover definitivamente</Button>
     </div>
   </>}
   {post.motivo && <p className="text-sm text-muted-foreground">Última orientação: {post.motivo}</p>}{erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
 </article>
}
export default function MuralModeracao() { const {slug=''}=useParams();const {usuario}=useAuth();return <Painel key={`${slug}-${usuario?.id}`} slug={slug} /> }
function Painel({slug}:{slug:string}) {
 const {usuario}=useAuth(),qc=useQueryClient()
 const [status,setStatus]=useState('pendente'),[antes,setAntes]=useState(0)
 const {data,isLoading,error}=useQuery({queryKey:['mural-moderacao',slug,status,antes,usuario?.id],queryFn:()=>api<PostMural[]>(`${baseMural(slug)}/moderacao?status=${status}&antes=${antes}`),refetchInterval:15000,retry:false})
 const atualizar=()=>Promise.all([qc.invalidateQueries({queryKey:['mural-moderacao',slug]}),qc.invalidateQueries({queryKey:['mural-publico',slug]}),qc.invalidateQueries({queryKey:['mural-eu',slug]})])
 return <main className="mx-auto flex max-w-3xl flex-col gap-5 px-4 py-10"><VoltarLink to={`/e/${slug}/mural`}>Voltar ao mural</VoltarLink><header><h1 className="text-2xl font-semibold">Revisar mural do evento</h1><p className="mt-2 text-sm text-muted-foreground">Os posts só ficam públicos após sua aprovação. A IA auxilia a revisão e nunca publica por conta própria.</p></header>
   <label className="flex flex-col gap-2 text-sm">Situação<select className="h-10 rounded-lg border border-border bg-background px-3" value={status} onChange={e=>{setStatus(e.target.value);setAntes(0)}}>{Object.entries(rotulosMural).map(([k,v])=><option key={k} value={k}>{v}</option>)}</select></label>
   {isLoading&&<p role="status">Carregando publicações…</p>}{error&&<p role="alert" className="text-destructive">{error instanceof ApiError?error.message:'Não foi possível carregar as publicações.'}</p>}
   {!error&&data?.length===0&&<p className="text-sm text-muted-foreground">Nenhuma publicação nesta situação.</p>}
   {!error&&data?.map(p=><Revisao key={`${p.id}-${p.status}`} slug={slug} post={p} atualizar={atualizar} />)}
   <div className="evve-actions flex flex-wrap gap-2">{antes>0&&<Button variant="outline" onClick={()=>setAntes(0)}>Voltar às mais recentes</Button>}{!error&&data?.length===20&&<Button variant="outline" onClick={()=>setAntes(data[data.length-1].id)}>Carregar anteriores</Button>}</div>
   {!error&&!isLoading&&<MuralStories slug={slug} />}
 </main>
}
