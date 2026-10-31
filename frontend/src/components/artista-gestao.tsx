import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, baixarArquivo } from '@/lib/api'
import type { GestaoArtista } from '@/lib/artistas'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const mensagem = (e: unknown) => e instanceof ApiError ? e.message : 'Não foi possível concluir. Tente novamente.'

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

export function ArtistaMidiaKit({ id }: { id: number }) {
  const [erro, setErro] = useState('')
  const [ocupado, setOcupado] = useState(false)
  const baixar = async () => {
    setErro(''); setOcupado(true)
    try { await baixarArquivo(`/me/artistas/${id}/midia-kit`, 'midia-kit.pdf') } catch (e) { setErro(mensagem(e)) } finally { setOcupado(false) }
  }
  return <Card><CardHeader><CardTitle>Mídia kit</CardTitle><p className="text-sm text-muted-foreground">PDF para enviar a contratantes, usando apenas o que já aparece no perfil público. Publique o perfil para gerar o arquivo.</p></CardHeader><CardContent><Button onClick={baixar} disabled={ocupado}>{ocupado ? 'Gerando…' : 'Baixar mídia kit em PDF'}</Button>{erro && <p role="alert" className="mt-2 text-sm text-destructive">{erro}</p>}</CardContent></Card>
}
