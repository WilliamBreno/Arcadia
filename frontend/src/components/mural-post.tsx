import { useEffect, useState } from 'react'
import { useAuth } from '@/hooks/use-auth'
import { fotoMural, type PostMural } from '@/lib/mural'

export function FotoDoMural({ slug, post, foto }: { slug: string; post: number; foto: number }) {
 const { usuario } = useAuth()
 const [src, setSrc] = useState(''), [erro, setErro] = useState(false)
 useEffect(() => {
   const abort = new AbortController(); let url = ''
   fotoMural(slug, post, foto, abort.signal).then(blob => { if (!abort.signal.aborted) { url = URL.createObjectURL(blob); setSrc(url) } }).catch(() => { if (!abort.signal.aborted) setErro(true) })
   return () => { abort.abort(); if (url) URL.revokeObjectURL(url) }
 }, [slug, post, foto, usuario?.id])
 if (erro) return <p className="text-xs text-muted-foreground">Foto indisponível ou em revisão.</p>
 return src ? <a href={src} target="_blank" rel="noreferrer"><img src={src} alt="Foto da publicação no mural" className="max-h-80 w-full rounded-lg object-contain" /></a> : <p className="text-xs text-muted-foreground">Carregando foto…</p>
}
export function ConteudoMural({ slug, post }: { slug: string; post: PostMural }) {
 const { usuario } = useAuth()
 return <>
   <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground"><span className="font-medium text-foreground">{post.autor}</span><time dateTime={post.criado_em}>{new Date(post.criado_em).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })}</time></div>
   {post.texto && <p className="whitespace-pre-wrap break-words text-sm">{post.texto}</p>}
   {post.fotos.length > 0 && <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">{post.fotos.map(f => <FotoDoMural key={`${f}-${usuario?.id}`} slug={slug} post={post.id} foto={f} />)}</div>}
 </>
}
