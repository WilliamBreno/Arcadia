import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '@/lib/api'
import { caravanasDoEvento } from '@/lib/caravana'
import { adicionarFotografo, decidirRemocao, fotografos, painelGaleria, removerFotografo } from '@/lib/galeria'

// Cartão do painel do evento (item 8.7): galeria oficial e caravanas.
export function GaleriaCard({ eventoId, slug }: { eventoId: number; slug: string }) {
  const queryClient = useQueryClient()
  const { data: fs } = useQuery({ queryKey: ['org-fotografos', eventoId], queryFn: () => fotografos(eventoId) })
  const { data: painel } = useQuery({ queryKey: ['org-galeria', slug], queryFn: () => painelGaleria(slug) })
  const { data: caravanas } = useQuery({ queryKey: ['org-caravanas', eventoId], queryFn: () => caravanasDoEvento(eventoId) })
  const [email, setEmail] = useState('')
  const [erro, setErro] = useState<string | null>(null)

  const rodar = async (f: () => Promise<unknown>, chaves: unknown[][]) => {
    setErro(null)
    try {
      await f()
      await Promise.all(chaves.map((k) => queryClient.invalidateQueries({ queryKey: k })))
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir agora.')
    }
  }
  const porCidade = new Map<string, number>()
  caravanas?.forEach((c) => porCidade.set(c.cidade, (porCidade.get(c.cidade) ?? 0) + c.membros))

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Galeria oficial e caravanas</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <section aria-labelledby="t-galeria" className="flex flex-col gap-2">
          <h3 id="t-galeria" className="text-sm font-medium text-foreground">
            Galeria ({painel?.fotos.length ?? 0} de {painel?.limite ?? 500} fotos)
          </h3>
          <p className="text-xs text-muted-foreground text-pretty">
            Você e os fotógrafos convidados enviam as fotos pela{' '}
            <Link to={`/e/${slug}#galeria`} className="underline underline-offset-4">
              página do evento
            </Link>
            . Quem aparece numa foto pode se marcar (adultos com perfil de talento) ou pedir para tirar: a foto some na hora e você decide aqui.
          </p>
          {painel && painel.remocoes.length > 0 && (
            <div className="flex flex-col gap-2">
              <p className="text-sm font-medium text-foreground">Pedidos de remoção ({painel.remocoes.length})</p>
              {painel.remocoes.map((r) => (
                <div key={r.id} className="flex flex-wrap items-center gap-3 rounded-lg border border-border p-2">
                  <img src={r.url} alt="" className="size-16 rounded-md object-cover" />
                  <p className="min-w-0 flex-1 text-sm text-foreground">{r.motivo || 'Sem motivo informado.'}</p>
                  <div className="flex gap-2">
                    <Button size="sm" variant="destructive" onClick={() => rodar(() => decidirRemocao(slug, r.id, false), [['org-galeria', slug]])}>
                      Remover foto
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => rodar(() => decidirRemocao(slug, r.id, true), [['org-galeria', slug]])}>
                      Manter
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>

        <section aria-labelledby="t-fotografos" className="flex flex-col gap-2">
          <h3 id="t-fotografos" className="text-sm font-medium text-foreground">
            Fotógrafos convidados
          </h3>
          {fs?.map((f) => (
            <div key={f.usuario_id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
              <span className="text-foreground">
                {f.nome} <span className="text-muted-foreground">· {f.email}</span>
              </span>
              <Button variant="ghost" size="sm" onClick={() => rodar(() => removerFotografo(eventoId, f.usuario_id), [['org-fotografos', eventoId]])}>
                Remover
              </Button>
            </div>
          ))}
          <form
            className="flex flex-wrap items-end gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              rodar(async () => {
                await adicionarFotografo(eventoId, email)
                setEmail('')
              }, [['org-fotografos', eventoId]])
            }}
          >
            <div className="flex min-w-48 flex-1 flex-col gap-1.5">
              <Label htmlFor="foto-email">E-mail do fotógrafo (precisa ter conta no Evve)</Label>
              <Input id="foto-email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            <Button type="submit">Convidar</Button>
          </form>
        </section>

        <section aria-labelledby="t-caravanas" className="flex flex-col gap-2">
          <h3 id="t-caravanas" className="text-sm font-medium text-foreground">
            Caravanas ({caravanas?.length ?? 0})
          </h3>
          {caravanas?.length === 0 && (
            <p className="text-sm text-muted-foreground">Nenhuma caravana ainda. O público cria pela página do evento e manda o link para o grupo.</p>
          )}
          {porCidade.size > 0 && (
            <p className="text-sm text-foreground">
              {[...porCidade.entries()].map(([cidade, n]) => `${cidade}: ${n}`).join(' · ')} pessoa(s)
            </p>
          )}
          {caravanas?.map((c, i) => (
            <div key={`${c.nome}-${i}`} className="rounded-lg border border-border px-3 py-2 text-sm">
              <p className="text-foreground">
                {c.nome} <span className="text-muted-foreground">· {c.cidade}{c.uf && `/${c.uf}`} · responsável {c.responsavel}</span>
              </p>
              <p className="text-xs text-muted-foreground">
                {c.membros} pessoa(s), {c.com_ingresso} com ingresso
                {c.saida_em && ` · saída ${new Date(c.saida_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })}`}
              </p>
            </div>
          ))}
        </section>

        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
