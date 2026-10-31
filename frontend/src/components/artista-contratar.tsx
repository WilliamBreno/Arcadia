import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import type { EstimativaViagem, Municipio, OfertaArtista } from '@/lib/contratacao'
import { MunicipioArtista } from '@/components/municipio-artista'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

export function ArtistaContratar({ slug, oferta: o }: { slug: string; oferta: OfertaArtista }) {
  const { usuario } = useAuth(), location = useLocation(), navigate = useNavigate()
  const [inicio, setInicio] = useState(''), [fim, setFim] = useState(''), [destino, setDestino] = useState<Municipio>()
  const [tipo, setTipo] = useState(''), [duracao, setDuracao] = useState(o.duracao_minutos), [obs, setObs] = useState('')
  const [erro, setErro] = useState(''), [ocupado, setOcupado] = useState(false)
  const { data: viagem, error: erroViagem } = useQuery({
    queryKey: ['viagem-artista', slug, destino?.id], enabled: !!destino,
    queryFn: () => api<EstimativaViagem>(`/artistas/${slug}/viagem?destino_id=${destino!.id}`, { semAuth: true }), retry: false,
  })
  const pedir = async (e: React.FormEvent) => {
    e.preventDefault(); setErro('')
    if (!destino) { setErro('Escolha a cidade do evento.'); return }
    setOcupado(true)
    try {
      const r = await api<{ id: number }>(`/artistas/${slug}/orcamentos`, { method: 'POST', body: { inicio, fim, destino_id: destino.id, tipo_evento: tipo, duracao_minutos: duracao, observacoes: obs } })
      navigate(`/orcamento-artista/${r.id}`)
    } catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível enviar o pedido.') } finally { setOcupado(false) }
  }
  return <section id="contratar" className="mx-4 mt-8 rounded-2xl border border-border p-4 sm:p-6">
    <h2 className="text-xl font-semibold">Contrate este artista</h2>
    <p className="mt-2 text-2xl font-semibold">{o.preco_tipo === 'consulta' || o.valor_centavos === null ? 'Sob consulta' : `${o.preco_tipo === 'a_partir' ? 'A partir de ' : ''}${formatarCentavos(o.valor_centavos)}`}</p>
    <p className="mt-2 text-sm text-muted-foreground">{o.integrantes} {o.integrantes === 1 ? 'pessoa' : 'pessoas'} · {o.duracao_minutos} minutos de apresentação</p>
    <p className="mt-2 text-sm">{o.passagem_inclusa ? 'Passagem inclusa' : 'Passagem a combinar'} · {o.hospedagem_inclusa ? 'Hospedagem inclusa' : 'Hospedagem a combinar'} · {o.alimentacao_inclusa ? 'Alimentação inclusa' : 'Alimentação a combinar'}</p>
    {o.origem && <p className="mt-2 text-sm text-muted-foreground">Saída de {o.origem.nome}/{o.origem.uf}</p>}
    {o.observacoes && <p className="mt-3 whitespace-pre-wrap text-sm">{o.observacoes}</p>}
    <p className="my-4 text-sm text-muted-foreground">A Evve aproxima as partes. O pedido não cobra nem reserva a data; o aceite da proposta confirma a reserva na agenda. Pagamento e demais condições são combinados diretamente.</p>
    {!usuario ? <Link to="/login" state={{ de: location.pathname + '#contratar' }} className={buttonVariants()}>Entrar para pedir orçamento</Link> : <form onSubmit={pedir} className="flex flex-col gap-4">
      <fieldset disabled={ocupado} className="flex min-w-0 flex-col gap-4">
        <legend className="sr-only">Seu pedido de orçamento</legend>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex min-w-0 flex-col gap-2"><Label htmlFor="pedido-inicio">Data inicial</Label><Input id="pedido-inicio" type="date" required value={inicio} onChange={e => { setInicio(e.target.value); if (!fim || fim < e.target.value) setFim(e.target.value) }} /></div>
          <div className="flex min-w-0 flex-col gap-2"><Label htmlFor="pedido-fim">Data final</Label><Input id="pedido-fim" type="date" required min={inicio} value={fim} onChange={e => setFim(e.target.value)} /></div>
        </div>
        <MunicipioArtista id="pedido-destino" label="Cidade do evento" valor={destino} onChange={setDestino} required />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-2"><Label htmlFor="pedido-tipo">Tipo de evento</Label><Input id="pedido-tipo" required minLength={2} maxLength={100} placeholder="Festival, festa, convenção…" value={tipo} onChange={e => setTipo(e.target.value)} /></div>
          <div className="flex flex-col gap-2"><Label htmlFor="pedido-duracao">Duração desejada (minutos)</Label><Input id="pedido-duracao" required type="number" min={1} max={1440} value={duracao} onChange={e => setDuracao(Number(e.target.value))} /></div>
        </div>
        <div className="flex flex-col gap-2"><Label htmlFor="pedido-obs">Observações (opcional)</Label><Textarea id="pedido-obs" maxLength={3000} value={obs} onChange={e => setObs(e.target.value)} /></div>
      </fieldset>
      {viagem && <div className="rounded-lg bg-muted p-3 text-sm">
        <p>{viagem.aviso}</p>
        {viagem.disponivel && <p className="mt-2 font-medium">{viagem.distancia_km} km · {viagem.transporte === 'aviao' ? 'Avião' : 'Ônibus'} · {viagem.ida_volta ? 'Ida e volta' : 'Só ida'} · {formatarCentavos(viagem.viagem_centavos ?? 0)}</p>}
        {viagem.total_centavos !== null && <p className="mt-2 font-medium">{viagem.preco_tipo === 'a_partir' ? 'Estimativa a partir de' : 'Total estimado'}: {formatarCentavos(viagem.total_centavos)}</p>}
      </div>}
      {erroViagem && <p className="text-sm text-muted-foreground">Estimativa indisponível agora. Você pode enviar o pedido e combinar o deslocamento na conversa.</p>}
      {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
      <Button type="submit" disabled={ocupado} className="self-start">{ocupado ? 'Enviando…' : 'Pedir orçamento'}</Button>
    </form>}
  </section>
}
