import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { AdminNav } from '@/components/admin-nav'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type ColecaoAdmin = {
  id: number
  slug: string
  titulo: string
  descricao: string
  ordem: number
  ativa: boolean
  eventos: string[]
}

type Rascunho = { id?: number; titulo: string; descricao: string; ordem: number; ativa: boolean; eventos: string }

const vazio: Rascunho = { titulo: '', descricao: '', ordem: 0, ativa: true, eventos: '' }

// Coleções curadas (item 6.2): o admin monta listas de eventos colando os
// links (ou slugs) dos eventos, um por linha. Aparecem na página inicial.
export default function AdminColecoes() {
  const queryClient = useQueryClient()
  const [rascunho, setRascunho] = useState<Rascunho>(vazio)
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)
  const [salvando, setSalvando] = useState(false)

  const { data, error } = useQuery({ queryKey: ['admin-colecoes'], queryFn: () => api<ColecaoAdmin[]>('/admin/colecoes') })

  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setAviso(null)
    setSalvando(true)
    try {
      const corpo = {
        titulo: rascunho.titulo,
        descricao: rascunho.descricao,
        ordem: rascunho.ordem,
        ativa: rascunho.ativa,
        eventos: rascunho.eventos.split(/\s+/).filter(Boolean),
      }
      const resp = await api<{ slug: string; ignorados: string[] }>(rascunho.id ? `/admin/colecoes/${rascunho.id}` : '/admin/colecoes', {
        method: rascunho.id ? 'PUT' : 'POST',
        body: corpo,
      })
      setAviso(
        resp.ignorados.length
          ? `Salva. Não encontramos estes eventos: ${resp.ignorados.join(', ')}`
          : 'Coleção salva.',
      )
      setRascunho(vazio)
      queryClient.invalidateQueries({ queryKey: ['admin-colecoes'] })
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Erro ao salvar')
    } finally {
      setSalvando(false)
    }
  }

  const excluir = async (id: number) => {
    setErro(null)
    try {
      await api(`/admin/colecoes/${id}`, { method: 'DELETE' })
      if (rascunho.id === id) setRascunho(vazio)
      queryClient.invalidateQueries({ queryKey: ['admin-colecoes'] })
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Erro ao excluir')
    }
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-destructive">Acesso restrito ao admin da plataforma.</p>
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <AdminNav />
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Coleções</h1>
      <p className="mb-6 text-sm text-muted-foreground">
        Listas de eventos escolhidas por você, mostradas na página inicial. Eventos que já terminaram ou foram cancelados somem sozinhos.
      </p>

      <Card className="mb-8">
        <CardContent className="py-5">
          <form onSubmit={salvar} className="flex flex-col gap-4">
            <h2 className="font-medium text-foreground">{rascunho.id ? 'Editar coleção' : 'Nova coleção'}</h2>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="titulo">Título</Label>
              <Input id="titulo" required minLength={2} maxLength={120} value={rascunho.titulo} placeholder="Rolês geek do mês"
                onChange={(e) => setRascunho({ ...rascunho, titulo: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="descricao">Descrição (opcional)</Label>
              <Input id="descricao" maxLength={500} value={rascunho.descricao}
                onChange={(e) => setRascunho({ ...rascunho, descricao: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="eventos">Eventos (cole o link ou o código de cada evento, um por linha, na ordem)</Label>
              <Textarea id="eventos" rows={6} value={rascunho.eventos} placeholder={'https://evve-delta.vercel.app/e/concurso-cosplay\nfestival-de-danca'}
                onChange={(e) => setRascunho({ ...rascunho, eventos: e.target.value })} />
            </div>
            <div className="flex flex-wrap items-center gap-4">
              <div className="flex items-center gap-2">
                <Label htmlFor="ordem">Ordem na página inicial</Label>
                <Input id="ordem" type="number" className="w-20" value={rascunho.ordem}
                  onChange={(e) => setRascunho({ ...rascunho, ordem: Number(e.target.value) || 0 })} />
              </div>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input type="checkbox" checked={rascunho.ativa} onChange={(e) => setRascunho({ ...rascunho, ativa: e.target.checked })} />
                Ativa (visível ao público)
              </label>
            </div>
            {erro && <p className="text-sm text-destructive">{erro}</p>}
            {aviso && <p className="text-sm text-muted-foreground">{aviso}</p>}
            <div className="flex gap-2">
              <Button type="submit" disabled={salvando}>{salvando ? 'Salvando…' : 'Salvar coleção'}</Button>
              {rascunho.id && (
                <Button type="button" variant="outline" onClick={() => setRascunho(vazio)}>
                  Cancelar edição
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      {data?.length === 0 && <p className="text-muted-foreground">Nenhuma coleção criada.</p>}
      <div className="flex flex-col gap-3">
        {data?.map((c) => (
          <Card key={c.id}>
            <CardContent className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <p className="font-medium text-foreground">
                  {c.titulo} {!c.ativa && <span className="text-xs font-normal text-muted-foreground">(inativa)</span>}
                </p>
                <p className="text-xs text-muted-foreground">
                  {c.eventos.length} evento(s) · ordem {c.ordem} ·{' '}
                  <Link to={`/c/${c.slug}`} className="underline underline-offset-4">/c/{c.slug}</Link>
                </p>
              </div>
              <div className="flex gap-2">
                <Button size="sm" variant="outline"
                  onClick={() => setRascunho({ id: c.id, titulo: c.titulo, descricao: c.descricao, ordem: c.ordem, ativa: c.ativa, eventos: c.eventos.join('\n') })}>
                  Editar
                </Button>
                <Button size="sm" variant="ghost" onClick={() => excluir(c.id)}>Excluir</Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
