import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, ApiError } from '@/lib/api'
import { obterOferta, lerCentavos, textoCentavos, type OfertaArtista } from '@/lib/contratacao'
import { MunicipioArtista } from '@/components/municipio-artista'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useAuth } from '@/hooks/use-auth'

export function ArtistaOferta({ id }: { id: number }) {
  const { usuario } = useAuth()
  const { data, error } = useQuery({ queryKey: ['oferta-artista', id, usuario?.id], queryFn: () => obterOferta(id) })
  return <Card><CardHeader><CardTitle>Contratação</CardTitle><p className="text-sm text-muted-foreground">Defina o que oferece e receba pedidos de orçamento. A Evve aproxima as partes; pagamento e condições são combinados entre vocês.</p></CardHeader><CardContent>
    {error && <p role="alert">Não foi possível carregar a oferta.</p>}
    {data && <FormularioOferta key={`${id}-${usuario?.id}`} id={id} inicial={data} />}
  </CardContent></Card>
}
function FormularioOferta({ id, inicial }: { id: number; inicial: OfertaArtista }) {
  const [d, setD] = useState(inicial)
  const [valor, setValor] = useState(textoCentavos(inicial.valor_centavos))
  const [erro, setErro] = useState(''), [ok, setOk] = useState(''), [ocupado, setOcupado] = useState(false)
  const salvar = async (e: React.FormEvent) => {
    e.preventDefault(); setErro(''); setOk('')
    const centavos = d.preco_tipo === 'consulta' ? null : lerCentavos(valor)
    if (d.preco_tipo !== 'consulta' && centavos === null) { setErro('Informe um valor válido em reais, com até duas casas decimais.'); return }
    setOcupado(true)
    try {
      await api(`/me/artistas/${id}/oferta`, { method: 'PUT', body: { ...d, valor_centavos: centavos, origem_id: d.origem?.id ?? null } })
      setOk('Oferta salva. Ela aparece no perfil quando estiver habilitada e o perfil for público.')
    } catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.') } finally { setOcupado(false) }
  }
  return <form onSubmit={salvar} className="flex flex-col gap-4">
    <fieldset disabled={ocupado} className="flex min-w-0 flex-col gap-4">
      <label className="flex gap-2 text-sm"><input type="checkbox" checked={d.habilitada} onChange={e => setD({ ...d, habilitada: e.target.checked })} />Receber pedidos de orçamento</label>
      <div className="flex flex-col gap-2"><Label htmlFor="oferta-tipo">Como mostrar o valor</Label>
        <select id="oferta-tipo" className="h-9 rounded-lg border border-border bg-background px-2 text-sm" value={d.preco_tipo} onChange={e => setD({ ...d, preco_tipo: e.target.value as OfertaArtista['preco_tipo'] })}>
          <option value="consulta">Sob consulta</option><option value="fixo">Valor fixo</option><option value="a_partir">A partir de</option>
        </select>
      </div>
      {d.preco_tipo !== 'consulta' && <div className="flex flex-col gap-2"><Label htmlFor="oferta-valor">Valor em reais</Label><Input id="oferta-valor" inputMode="decimal" required value={valor} onChange={e => setValor(e.target.value)} placeholder="1500,00" /></div>}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-2"><Label htmlFor="oferta-pessoas">Quantas pessoas viajam</Label><Input id="oferta-pessoas" type="number" min={1} max={1000} required value={d.integrantes} onChange={e => setD({ ...d, integrantes: Number(e.target.value) })} /></div>
        <div className="flex flex-col gap-2"><Label htmlFor="oferta-duracao">Duração da apresentação (minutos)</Label><Input id="oferta-duracao" type="number" min={1} max={1440} required value={d.duracao_minutos} onChange={e => setD({ ...d, duracao_minutos: Number(e.target.value) })} /></div>
      </div>
      <MunicipioArtista id="origem" label="Cidade de saída" valor={d.origem} onChange={origem => setD({ ...d, origem, origem_id: origem?.id ?? null })} />
      <fieldset className="flex flex-col gap-2 text-sm"><legend className="mb-2 font-medium">O que está incluso no valor</legend>
        {([{ k: 'passagem_inclusa', nome: 'Passagem' }, { k: 'hospedagem_inclusa', nome: 'Hospedagem' }, { k: 'alimentacao_inclusa', nome: 'Alimentação' }] as const).map(({ k, nome }) => <label key={k} className="flex gap-2"><input type="checkbox" checked={d[k]} onChange={e => setD({ ...d, [k]: e.target.checked })} />{nome}</label>)}
      </fieldset>
      <div className="flex flex-col gap-2"><Label htmlFor="oferta-obs">Detalhes da oferta</Label><Textarea id="oferta-obs" maxLength={1500} value={d.observacoes} onChange={e => setD({ ...d, observacoes: e.target.value })} /></div>
    </fieldset>
    {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}{ok && <p role="status" className="text-sm">{ok}</p>}
    <Button type="submit" disabled={ocupado} className="self-start">{ocupado ? 'Salvando…' : 'Salvar oferta'}</Button>
  </form>
}
