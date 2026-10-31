import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import { listarMeusArtistas, nomeTipoArtista, tiposArtista, type DadosArtista, type TipoArtista } from '@/lib/artistas'

// Item 11.1: "Meus artistas" — grupos que a pessoa lidera ou de que faz parte.
export default function MeusArtistas() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['meus-artistas'], queryFn: listarMeusArtistas })
  const [criando, setCriando] = useState(false)
  const [nome, setNome] = useState('')
  const [tipo, setTipo] = useState<TipoArtista>('danca')
  const [solo, setSolo] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

  useEffect(() => {
    document.title = 'Meus artistas | Evve'
  }, [])

  const sair = async (id: number) => {
    if (!window.confirm('Sair deste grupo? Você deixa de aparecer no perfil dele na hora.')) return
    try {
      await api(`/me/artistas/${id}/sair`, { method: 'POST' })
      queryClient.invalidateQueries({ queryKey: ['meus-artistas'] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível sair agora.')
    }
  }

  const criar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setEnviando(true)
    try {
      const g = await api<DadosArtista & { id: number }>('/me/artistas', { method: 'POST', body: { nome, tipo, solo } })
      navigate(`/meus-artistas/${g.id}`)
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Não foi possível criar.')
    } finally {
      setEnviando(false)
    }
  }

  if (isLoading) return <Carregando />

  return (
    <main className="mx-auto max-w-3xl px-4 py-10">
      <h1 className="text-2xl font-bold text-foreground">Meus artistas</h1>
      <p className="mt-1 text-muted-foreground">Perfis de grupo ou solo para mostrar o seu trabalho na vitrine de artistas e nos eventos.</p>

      {erro && (
        <p role="alert" className="mt-4 text-sm text-destructive">
          {erro}
        </p>
      )}

      <ul className="mt-6 flex flex-col gap-3">
        {data?.map((g) => (
          <li key={g.id} className="flex items-center justify-between gap-3 rounded-2xl border border-border p-4">
            <div className="flex min-w-0 items-center gap-3">
              <span className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted font-semibold text-foreground">
                {g.logo_url ? <img src={g.logo_url} alt="" className="size-full object-cover" /> : g.nome.slice(0, 1).toUpperCase()}
              </span>
              <div className="min-w-0">
                <p className="truncate font-medium text-foreground">{g.nome}</p>
                <p className="text-sm text-muted-foreground">
                  {g.solo ? `Solo · ${nomeTipoArtista(g.tipo)}` : `${nomeTipoArtista(g.tipo)} · ${g.integrantes} ${g.integrantes === 1 ? 'integrante' : 'integrantes'}`}
                  {g.lider ? ' · você é o líder' : ' · você é integrante'}
                </p>
                {g.atracoes_pendentes > 0 && (
                  <p className="text-sm font-medium text-foreground">
                    {g.atracoes_pendentes === 1 ? '1 convite de evento para responder' : `${g.atracoes_pendentes} convites de eventos para responder`}
                  </p>
                )}
              </div>
            </div>
            <div className="flex shrink-0 gap-2">
              <Link to={`/artista/${g.slug}`} className={buttonVariants({ variant: 'outline', size: 'sm' })}>
                Ver
              </Link>
              {g.lider ? (
                <Link to={`/meus-artistas/${g.id}`} className={buttonVariants({ size: 'sm' })}>
                  Gerenciar
                </Link>
              ) : (
                <Button size="sm" variant="ghost" onClick={() => sair(g.id)}>
                  Sair
                </Button>
              )}
            </div>
          </li>
        ))}
      </ul>
      {data?.length === 0 && !criando && <p className="mt-6 text-muted-foreground">Você ainda não tem nenhum perfil de artista.</p>}

      {criando ? (
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>Novo perfil</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={criar} className="flex flex-col gap-4">
              <fieldset className="grid grid-cols-2 gap-2">
                <legend className="mb-2 text-sm font-medium text-foreground">É um grupo ou só você?</legend>
                {[
                  { v: false, t: 'Grupo', d: 'Vários integrantes' },
                  { v: true, t: 'Solo', d: 'Só você' },
                ].map((o) => (
                  <button
                    key={o.t}
                    type="button"
                    aria-pressed={solo === o.v}
                    onClick={() => setSolo(o.v)}
                    className={`rounded-xl border px-3 py-3 text-left transition-colors ${solo === o.v ? 'border-foreground bg-muted' : 'border-border hover:bg-muted'}`}
                  >
                    <span className="block font-medium text-foreground">{o.t}</span>
                    <span className="block text-xs text-muted-foreground">{o.d}</span>
                  </button>
                ))}
              </fieldset>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="nome-artista">{solo ? 'Nome artístico' : 'Nome do grupo'}</Label>
                <Input id="nome-artista" required minLength={2} maxLength={80} value={nome} onChange={(e) => setNome(e.target.value)} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="tipo-artista">O que faz</Label>
                <select
                  id="tipo-artista"
                  className="h-9 rounded-lg border border-border bg-background px-2.5 text-sm"
                  value={tipo}
                  onChange={(e) => setTipo(e.target.value as TipoArtista)}
                >
                  {tiposArtista.map((t) => (
                    <option key={t.valor} value={t.valor}>
                      {t.nome}
                    </option>
                  ))}
                </select>
              </div>
              <p className="text-xs text-muted-foreground">Você será o líder: só você edita o perfil e convida os integrantes. Depois dá para completar foto, Instagram e o portfólio.</p>
              <div className="flex gap-2">
                <Button type="submit" disabled={enviando || nome.trim().length < 2}>
                  {enviando ? 'Criando…' : 'Criar perfil'}
                </Button>
                <Button type="button" variant="ghost" onClick={() => setCriando(false)}>
                  Cancelar
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      ) : (
        <Button className="mt-6" onClick={() => setCriando(true)}>
          <Plus className="size-4" aria-hidden="true" />
          Criar perfil de artista
        </Button>
      )}
    </main>
  )
}
