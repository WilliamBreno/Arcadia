import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api, ApiError } from '@/lib/api'
import type { Municipio } from '@/lib/contratacao'
import { Label } from '@/components/ui/label'

const estados = 'AC AL AP AM BA CE DF ES GO MA MT MS MG PA PB PR PE PI RJ RN RS RO RR SC SP SE TO'.split(' ')
export function MunicipioArtista({ id, label, valor, onChange, required = false }: { id: string; label: string; valor?: Municipio; onChange: (m: Municipio | undefined) => void; required?: boolean }) {
  const [uf, setUF] = useState(valor?.uf ?? '')
  const { data, isLoading, error, refetch } = useQuery({ queryKey: ['municipios-artistas', uf], queryFn: () => api<Municipio[]>(`/artistas-municipios?uf=${uf}`, { semAuth: true }), enabled: !!uf, staleTime: 86400000, retry: false })
  return <fieldset className="flex min-w-0 flex-col gap-2">
    <legend className="mb-2 text-sm font-medium">{label}</legend>
    <div className="flex min-w-0 gap-2">
      <div className="flex w-20 shrink-0 flex-col gap-1"><Label className="sr-only" htmlFor={`${id}-uf`}>UF de {label}</Label>
        <select id={`${id}-uf`} className="h-9 rounded-lg border border-border bg-background px-2 text-sm" value={uf} required={required} onChange={e => { setUF(e.target.value); onChange(undefined) }}>
          <option value="">UF</option>{estados.map(uf => <option key={uf}>{uf}</option>)}
        </select>
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-1"><Label className="sr-only" htmlFor={`${id}-cidade`}>{label}</Label>
        <select id={`${id}-cidade`} className="h-9 min-w-0 rounded-lg border border-border bg-background px-2 text-sm" disabled={!uf || isLoading || !!error} required={required} value={valor?.id ?? ''} onChange={e => onChange(data?.find(m => m.id === Number(e.target.value)))}>
          <option value="">{isLoading ? 'Carregando…' : 'Selecione a cidade'}</option>
          {data?.map(m => <option key={m.id} value={m.id}>{m.nome}</option>)}
        </select>
      </div>
    </div>
    {error && <p role="alert" className="text-xs text-destructive">{error instanceof ApiError ? error.message : 'Não foi possível buscar as cidades.'} <button type="button" className="underline" onClick={() => refetch()}>Tentar novamente</button></p>}
  </fieldset>
}
