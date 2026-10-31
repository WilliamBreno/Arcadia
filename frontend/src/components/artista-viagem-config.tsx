import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, ApiError } from '@/lib/api'
import { lerCentavos, textoCentavos, type ViagemConfig } from '@/lib/contratacao'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/use-auth'

export function ArtistaViagemConfig() {
  const { usuario } = useAuth()
  const { data, error } = useQuery({ queryKey: ['viagem-config-artista', usuario?.id], queryFn: () => api<ViagemConfig>('/admin/artistas/viagem-config'), retry: false })
  return <Card className="mb-8"><CardHeader><CardTitle>Estimativa de deslocamento</CardTitle><p className="text-sm text-muted-foreground">Começa desativada. Configure os parâmetros antes de habilitar. São estimativas por distância e pessoa, sem consulta ao preço real das passagens.</p></CardHeader><CardContent>
    {error && <p role="alert">Não foi possível carregar a configuração.</p>}{data && <Formulario key={usuario?.id} inicial={data} />}
  </CardContent></Card>
}
function Formulario({ inicial }: { inicial: ViagemConfig }) {
  const [d, setD] = useState(inicial), [onibus, setOnibus] = useState(textoCentavos(inicial.onibus_centavos_km)), [aviao, setAviao] = useState(textoCentavos(inicial.aviao_centavos_km))
  const [minimo, setMinimo] = useState(inicial.distancia_min_km?.toString() ?? ''), [limiar, setLimiar] = useState(inicial.aviao_desde_km?.toString() ?? '')
  const [erro, setErro] = useState(''), [ok, setOk] = useState(''), [ocupado, setOcupado] = useState(false)
  const salvar = async (e: React.FormEvent) => {
    e.preventDefault(); setErro(''); setOk('')
    const onibusC = onibus ? lerCentavos(onibus) : null, aviaoC = aviao ? lerCentavos(aviao) : null
    if ((onibus && onibusC === null) || (aviao && aviaoC === null)) { setErro('Informe valores em reais com até duas casas decimais.'); return }
    setOcupado(true)
    try {
      await api('/admin/artistas/viagem-config', { method: 'PUT', body: { ...d, onibus_centavos_km: onibusC, aviao_centavos_km: aviaoC, distancia_min_km: minimo === '' ? null : Number(minimo), aviao_desde_km: limiar === '' ? null : Number(limiar) } })
      setOk('Configuração salva.')
    } catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.') } finally { setOcupado(false) }
  }
  return <form onSubmit={salvar} className="flex flex-col gap-4">
    <fieldset disabled={ocupado} className="flex min-w-0 flex-col gap-4">
      <label className="flex gap-2 text-sm"><input type="checkbox" checked={d.habilitada} onChange={e => setD({ ...d, habilitada: e.target.checked })} />Habilitar estimativas</label>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-2"><Label htmlFor="viagem-onibus">Ônibus: R$ por km e pessoa</Label><Input id="viagem-onibus" inputMode="decimal" required={d.habilitada} value={onibus} onChange={e => setOnibus(e.target.value)} /></div>
        <div className="flex flex-col gap-2"><Label htmlFor="viagem-aviao">Avião: R$ por km e pessoa</Label><Input id="viagem-aviao" inputMode="decimal" required={d.habilitada} value={aviao} onChange={e => setAviao(e.target.value)} /></div>
        <div className="flex flex-col gap-2"><Label htmlFor="viagem-minimo">Calcular a partir de (km)</Label><Input id="viagem-minimo" type="number" min={0} max={40000} required={d.habilitada} value={minimo} onChange={e => setMinimo(e.target.value)} /></div>
        <div className="flex flex-col gap-2"><Label htmlFor="viagem-limiar">Comparar avião a partir de (km)</Label><Input id="viagem-limiar" type="number" min={0} max={40000} required={d.habilitada} value={limiar} onChange={e => setLimiar(e.target.value)} /></div>
      </div>
      <label className="flex gap-2 text-sm"><input type="checkbox" checked={d.ida_volta} onChange={e => setD({ ...d, ida_volta: e.target.checked })} />Calcular ida e volta (desmarcado: só ida)</label>
      <p className="text-xs text-muted-foreground">Abaixo do limiar do avião, considera ônibus; acima, compara os dois e mostra o menor valor. A distância é aproximada, em linha reta entre centros das malhas municipais do IBGE.</p>
    </fieldset>
    {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}{ok && <p role="status" className="text-sm">{ok}</p>}
    <Button type="submit" disabled={ocupado} className="self-start">{ocupado ? 'Salvando…' : 'Salvar estimativa de viagem'}</Button>
  </form>
}
