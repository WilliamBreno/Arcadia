import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export type Criterio = {
  id: number
  nome: string
  peso: number
  nota_min: number
  nota_max: number
  passo: number
  tipo_apresentacao: string | null
}

type Formulario = {
  criterios: Criterio[]
  notas: { criterio_id: number; nota: number; comentario: string; finalizada: boolean }[]
}

// Formulário de notas do jurado para uma ficha (item 3.3). Notas ficam
// como rascunho até "Finalizar"; depois de finalizada a avaliação trava.
export function AvaliarFicha({ slug, fichaId }: { slug: string; fichaId: number }) {
  const queryClient = useQueryClient()
  const chave = ['jurado-avaliacao', slug, fichaId]
  const { data } = useQuery({ queryKey: chave, queryFn: () => api<Formulario>(`/eventos/${slug}/participantes/${fichaId}/avaliacao`) })
  const [valores, setValores] = useState<Record<number, { nota: string; comentario: string }>>({})
  const [erro, setErro] = useState<string | null>(null)
  const [salvando, setSalvando] = useState(false)

  useEffect(() => {
    if (!data) return
    const inicial: Record<number, { nota: string; comentario: string }> = {}
    data.notas.forEach((n) => (inicial[n.criterio_id] = { nota: String(n.nota), comentario: n.comentario }))
    setValores(inicial)
  }, [data])

  if (!data || data.criterios.length === 0) return null
  const finalizada = data.notas.length > 0 && data.notas.every((n) => n.finalizada)

  const enviar = async (finalizar: boolean) => {
    setErro(null)
    setSalvando(true)
    try {
      const notas = Object.entries(valores)
        .filter(([, v]) => v.nota !== '')
        .map(([id, v]) => ({ criterio_id: Number(id), nota: Number(v.nota), comentario: v.comentario }))
      await api(`/eventos/${slug}/participantes/${fichaId}/avaliacao`, { method: 'PUT', body: { notas, finalizar } })
      await queryClient.invalidateQueries({ queryKey: chave })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao salvar notas')
    } finally {
      setSalvando(false)
    }
  }

  return (
    <div className="mt-3 flex flex-col gap-2 border-t border-border pt-3">
      <p className="text-sm font-medium text-foreground">
        Notas {finalizada && <span className="text-xs font-normal text-primary">(finalizada)</span>}
      </p>
      {data.criterios.map((c) => (
        <div key={c.id} className="grid grid-cols-[1fr_5rem_1.5fr] items-center gap-2 text-sm">
          <span className="text-foreground">
            {c.nome} <span className="text-xs text-muted-foreground">(peso {c.peso}, {c.nota_min}–{c.nota_max})</span>
          </span>
          <Input
            type="number"
            min={c.nota_min}
            max={c.nota_max}
            step={c.passo}
            disabled={finalizada}
            value={valores[c.id]?.nota ?? ''}
            onChange={(e) => setValores({ ...valores, [c.id]: { nota: e.target.value, comentario: valores[c.id]?.comentario ?? '' } })}
          />
          <Input
            placeholder="Comentário (privado)"
            disabled={finalizada}
            value={valores[c.id]?.comentario ?? ''}
            onChange={(e) => setValores({ ...valores, [c.id]: { nota: valores[c.id]?.nota ?? '', comentario: e.target.value } })}
          />
        </div>
      ))}
      {erro && <p className="text-xs text-destructive">{erro}</p>}
      {!finalizada && (
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={salvando} onClick={() => enviar(false)}>
            Salvar rascunho
          </Button>
          <Button
            size="sm"
            disabled={salvando}
            onClick={() => confirm('Depois de finalizada, a avaliação não pode mais ser editada. Finalizar?') && enviar(true)}
          >
            Finalizar
          </Button>
        </div>
      )}
    </div>
  )
}
