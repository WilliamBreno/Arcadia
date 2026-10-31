import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { AdminNav } from '@/components/admin-nav'
import { IconeColecao, iconesColecao } from '@/components/icone-colecao'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type Regra = 'manual' | 'gratuitos' | 'esta_semana' | 'online' | 'categoria'

type ColecaoAdmin = {
  id: number
  slug: string
  titulo: string
  descricao: string
  ordem: number
  ativa: boolean
  eventos: string[]
  // Item 11.7.
  icone: string
  exibir_de: string | null
  exibir_ate: string | null
  regra: Regra
  regra_valor: string
  aberta_organizador: boolean
  na_home: boolean
  visivel_agora: boolean
  marcados_organizador: { slug: string; titulo: string; bloqueado: boolean }[]
}

type Rascunho = {
  id?: number
  titulo: string
  descricao: string
  ordem: number
  ativa: boolean
  eventos: string
  icone: string
  exibir_de: string
  exibir_ate: string
  regra: Regra
  regra_valor: string
  aberta_organizador: boolean
  na_home: boolean
}

const vazio: Rascunho = {
  titulo: '',
  descricao: '',
  ordem: 0,
  ativa: true,
  eventos: '',
  icone: 'estrela',
  exibir_de: '',
  exibir_ate: '',
  regra: 'manual',
  regra_valor: '',
  aberta_organizador: false,
  na_home: true,
}

const regras: { valor: Regra; nome: string }[] = [
  { valor: 'manual', nome: 'Escolhidos à mão (e pelos organizadores, se aberta)' },
  { valor: 'gratuitos', nome: 'Automática: eventos gratuitos' },
  { valor: 'esta_semana', nome: 'Automática: próximos 7 dias' },
  { valor: 'online', nome: 'Automática: eventos online' },
  { valor: 'categoria', nome: 'Automática: uma categoria' },
]

const nomeRegra = (r: Regra) => regras.find((x) => x.valor === r)?.nome ?? r
const dataBR = (d: string | null) => (d ? d.split('-').reverse().join('/') : '')

// Coleções (item 6.2) e temas da época (item 11.7): o admin monta listas
// de eventos, abre temas para os organizadores marcarem os próprios eventos
// e cria coleções automáticas. Aparecem na página inicial e em /colecoes.
export default function AdminColecoes() {
  const queryClient = useQueryClient()
  const [rascunho, setRascunho] = useState<Rascunho>(vazio)
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)
  const [salvando, setSalvando] = useState(false)

  const { data, error } = useQuery({ queryKey: ['admin-colecoes'], queryFn: () => api<ColecaoAdmin[]>('/admin/colecoes') })
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['admin-colecoes'] })
  const mudar = (parcial: Partial<Rascunho>) => setRascunho({ ...rascunho, ...parcial })

  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setAviso(null)
    setSalvando(true)
    try {
      const manual = rascunho.regra === 'manual'
      const corpo = {
        titulo: rascunho.titulo,
        descricao: rascunho.descricao,
        ordem: rascunho.ordem,
        ativa: rascunho.ativa,
        eventos: manual ? rascunho.eventos.split(/\s+/).filter(Boolean) : [],
        icone: rascunho.icone,
        exibir_de: rascunho.exibir_de,
        exibir_ate: rascunho.exibir_ate,
        regra: rascunho.regra,
        regra_valor: rascunho.regra_valor,
        aberta_organizador: manual && rascunho.aberta_organizador,
        na_home: rascunho.na_home,
      }
      const resp = await api<{ slug: string; ignorados: string[] }>(rascunho.id ? `/admin/colecoes/${rascunho.id}` : '/admin/colecoes', {
        method: rascunho.id ? 'PUT' : 'POST',
        body: corpo,
      })
      setAviso(resp.ignorados.length ? `Salva. Não encontramos estes eventos: ${resp.ignorados.join(', ')}` : 'Coleção salva.')
      setRascunho(vazio)
      invalidar()
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Erro ao salvar')
    } finally {
      setSalvando(false)
    }
  }

  const excluir = async (id: number) => {
    if (!window.confirm('Excluir esta coleção? Os eventos não são afetados.')) return
    setErro(null)
    try {
      await api(`/admin/colecoes/${id}`, { method: 'DELETE' })
      if (rascunho.id === id) setRascunho(vazio)
      invalidar()
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Erro ao excluir')
    }
  }

  const bloquear = async (colecaoId: number, slug: string, bloqueado: boolean) => {
    setErro(null)
    try {
      await api(`/admin/colecoes/${colecaoId}/marcados/${encodeURIComponent(slug)}`, { method: 'POST', body: { bloqueado } })
      invalidar()
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Erro ao salvar')
    }
  }

  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-destructive">Acesso restrito ao admin da plataforma.</p>
  }

  const manual = rascunho.regra === 'manual'

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <AdminNav />
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Coleções e temas</h1>
      <p className="mb-6 text-sm text-muted-foreground">
        Listas de eventos com ícone, mostradas em <Link to="/colecoes" className="underline underline-offset-4">/colecoes</Link> e na página inicial.
        Com um período, a coleção só aparece na época (Halloween, Réveillon...). Eventos que já terminaram somem sozinhos.
      </p>

      <Card className="mb-8">
        <CardContent className="py-5">
          <form onSubmit={salvar} className="flex flex-col gap-4">
            <h2 className="font-medium text-foreground">{rascunho.id ? 'Editar coleção' : 'Nova coleção'}</h2>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="titulo">Título</Label>
              <Input id="titulo" required minLength={2} maxLength={120} value={rascunho.titulo} placeholder="Halloween" onChange={(e) => mudar({ titulo: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="descricao">Descrição (opcional)</Label>
              <Input id="descricao" maxLength={500} value={rascunho.descricao} onChange={(e) => mudar({ descricao: e.target.value })} />
            </div>

            <fieldset className="flex flex-col gap-2">
              <legend className="mb-1 text-sm font-medium text-foreground">Ícone</legend>
              <div className="grid grid-cols-5 gap-2 sm:grid-cols-10">
                {iconesColecao.map((nome) => (
                  <button
                    key={nome}
                    type="button"
                    aria-label={nome}
                    aria-pressed={rascunho.icone === nome}
                    onClick={() => mudar({ icone: nome })}
                    className={`flex aspect-square items-center justify-center rounded-xl border transition-colors ${
                      rascunho.icone === nome ? 'border-foreground bg-muted' : 'border-border hover:bg-muted'
                    }`}
                  >
                    <IconeColecao nome={nome} className="size-8" />
                  </button>
                ))}
              </div>
            </fieldset>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="regra">Quais eventos entram</Label>
              <select
                id="regra"
                className="h-9 rounded-lg border border-border bg-background px-2.5 text-sm"
                value={rascunho.regra}
                onChange={(e) => mudar({ regra: e.target.value as Regra })}
              >
                {regras.map((r) => (
                  <option key={r.valor} value={r.valor}>
                    {r.nome}
                  </option>
                ))}
              </select>
            </div>
            {rascunho.regra === 'categoria' && (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="regra_valor">Categoria (código, ex.: geek, danca, musica)</Label>
                <Input id="regra_valor" maxLength={60} value={rascunho.regra_valor} onChange={(e) => mudar({ regra_valor: e.target.value })} />
              </div>
            )}
            {manual && (
              <>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="eventos">Eventos escolhidos por você (link ou código, um por linha, na ordem)</Label>
                  <Textarea
                    id="eventos"
                    rows={5}
                    value={rascunho.eventos}
                    placeholder={'https://evve-delta.vercel.app/e/concurso-cosplay\nfestival-de-danca'}
                    onChange={(e) => mudar({ eventos: e.target.value })}
                  />
                </div>
                <label className="flex items-start gap-2 text-sm text-foreground">
                  <input type="checkbox" className="mt-0.5" checked={rascunho.aberta_organizador} onChange={(e) => mudar({ aberta_organizador: e.target.checked })} />
                  <span>
                    Tema aberto aos organizadores
                    <span className="block text-xs text-muted-foreground">Eles marcam os próprios eventos neste tema (até 3 temas por evento). Você pode tirar um evento depois.</span>
                  </span>
                </label>
              </>
            )}

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="exibir_de">Aparece a partir de</Label>
                <Input id="exibir_de" type="date" value={rascunho.exibir_de} onChange={(e) => mudar({ exibir_de: e.target.value })} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="exibir_ate">Aparece até</Label>
                <Input id="exibir_ate" type="date" value={rascunho.exibir_ate} onChange={(e) => mudar({ exibir_ate: e.target.value })} />
              </div>
            </div>
            <p className="-mt-2 text-xs text-muted-foreground">Em branco: aparece sempre.</p>

            <div className="flex flex-wrap items-center gap-x-5 gap-y-3">
              <div className="flex items-center gap-2">
                <Label htmlFor="ordem">Ordem</Label>
                <Input id="ordem" type="number" className="w-20" value={rascunho.ordem} onChange={(e) => mudar({ ordem: Number(e.target.value) || 0 })} />
              </div>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input type="checkbox" checked={rascunho.ativa} onChange={(e) => mudar({ ativa: e.target.checked })} />
                Ativa
              </label>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input type="checkbox" checked={rascunho.na_home} onChange={(e) => mudar({ na_home: e.target.checked })} />
                Trilho na página inicial
              </label>
            </div>
            {erro && (
              <p role="alert" className="text-sm text-destructive">
                {erro}
              </p>
            )}
            {aviso && (
              <p role="status" className="text-sm text-muted-foreground">
                {aviso}
              </p>
            )}
            <div className="flex gap-2">
              <Button type="submit" disabled={salvando}>
                {salvando ? 'Salvando…' : 'Salvar coleção'}
              </Button>
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
            <CardContent className="flex flex-col gap-3 py-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="flex min-w-0 items-center gap-3">
                  <span className="flex size-12 shrink-0 items-center justify-center rounded-full bg-muted text-foreground">
                    <IconeColecao nome={c.icone} className="size-7" />
                  </span>
                  <div className="min-w-0">
                    <p className="font-medium text-foreground">
                      {c.titulo}{' '}
                      <span className="text-xs font-normal text-muted-foreground">
                        {!c.ativa ? '(inativa)' : c.visivel_agora ? '(no ar)' : '(fora do período)'}
                      </span>
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {nomeRegra(c.regra)}
                      {c.regra === 'manual' && ` · ${c.eventos.length} escolhido(s)`}
                      {c.aberta_organizador && ` · aberto aos organizadores (${c.marcados_organizador.filter((m) => !m.bloqueado).length})`}
                      {(c.exibir_de || c.exibir_ate) && ` · ${dataBR(c.exibir_de) || '…'} a ${dataBR(c.exibir_ate) || '…'}`} ·{' '}
                      <Link to={`/c/${c.slug}`} className="underline underline-offset-4">
                        /c/{c.slug}
                      </Link>
                    </p>
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() =>
                      setRascunho({
                        id: c.id,
                        titulo: c.titulo,
                        descricao: c.descricao,
                        ordem: c.ordem,
                        ativa: c.ativa,
                        eventos: c.eventos.join('\n'),
                        icone: c.icone,
                        exibir_de: c.exibir_de ?? '',
                        exibir_ate: c.exibir_ate ?? '',
                        regra: c.regra,
                        regra_valor: c.regra_valor,
                        aberta_organizador: c.aberta_organizador,
                        na_home: c.na_home,
                      })
                    }
                  >
                    Editar
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => excluir(c.id)}>
                    Excluir
                  </Button>
                </div>
              </div>
              {c.marcados_organizador.length > 0 && (
                <details className="rounded-lg border border-border px-3 py-2">
                  <summary className="cursor-pointer text-sm text-foreground">Marcados pelos organizadores ({c.marcados_organizador.length})</summary>
                  <ul className="mt-2 flex flex-col gap-1.5">
                    {c.marcados_organizador.map((m) => (
                      <li key={m.slug} className="flex items-center justify-between gap-3 text-sm">
                        <Link to={`/e/${m.slug}`} className={`truncate underline-offset-4 hover:underline ${m.bloqueado ? 'text-muted-foreground line-through' : 'text-foreground'}`}>
                          {m.titulo}
                        </Link>
                        <Button size="sm" variant="ghost" onClick={() => bloquear(c.id, m.slug, !m.bloqueado)}>
                          {m.bloqueado ? 'Devolver' : 'Tirar do tema'}
                        </Button>
                      </li>
                    ))}
                  </ul>
                </details>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
