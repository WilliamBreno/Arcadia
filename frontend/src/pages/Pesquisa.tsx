import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api, ApiError } from '@/lib/api'

type Pesquisa = { evento_titulo: string; evento_slug: string; ja_respondeu: boolean; nota: number | null; comentario: string; aberta: boolean }

// Item 9.7: pesquisa pós-evento, aberta pelo link do e-mail (sem login).
// A nota clicada no e-mail já vem marcada (?nota=8).
export default function Pesquisa() {
  const { token = '' } = useParams()
  const [params] = useSearchParams()
  const { data, isLoading } = useQuery({
    queryKey: ['pesquisa', token],
    retry: false,
    queryFn: () => api<Pesquisa>(`/pesquisa/${encodeURIComponent(token)}`, { semAuth: true }),
  })
  const daUrl = Number(params.get('nota'))
  const [nota, setNota] = useState<number | null>(params.get('nota') !== null && daUrl >= 0 && daUrl <= 10 ? daUrl : null)
  const [comentario, setComentario] = useState<string | null>(null)
  const [enviada, setEnviada] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  if (isLoading) return <Carregando />
  if (!data) {
    return (
      <main className="mx-auto max-w-md px-4 py-16 text-center">
        <h1 className="text-xl font-semibold text-foreground">Link inválido</h1>
        <p className="mt-2 text-muted-foreground">Use o link que chegou no seu e-mail.</p>
      </main>
    )
  }
  const notaAtual = nota ?? data.nota
  const textoAtual = comentario ?? data.comentario

  const enviar = async () => {
    setErro(null)
    try {
      await api(`/pesquisa/${encodeURIComponent(token)}`, { method: 'POST', body: { nota: notaAtual, comentario: textoAtual }, semAuth: true })
      setEnviada(true)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível enviar.')
    }
  }

  if (enviada) {
    return (
      <main className="mx-auto max-w-md px-4 py-16 text-center">
        <h1 className="text-2xl font-semibold text-foreground">Obrigado!</h1>
        <p className="mt-2 text-muted-foreground">Sua resposta ajuda o organizador a fazer um evento ainda melhor.</p>
        <Link to={`/e/${data.evento_slug}`} className="mt-6 inline-block text-sm underline underline-offset-4">
          Ver a página do evento
        </Link>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <p className="text-sm font-medium text-primary">{data.evento_titulo}</p>
      <h1 className="mt-1 text-2xl font-semibold text-foreground">De 0 a 10, quanto você recomendaria este evento a um amigo?</h1>
      {!data.aberta && <p className="mt-4 text-muted-foreground">A pesquisa deste evento já fechou. Obrigado!</p>}
      {data.aberta && (
        <>
          <fieldset className="mt-6">
            <legend className="sr-only">Nota de 0 a 10</legend>
            <div className="grid grid-cols-6 gap-2 sm:grid-cols-11">
              {Array.from({ length: 11 }, (_, n) => (
                <button
                  key={n}
                  type="button"
                  aria-pressed={notaAtual === n}
                  onClick={() => setNota(n)}
                  className={`h-11 rounded-lg border text-base font-medium tabular-nums transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring ${
                    notaAtual === n ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-foreground hover:border-primary'
                  }`}
                >
                  {n}
                </button>
              ))}
            </div>
            <div className="mt-2 flex justify-between text-xs text-muted-foreground">
              <span>Nada provável</span>
              <span>Com certeza</span>
            </div>
          </fieldset>
          <div className="mt-6 flex flex-col gap-1.5">
            <Label htmlFor="comentario-pesquisa">O que mais marcou, ou o que podia melhorar? (opcional)</Label>
            <Textarea id="comentario-pesquisa" rows={4} maxLength={1000} value={textoAtual} onChange={(e) => setComentario(e.target.value)} />
          </div>
          <p className="mt-3 text-xs text-muted-foreground">
            Resposta anônima: o organizador vê só as notas e os comentários, sem o seu nome nem o seu e-mail.
            {data.ja_respondeu ? ' Você já respondeu; enviar de novo troca a resposta.' : ''}
          </p>
          {erro && (
            <p role="alert" className="mt-2 text-sm text-destructive">
              {erro}
            </p>
          )}
          <Button className="mt-4 w-full sm:w-auto" disabled={notaAtual === null} onClick={enviar}>
            Enviar resposta
          </Button>
        </>
      )}
    </main>
  )
}
