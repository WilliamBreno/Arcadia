import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, baixarArquivo } from '@/lib/api'
import type { AgendaArtista, GestaoArtista } from '@/lib/artistas'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const mensagem = (e: unknown) => e instanceof ApiError ? e.message : 'Não foi possível concluir. Tente novamente.'
const dia = (d: string) => d.split('-').reverse().join('/')

export function ArtistaGestao({ id, solo }: { id: number; solo: boolean }) {
  const qc = useQueryClient()
  const { data, error } = useQuery({ queryKey: ['gestao-artista', id], queryFn: () => api<GestaoArtista>(`/me/artistas/${id}/gestao`) })
  const [email, setEmail] = useState('')
  const [papel, setPapel] = useState('cogestor')
  const [antigo, setAntigo] = useState(true)
  const [erro, setErro] = useState('')
  const [ok, setOk] = useState('')
  const [ocupado, setOcupado] = useState(false)
  const recarregar = () => {
    qc.invalidateQueries({ queryKey: ['gestao-artista', id] })
    qc.invalidateQueries({ queryKey: ['meus-artistas'] })
  }
  const convidar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(''); setOk(''); setOcupado(true)
    try {
      await api(`/me/artistas/${id}/gestao`, { method: 'POST', body: { email, papel, antigo_cogestor: antigo } })
      setEmail(''); setOk('Convite enviado. A pessoa precisa aceitar com a própria conta em até 7 dias.'); recarregar()
    } catch (e) { setErro(mensagem(e)) } finally { setOcupado(false) }
  }
  const remover = async (alvo: number, convite = false) => {
    if (!window.confirm(convite ? 'Revogar este convite?' : 'Remover o acesso deste cogestor?')) return
    setErro(''); setOcupado(true)
    try {
      await api(`/me/artistas/${id}/gestao/${alvo}${convite ? '?convite=1' : ''}`, { method: 'DELETE' }); recarregar()
    } catch (e) { setErro(mensagem(e)) } finally { setOcupado(false) }
  }
  return <Card>
    <CardHeader><CardTitle>Liderança e cogestores</CardTitle>
      <p className="text-sm text-muted-foreground">Cada pessoa usa sua própria conta e senha. Cogestores editam perfil, integrantes, portfólio e agenda. Só o líder administra os acessos e confirma atrações.</p>
    </CardHeader>
    <CardContent className="flex flex-col gap-4">
      {error && <p role="alert">{mensagem(error)}</p>}
      {data?.gestores.map(g => <div key={g.usuario_id} className="flex flex-wrap items-center justify-between gap-2 border-b border-border pb-2 text-sm">
        <span className="break-all">{g.email} · {g.lider ? 'Líder' : 'Cogestor'}</span>
        {data.lider && !g.lider && <Button variant="ghost" size="sm" disabled={ocupado} onClick={() => remover(g.usuario_id)}>Remover acesso</Button>}
      </div>)}
      {data?.lider && <>
        {data.convites.map(c => <div key={c.id} className="flex flex-wrap items-center justify-between gap-2 text-sm">
          <span className="break-all">{c.email} · {c.papel === 'lider' ? 'Liderança' : 'Cogestor'} · convite até {new Date(c.expira_em).toLocaleDateString('pt-BR')}</span>
          <Button size="sm" variant="ghost" disabled={ocupado} onClick={() => remover(c.id, true)}>Revogar</Button>
        </div>)}
        <form onSubmit={convidar} className="flex flex-col gap-3">
          <Label htmlFor="gestao-email">E-mail da pessoa</Label>
          <Input id="gestao-email" type="email" required maxLength={255} value={email} onChange={e => setEmail(e.target.value)} />
          <Label htmlFor="gestao-papel">Acesso</Label>
          <select id="gestao-papel" className="h-9 rounded-lg border border-border bg-background px-2 text-sm" value={papel} onChange={e => setPapel(e.target.value)}>
            <option value="cogestor">Convidar cogestor</option>
            {!solo && <option value="lider">Transferir liderança</option>}
          </select>
          {papel === 'lider' && <>
            <label className="flex items-start gap-2 text-sm"><input type="checkbox" checked={antigo} onChange={e => setAntigo(e.target.checked)} />Continuar como cogestor após o aceite</label>
            <p className="text-sm text-muted-foreground">A transferência só acontece quando a pessoa aceitar. Você perderá o controle da liderança. Desmarcado, você sai da gestão; seu vínculo como integrante permanece.</p>
          </>}
          {solo && <p className="text-xs text-muted-foreground">A liderança do perfil solo pertence ao próprio artista. Você pode convidar cogestores para ajudar.</p>}
          <Button type="submit" disabled={ocupado} className="self-start">{ocupado ? 'Enviando…' : 'Enviar convite'}</Button>
        </form>
      </>}
      {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
      {ok && <p role="status" className="text-sm">{ok}</p>}
    </CardContent>
  </Card>
}

const novaEntrada: AgendaArtista = { id: 0, tipo: 'apresentacao', titulo: '', inicio: '', fim: '', publico: false }
export function ArtistaAgenda({ id }: { id: number }) {
  const qc = useQueryClient()
  const { data, error } = useQuery({ queryKey: ['agenda-artista', id], queryFn: () => api<AgendaArtista[]>(`/me/artistas/${id}/agenda`) })
  const [d, setD] = useState(novaEntrada)
  const [erro, setErro] = useState('')
  const [ocupado, setOcupado] = useState(false)
  const recarregar = () => qc.invalidateQueries({ queryKey: ['agenda-artista', id] })
  const salvar = async (e: React.FormEvent) => {
    e.preventDefault(); setErro(''); setOcupado(true)
    try {
      await api(`/me/artistas/${id}/agenda${d.id ? `/${d.id}` : ''}`, { method: d.id ? 'PUT' : 'POST', body: d })
      setD(novaEntrada); recarregar()
    } catch (e) { setErro(mensagem(e)) } finally { setOcupado(false) }
  }
  const remover = async (aid: number) => {
    if (!window.confirm('Remover esta entrada da agenda?')) return
    setErro(''); setOcupado(true)
    try {
      await api(`/me/artistas/${id}/agenda/${aid}`, { method: 'DELETE' })
      if (d.id === aid) setD(novaEntrada)
      recarregar()
    } catch (e) { setErro(mensagem(e)) } finally { setOcupado(false) }
  }
  return <Card>
    <CardHeader><CardTitle>Agenda</CardTitle><p className="text-sm text-muted-foreground">Apresentações e datas bloqueadas. Cada entrada começa privada. Os dias de início e fim fazem parte do período.</p></CardHeader>
    <CardContent className="flex flex-col gap-4">
      {error && <p role="alert">{mensagem(error)}</p>}
      {data?.length === 0 && <p className="text-sm text-muted-foreground">Nenhuma data adicionada.</p>}
      {data?.map(a => <div key={a.id} className="flex flex-col gap-2 border-b border-border pb-3 sm:flex-row sm:items-center sm:justify-between">
        <div><p className="break-words text-sm font-medium">{a.titulo || 'Indisponível'}</p><p className="text-xs text-muted-foreground">{dia(a.inicio)} a {dia(a.fim)} · {a.publico ? 'Pública' : 'Privada'} · {a.tipo === 'bloqueio' ? 'Bloqueio' : 'Apresentação'}</p></div>
        <div className="flex gap-2"><Button size="sm" variant="outline" disabled={ocupado} onClick={() => setD(a)}>Editar</Button><Button size="sm" variant="ghost" disabled={ocupado} onClick={() => remover(a.id)}>Remover</Button></div>
      </div>)}
      <form onSubmit={salvar} className="flex flex-col gap-3">
        <Label htmlFor="agenda-tipo">Tipo</Label>
        <select id="agenda-tipo" className="h-9 rounded-lg border border-border bg-background px-2 text-sm" value={d.tipo} onChange={e => setD({ ...d, tipo: e.target.value as AgendaArtista['tipo'] })}><option value="apresentacao">Apresentação</option><option value="bloqueio">Data bloqueada</option></select>
        <Label htmlFor="agenda-titulo">{d.tipo === 'bloqueio' ? 'Motivo (privado, opcional)' : 'Título da apresentação'}</Label>
        <Input id="agenda-titulo" maxLength={120} required={d.tipo === 'apresentacao'} value={d.titulo} onChange={e => setD({ ...d, titulo: e.target.value })} />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-2"><Label htmlFor="agenda-inicio">Início</Label><Input id="agenda-inicio" type="date" required value={d.inicio} onChange={e => setD({ ...d, inicio: e.target.value, fim: d.fim || e.target.value })} /></div>
          <div className="flex flex-col gap-2"><Label htmlFor="agenda-fim">Fim</Label><Input id="agenda-fim" type="date" required min={d.inicio} value={d.fim} onChange={e => setD({ ...d, fim: e.target.value })} /></div>
        </div>
        <label className="flex items-start gap-2 text-sm"><input type="checkbox" checked={d.publico} onChange={e => setD({ ...d, publico: e.target.checked })} />Mostrar no perfil público</label>
        {d.tipo === 'bloqueio' && <p className="text-xs text-muted-foreground">Se publicado, o bloqueio mostra apenas “Indisponível”, sem o motivo.</p>}
        {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
        <div className="flex gap-2"><Button type="submit" disabled={ocupado}>{ocupado ? 'Salvando…' : d.id ? 'Salvar alterações' : 'Adicionar data'}</Button>{d.id > 0 && <Button type="button" variant="ghost" onClick={() => setD(novaEntrada)}>Cancelar edição</Button>}</div>
      </form>
    </CardContent>
  </Card>
}

export function ArtistaMidiaKit({ id }: { id: number }) {
  const [erro, setErro] = useState('')
  const [ocupado, setOcupado] = useState(false)
  const baixar = async () => {
    setErro(''); setOcupado(true)
    try { await baixarArquivo(`/me/artistas/${id}/midia-kit`, 'midia-kit.pdf') } catch (e) { setErro(mensagem(e)) } finally { setOcupado(false) }
  }
  return <Card><CardHeader><CardTitle>Mídia kit</CardTitle><p className="text-sm text-muted-foreground">PDF para enviar a contratantes, usando apenas o que já aparece no perfil público. Publique o perfil para gerar o arquivo.</p></CardHeader><CardContent><Button onClick={baixar} disabled={ocupado}>{ocupado ? 'Gerando…' : 'Baixar mídia kit em PDF'}</Button>{erro && <p role="alert" className="mt-2 text-sm text-destructive">{erro}</p>}</CardContent></Card>
}
