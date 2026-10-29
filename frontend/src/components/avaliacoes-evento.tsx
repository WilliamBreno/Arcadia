import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { formatarMedia, MAX_COMENTARIO, type AvaliacoesEvento, type MinhaAvaliacao } from '@/lib/avaliacoes'
import { useAuth } from '@/hooks/use-auth'
import { EscolherNota, Estrelas } from '@/components/estrelas'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

const plural = (n: number) => `${n} ${n === 1 ? 'avaliação' : 'avaliações'}`

// Seção "Avaliações" da página do evento (item 6.5). Aparece depois que o
// evento começa. A média só aparece com 3 avaliações ou mais.
export function AvaliacoesDoEvento({ slug, eventoComecou }: { slug: string; eventoComecou: boolean }) {
  const { usuario } = useAuth()
  const location = useLocation()
  const queryClient = useQueryClient()
  const veioDoEmail = location.hash === '#avaliar'

  const publicas = useQuery({
    queryKey: ['avaliacoes', slug],
    queryFn: () => api<AvaliacoesEvento>(`/eventos/${slug}/avaliacoes`),
    enabled: eventoComecou,
  })
  const minha = useQuery({
    queryKey: ['minha-avaliacao', slug, usuario?.id],
    queryFn: () => api<MinhaAvaliacao>(`/eventos/${slug}/minha-avaliacao`),
    enabled: eventoComecou && !!usuario,
  })

  const [nota, setNota] = useState(0)
  const [comentario, setComentario] = useState('')
  const [editando, setEditando] = useState(false)
  const [salvando, setSalvando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const [salvo, setSalvo] = useState(false)

  useEffect(() => {
    const a = minha.data?.avaliacao
    if (a) {
      setNota(a.nota)
      setComentario(a.comentario)
    }
  }, [minha.data])

  // O link do e-mail "Como foi?" (#avaliar) rola até o formulário.
  useEffect(() => {
    if (veioDoEmail && (minha.data || !usuario)) {
      document.getElementById('avaliar')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }, [veioDoEmail, minha.data, usuario])

  if (!eventoComecou) return null
  const resumo = publicas.data?.resumo
  const lista = publicas.data?.avaliacoes ?? []
  const podeAvaliar = !!minha.data?.pode_avaliar
  const minhaAv = minha.data?.avaliacao ?? null
  // Sem nada para mostrar nem para fazer, a seção não aparece.
  if (!lista.length && !podeAvaliar && !minhaAv && !(veioDoEmail && !usuario)) return null

  const enviar = async (e: React.FormEvent) => {
    e.preventDefault()
    if (nota < 1) {
      setErro('Escolha de 1 a 5 estrelas.')
      return
    }
    setErro(null)
    setSalvando(true)
    try {
      await api(`/eventos/${slug}/minha-avaliacao`, { method: 'PUT', body: { nota, comentario } })
      setSalvo(true)
      setEditando(false)
      queryClient.invalidateQueries({ queryKey: ['avaliacoes', slug] })
      queryClient.invalidateQueries({ queryKey: ['minha-avaliacao', slug] })
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Não foi possível salvar. Tente de novo.')
    } finally {
      setSalvando(false)
    }
  }

  const mostrarFormulario = podeAvaliar && (!minhaAv || editando)

  return (
    <section id="avaliar" aria-labelledby="titulo-avaliacoes" className="mt-8 scroll-mt-20">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="titulo-avaliacoes" className="text-xl font-semibold text-foreground">
          Avaliações
        </h2>
        {resumo && resumo.total > 0 && (
          <p className="flex items-center gap-2 text-sm text-muted-foreground">
            {resumo.mostrar_media ? (
              <>
                <span className="text-2xl font-semibold tabular-nums text-foreground">{formatarMedia(resumo.media)}</span>
                <Estrelas nota={resumo.media} />
                <span>{plural(resumo.total)}</span>
              </>
            ) : (
              <span>{plural(resumo.total)} · a nota média aparece a partir de 3</span>
            )}
          </p>
        )}
      </div>

      {!usuario && veioDoEmail && (
        <p className="mt-3 rounded-xl border border-border p-4 text-sm text-foreground">
          <Link to="/login" state={{ de: `${location.pathname}#avaliar` }} className="font-medium underline underline-offset-4">
            Entre na sua conta
          </Link>{' '}
          para avaliar este evento.
        </p>
      )}

      {usuario && veioDoEmail && minha.data && !podeAvaliar && !minhaAv && (
        <p className="mt-3 rounded-xl bg-muted p-4 text-sm text-foreground">{minha.data.motivo}.</p>
      )}

      {mostrarFormulario && (
        <form onSubmit={enviar} className="mt-4 flex flex-col gap-3 rounded-xl border border-border p-4">
          <p className="text-sm text-foreground">{minhaAv ? 'Editar sua avaliação' : 'Você foi a este evento. Como foi?'}</p>
          <EscolherNota valor={nota} aoMudar={setNota} />
          <div>
            <label htmlFor="comentario-avaliacao" className="text-sm font-medium text-foreground">
              Comentário <span className="font-normal text-muted-foreground">(opcional)</span>
            </label>
            <Textarea
              id="comentario-avaliacao"
              className="mt-1"
              rows={3}
              maxLength={MAX_COMENTARIO}
              value={comentario}
              onChange={(e) => setComentario(e.target.value)}
              placeholder="O que foi bom? O que pode melhorar?"
            />
            <p className="mt-1 text-right text-xs tabular-nums text-muted-foreground">
              {comentario.length}/{MAX_COMENTARIO}
            </p>
          </div>
          {erro && (
            <p role="alert" className="text-sm text-destructive">
              {erro}
            </p>
          )}
          <div className="flex gap-2">
            <Button type="submit" disabled={salvando}>
              {salvando ? 'Salvando…' : minhaAv ? 'Salvar alteração' : 'Enviar avaliação'}
            </Button>
            {editando && (
              <Button type="button" variant="ghost" onClick={() => setEditando(false)}>
                Cancelar
              </Button>
            )}
          </div>
          <p className="text-xs text-muted-foreground">Aparece com seu primeiro nome e a inicial do sobrenome.</p>
        </form>
      )}

      {minhaAv && !editando && (
        <div className="mt-4 flex flex-wrap items-center justify-between gap-2 rounded-xl bg-muted p-4 text-sm">
          <span className="flex items-center gap-2 text-foreground">
            {salvo ? 'Obrigado! Sua avaliação:' : 'Sua avaliação:'} <Estrelas nota={minhaAv.nota} />
          </span>
          {minhaAv.ocultada && <span className="text-muted-foreground">Ocultada pela moderação por não seguir as regras.</span>}
          {podeAvaliar && (
            <Button type="button" variant="outline" size="sm" onClick={() => setEditando(true)}>
              Editar
            </Button>
          )}
        </div>
      )}

      {lista.length > 0 && (
        <ul className="mt-4 flex flex-col divide-y divide-border">
          {lista.map((a) => (
            <li key={a.id} className="py-3">
              <div className="flex flex-wrap items-center gap-2 text-sm">
                <Estrelas nota={a.nota} tamanho={14} />
                <span className="font-medium text-foreground">{a.nome}</span>
                <span className="text-muted-foreground">
                  · {new Date(a.criado_em).toLocaleDateString('pt-BR', { day: 'numeric', month: 'short' })}
                </span>
              </div>
              {a.comentario && <p className="mt-1 whitespace-pre-line text-pretty text-foreground">{a.comentario}</p>}
            </li>
          ))}
        </ul>
      )}
      <p className="mt-2 text-xs text-muted-foreground">Só avalia quem foi ao evento e fez check-in.</p>
    </section>
  )
}
