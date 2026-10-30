import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { VoltarLink } from '@/components/voltar-link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '@/lib/api'
import { criarCircuito, dataCurta, decidirPedidoCircuito, meusCircuitos } from '@/lib/circuito'

// Circuitos do organizador (item 8.5): criar um circuito com temporada e
// aceitar ou recusar os eventos que pedem para virar etapa.
export default function Circuitos() {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['org-circuitos'], queryFn: meusCircuitos })
  const [form, setForm] = useState({ nome: '', descricao: '', temporada_inicio: '', temporada_fim: '' })
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-circuitos'] })

  const criar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setEnviando(true)
    try {
      await criarCircuito(form)
      setForm({ nome: '', descricao: '', temporada_inicio: '', temporada_fim: '' })
      invalidar()
    } catch (er) {
      setErro(er instanceof ApiError ? er.message : 'Não foi possível criar o circuito.')
    } finally {
      setEnviando(false)
    }
  }
  const decidir = async (id: number, aceitar: boolean) => {
    await decidirPedidoCircuito(id, aceitar)
    invalidar()
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar ao painel</VoltarLink>
      <h1 className="text-2xl font-semibold text-foreground">Circuitos</h1>
      <p className="mt-2 text-sm text-muted-foreground text-pretty">
        Um circuito junta vários concursos numa temporada (por exemplo, "Circuito Nordeste de Cosplay 2027"). Cada evento vira uma etapa: os 10 primeiros
        de cada categoria somam pontos num ranking público. Outros organizadores podem pedir para colocar eventos no seu circuito; você aceita ou recusa.
      </p>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>Novo circuito</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={criar} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="c-nome">Nome</Label>
              <Input id="c-nome" required minLength={3} maxLength={120} value={form.nome} onChange={(e) => setForm({ ...form, nome: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="c-desc">Descrição (opcional)</Label>
              <Input id="c-desc" value={form.descricao} onChange={(e) => setForm({ ...form, descricao: e.target.value })} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="c-ini">Temporada começa</Label>
                <Input id="c-ini" type="date" required value={form.temporada_inicio} onChange={(e) => setForm({ ...form, temporada_inicio: e.target.value })} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="c-fim">Temporada termina</Label>
                <Input id="c-fim" type="date" required value={form.temporada_fim} onChange={(e) => setForm({ ...form, temporada_fim: e.target.value })} />
              </div>
            </div>
            {erro && (
              <p role="alert" className="text-sm text-destructive">
                {erro}
              </p>
            )}
            <Button type="submit" disabled={enviando}>
              Criar circuito
            </Button>
          </form>
        </CardContent>
      </Card>

      {data?.map((c) => {
        const pendentes = c.pedidos.filter((p) => p.status === 'pendente')
        const etapas = c.pedidos.filter((p) => p.status === 'aceito')
        return (
          <Card key={c.id} className="mt-6">
            <CardHeader>
              <CardTitle>{c.nome}</CardTitle>
              <p className="text-sm text-muted-foreground">
                {dataCurta(c.temporada_inicio)} a {dataCurta(c.temporada_fim)} ·{' '}
                <Link to={`/circuitos/${c.slug}`} className="underline underline-offset-4">
                  ver página pública
                </Link>
              </p>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              {pendentes.length > 0 && (
                <div className="flex flex-col gap-2">
                  <p className="text-sm font-medium text-foreground">Pedidos para entrar</p>
                  {pendentes.map((p) => (
                    <div key={p.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border px-3 py-2">
                      <Link to={`/e/${p.evento_slug}`} className="text-sm text-foreground underline-offset-4 hover:underline">
                        {p.evento_titulo}
                      </Link>
                      <div className="flex gap-2">
                        <Button size="sm" onClick={() => decidir(p.id, true)}>
                          Aceitar
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => decidir(p.id, false)}>
                          Recusar
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              <div>
                <p className="text-sm font-medium text-foreground">Etapas ({etapas.length})</p>
                {etapas.length === 0 ? (
                  <p className="mt-1 text-sm text-muted-foreground">
                    Nenhuma ainda. No painel de cada evento, use o cartão "Circuito e seletivas" para incluir o evento.
                  </p>
                ) : (
                  <ul className="mt-1 flex flex-col gap-1">
                    {etapas.map((p) => (
                      <li key={p.id} className="text-sm text-foreground">
                        {p.evento_titulo}
                        {p.inicio_em && <span className="text-muted-foreground"> · {new Date(p.inicio_em).toLocaleDateString('pt-BR')}</span>}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </CardContent>
          </Card>
        )
      })}
    </main>
  )
}
