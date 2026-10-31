import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Ban, ChevronLeft, ChevronRight, Circle, Plus } from 'lucide-react'

import { api, ApiError } from '@/lib/api'
import type { AgendaArtista } from '@/lib/artistas'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const isoLocal = (data: Date) => `${data.getFullYear()}-${String(data.getMonth() + 1).padStart(2, '0')}-${String(data.getDate()).padStart(2, '0')}`
const dataLocal = (iso: string) => new Date(`${iso}T12:00:00`)
const diaCompleto = (iso: string) => dataLocal(iso).toLocaleDateString('pt-BR', { day: 'numeric', month: 'long', year: 'numeric' })
const diaCurto = (iso: string) => iso.split('-').reverse().join('/')
const entradaNova = (dia: string): AgendaArtista => ({ id: 0, tipo: 'apresentacao', titulo: '', inicio: dia, fim: dia, publico: false })
const mensagem = (e: unknown) => e instanceof ApiError ? e.message : 'Não foi possível concluir. Tente novamente.'
const ocorreNoDia = (a: AgendaArtista, dia: string) => a.inicio <= dia && a.fim >= dia

export function ArtistaAgenda({ id }: { id: number }) {
  const qc = useQueryClient()
  const { data = [], error, isLoading } = useQuery({
    queryKey: ['agenda-artista', id],
    queryFn: () => api<AgendaArtista[]>(`/me/artistas/${id}/agenda`),
  })
  const [selecionado, setSelecionado] = useState(() => isoLocal(new Date()))
  const [mes, setMes] = useState(() => new Date(new Date().getFullYear(), new Date().getMonth(), 1, 12))
  const [d, setD] = useState(() => entradaNova(isoLocal(new Date())))
  const [erro, setErro] = useState('')
  const [ok, setOk] = useState('')
  const [ocupado, setOcupado] = useState(false)
  const hoje = isoLocal(new Date())
  const primeiro = new Date(mes.getFullYear(), mes.getMonth(), 1, 12)
  const ultimo = new Date(mes.getFullYear(), mes.getMonth() + 1, 0, 12)
  const quantidadeDias = Math.ceil((primeiro.getDay() + ultimo.getDate()) / 7) * 7
  const dias = Array.from({ length: quantidadeDias }, (_, i) => new Date(mes.getFullYear(), mes.getMonth(), 1 - primeiro.getDay() + i, 12))
  const atividadesMes = data.filter(a => a.inicio <= isoLocal(ultimo) && a.fim >= isoLocal(primeiro))
  const atividadesDia = data.filter(a => ocorreNoDia(a, selecionado))
  const apresentacoes = atividadesMes.filter(a => a.tipo === 'apresentacao').length
  const bloqueios = atividadesMes.filter(a => a.tipo === 'bloqueio').length

  const selecionar = (dia: string) => {
    setSelecionado(dia)
    setMes(new Date(dataLocal(dia).getFullYear(), dataLocal(dia).getMonth(), 1, 12))
    setD(entradaNova(dia))
    setErro('')
    setOk('')
  }
  const navegar = (passo: number) => selecionar(isoLocal(new Date(mes.getFullYear(), mes.getMonth() + passo, 1, 12)))
  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro('')
    setOk('')
    setOcupado(true)
    try {
      await api(`/me/artistas/${id}/agenda${d.id ? `/${d.id}` : ''}`, { method: d.id ? 'PUT' : 'POST', body: d })
      selecionar(d.inicio)
      setOk(d.id ? 'Atividade atualizada.' : 'Atividade adicionada à agenda.')
      await qc.invalidateQueries({ queryKey: ['agenda-artista', id] })
    } catch (e) {
      setErro(mensagem(e))
    } finally {
      setOcupado(false)
    }
  }
  const remover = async (aid: number) => {
    if (!window.confirm('Remover esta entrada da agenda?')) return
    setErro('')
    setOk('')
    setOcupado(true)
    try {
      await api(`/me/artistas/${id}/agenda/${aid}`, { method: 'DELETE' })
      if (d.id === aid) setD(entradaNova(selecionado))
      setOk('Atividade removida.')
      await qc.invalidateQueries({ queryKey: ['agenda-artista', id] })
    } catch (e) {
      setErro(mensagem(e))
    } finally {
      setOcupado(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Agenda</CardTitle>
        <p className="text-sm text-muted-foreground">Veja o mês inteiro e clique em um dia para marcar uma apresentação ou um bloqueio.</p>
      </CardHeader>
      <CardContent className="flex min-w-0 flex-col gap-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 aria-live="polite" className="text-lg font-semibold capitalize">{mes.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' })}</h2>
            <p className="text-xs text-muted-foreground">{apresentacoes} {apresentacoes === 1 ? 'apresentação' : 'apresentações'} · {bloqueios} {bloqueios === 1 ? 'bloqueio' : 'bloqueios'}</p>
          </div>
          <div className="flex items-center gap-1">
            <Button type="button" size="sm" variant="outline" disabled={ocupado} onClick={() => selecionar(hoje)}>Hoje</Button>
            <Button type="button" size="icon" variant="ghost" aria-label="Mês anterior" disabled={ocupado} onClick={() => navegar(-1)}><ChevronLeft className="size-4" /></Button>
            <Button type="button" size="icon" variant="ghost" aria-label="Próximo mês" disabled={ocupado} onClick={() => navegar(1)}><ChevronRight className="size-4" /></Button>
          </div>
        </div>

        {error && <p role="alert" className="text-sm text-destructive">{mensagem(error)}</p>}
        <div aria-busy={isLoading}>
          <div className="mb-2 grid grid-cols-7 text-center text-xs text-muted-foreground" aria-hidden="true">
            {['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'].map(dia => <span key={dia}>{dia}</span>)}
          </div>
          <div className="grid grid-cols-7 gap-px overflow-hidden rounded-xl border border-border bg-border" role="group" aria-label="Calendário da agenda">
            {dias.map(dataDia => {
              const dia = isoLocal(dataDia)
              const atividades = data.filter(a => ocorreNoDia(a, dia))
              const mesmoMes = dataDia.getMonth() === mes.getMonth()
              const marcado = dia === selecionado
              const temApresentacao = atividades.some(a => a.tipo === 'apresentacao')
              const temBloqueio = atividades.some(a => a.tipo === 'bloqueio')
              return (
                <button
                  key={dia}
                  type="button"
                  aria-label={`${diaCompleto(dia)}${dia === hoje ? ', hoje' : ''}, ${atividades.length} ${atividades.length === 1 ? 'atividade' : 'atividades'}`}
                  aria-pressed={marcado}
                  aria-current={dia === hoje ? 'date' : undefined}
                  disabled={ocupado}
                  onClick={() => selecionar(dia)}
                  className={`flex min-h-20 min-w-0 flex-col gap-1 overflow-hidden p-1.5 text-left transition-colors focus-visible:z-10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary sm:min-h-28 sm:p-2 ${marcado ? 'bg-accent ring-2 ring-inset ring-primary' : mesmoMes ? 'bg-card hover:bg-muted' : 'bg-muted/60 text-muted-foreground hover:bg-muted'}`}
                >
                  <span className={`flex size-6 shrink-0 items-center justify-center rounded-full text-xs ${dia === hoje ? 'bg-foreground font-semibold text-background' : 'font-medium'}`}>{dataDia.getDate()}</span>
                  <span className="flex flex-wrap items-center gap-1 sm:hidden" aria-hidden="true">
                    {temApresentacao && <Circle className="size-2 fill-current" />}
                    {temBloqueio && <Ban className="size-3 text-muted-foreground" />}
                    {atividades.length > 1 && <span className="text-xs">{atividades.length}</span>}
                  </span>
                  <span className="hidden w-full min-w-0 flex-col gap-1 sm:flex" aria-hidden="true">
                    {atividades.slice(0, 2).map(a => <span key={a.id} className="flex min-w-0 items-center gap-1 text-xs">
                      {a.tipo === 'bloqueio' ? <Ban className="size-3 shrink-0 text-muted-foreground" /> : <Circle className="size-2 shrink-0 fill-current" />}
                      <span className="truncate">{a.titulo || 'Indisponível'}</span>
                    </span>)}
                    {atividades.length > 2 && <span className="text-xs text-muted-foreground">+{atividades.length - 2}</span>}
                  </span>
                </button>
              )
            })}
          </div>
          <div className="mt-3 flex flex-wrap gap-4 text-xs text-muted-foreground">
            <span className="flex items-center gap-1.5"><Circle className="size-2 fill-current" aria-hidden="true" />Apresentação</span>
            <span className="flex items-center gap-1.5"><Ban className="size-3" aria-hidden="true" />Bloqueio</span>
          </div>
          {isLoading && <p role="status" className="mt-2 text-sm text-muted-foreground">Carregando atividades…</p>}
        </div>

        <section aria-label="Atividades do dia" className="flex flex-col gap-3">
          <h3 className="font-semibold">{diaCompleto(selecionado)}</h3>
          {atividadesDia.length === 0 && !isLoading && !error && <p className="text-sm text-muted-foreground">Nenhuma atividade neste dia. Use o formulário abaixo para marcar.</p>}
          {atividadesDia.map(a => (
            <div key={a.id} className="flex flex-col gap-2 rounded-lg border border-border p-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <p className="break-words text-sm font-medium">{a.titulo || 'Indisponível'}</p>
                <p className="text-xs text-muted-foreground">{a.tipo === 'bloqueio' ? 'Bloqueio' : 'Apresentação'} · {a.publico ? 'Pública' : 'Privada'}</p>
                {a.inicio !== a.fim && <p className="text-xs text-muted-foreground">De {diaCurto(a.inicio)} a {diaCurto(a.fim)}</p>}
              </div>
              <div className="flex shrink-0 gap-2">
                <Button type="button" size="sm" variant="outline" disabled={ocupado} onClick={() => { setD(a); setErro(''); setOk('') }}>Editar</Button>
                <Button type="button" size="sm" variant="ghost" disabled={ocupado} onClick={() => remover(a.id)}>Remover</Button>
              </div>
            </div>
          ))}
        </section>

        <form onSubmit={salvar} className="flex flex-col gap-4 rounded-xl bg-muted/50 p-3 sm:p-4">
          <h3 className="flex items-center gap-2 text-sm font-semibold"><Plus className="size-4" aria-hidden="true" />{d.id ? 'Editar atividade' : 'Marcar atividade'}</h3>
          <fieldset disabled={ocupado} className="flex min-w-0 flex-col gap-4">
            <legend className="sr-only">Dados da atividade</legend>
            <div className="grid grid-cols-2 gap-2">
              {([{ tipo: 'apresentacao', nome: 'Apresentação', Icone: Circle }, { tipo: 'bloqueio', nome: 'Bloqueio', Icone: Ban }] as const).map(({ tipo, nome, Icone }) => (
                <button key={tipo} type="button" aria-pressed={d.tipo === tipo} onClick={() => setD({ ...d, tipo })} className={`flex min-w-0 items-center justify-center gap-2 rounded-lg border px-2 py-3 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary ${d.tipo === tipo ? 'border-foreground bg-background font-medium' : 'border-border text-muted-foreground hover:bg-background'}`}>
                  <Icone className="size-4 shrink-0" aria-hidden="true" />{nome}
                </button>
              ))}
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="agenda-titulo">{d.tipo === 'bloqueio' ? 'Motivo (privado, opcional)' : 'Título da apresentação'}</Label>
              <Input id="agenda-titulo" maxLength={120} required={d.tipo === 'apresentacao'} value={d.titulo} onChange={e => setD({ ...d, titulo: e.target.value })} />
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="flex min-w-0 flex-col gap-2"><Label htmlFor="agenda-inicio">Início</Label><Input id="agenda-inicio" type="date" required value={d.inicio} onChange={e => setD({ ...d, inicio: e.target.value, fim: d.fim < e.target.value ? e.target.value : d.fim })} /></div>
              <div className="flex min-w-0 flex-col gap-2"><Label htmlFor="agenda-fim">Fim</Label><Input id="agenda-fim" type="date" required min={d.inicio} value={d.fim} onChange={e => setD({ ...d, fim: e.target.value })} /></div>
            </div>
            <p className="text-xs text-muted-foreground">Para mais de um dia, ajuste o fim do período. O início e o fim ficam marcados no calendário.</p>
            <label className="flex items-start gap-2 text-sm"><input type="checkbox" checked={d.publico} onChange={e => setD({ ...d, publico: e.target.checked })} />Mostrar no perfil público</label>
            <p className="text-xs text-muted-foreground">{d.tipo === 'bloqueio' ? 'Se publicado, o bloqueio mostra apenas “Indisponível”, sem o motivo.' : 'Cada nova atividade começa privada. Publique somente se quiser exibi-la no perfil.'}</p>
          </fieldset>
          {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
          {ok && <p role="status" className="text-sm">{ok}</p>}
          <div className="flex flex-wrap gap-2">
            <Button type="submit" disabled={ocupado || isLoading || !!error}>{ocupado ? 'Salvando…' : d.id ? 'Salvar alterações' : 'Adicionar data'}</Button>
            {d.id > 0 && <Button type="button" variant="ghost" disabled={ocupado} onClick={() => setD(entradaNova(selecionado))}>Cancelar edição</Button>}
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
