import { useState } from 'react'

import { Input } from '@/components/ui/input'
import { CATEGORIAS_EVENTO } from '@/lib/organizador'

const OUTRA = '__outra__'
const selectClass = 'h-8 w-full rounded-lg border border-border bg-background px-2.5 text-sm'

// Lista de categorias da plataforma + "Outra" com texto livre para eventos
// muito específicos. O valor guardado é sempre o texto da categoria, então
// eventos antigos (texto livre) continuam válidos e aparecem como "Outra".
export function CategoriaSelect({
  id,
  value,
  onChange,
}: {
  id: string
  value: string
  onChange: (valor: string) => void
}) {
  const conhecida = (CATEGORIAS_EVENTO as readonly string[]).includes(value)
  const [escolheuOutra, setEscolheuOutra] = useState(false)
  // valor fora da lista (evento antigo ou digitado) também conta como "Outra"
  const outra = escolheuOutra || (value !== '' && !conhecida)
  const selecionado = outra ? OUTRA : value

  return (
    <div className="flex flex-col gap-2">
      <select
        id={id}
        className={selectClass}
        value={selecionado}
        onChange={(e) => {
          if (e.target.value === OUTRA) {
            setEscolheuOutra(true)
            onChange(conhecida ? '' : value)
          } else {
            setEscolheuOutra(false)
            onChange(e.target.value)
          }
        }}
      >
        <option value="">Selecione uma categoria</option>
        {CATEGORIAS_EVENTO.map((c) => (
          <option key={c} value={c}>
            {c}
          </option>
        ))}
        <option value={OUTRA}>Outra (digitar)</option>
      </select>
      {outra && (
        <Input
          aria-label="Outra categoria"
          placeholder="Digite a categoria do seu evento"
          maxLength={60}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
      )}
    </div>
  )
}
