import { Star } from 'lucide-react'

// Estrelas só de leitura. A nota vai no aria-label; as estrelas são enfeite.
export function Estrelas({ nota, tamanho = 16 }: { nota: number; tamanho?: number }) {
  return (
    <span className="inline-flex items-center gap-0.5" role="img" aria-label={`${nota} de 5 estrelas`}>
      {[1, 2, 3, 4, 5].map((i) => (
        <Star
          key={i}
          aria-hidden="true"
          style={{ width: tamanho, height: tamanho }}
          className={i <= Math.round(nota) ? 'fill-foreground text-foreground' : 'text-muted-foreground opacity-40'}
        />
      ))}
    </span>
  )
}

const ROTULOS = ['', 'Ruim', 'Fraco', 'Ok', 'Bom', 'Excelente']

// Escolha da nota: um grupo de rádio (funciona com teclado e leitor de tela).
export function EscolherNota({ valor, aoMudar }: { valor: number; aoMudar: (n: number) => void }) {
  return (
    <fieldset>
      <legend className="text-sm font-medium text-foreground">Sua nota</legend>
      <div className="mt-1 flex items-center gap-1">
        {[1, 2, 3, 4, 5].map((i) => (
          <label key={i} className="cursor-pointer rounded-md p-1 focus-within:ring-2 focus-within:ring-ring">
            <input
              type="radio"
              name="nota"
              value={i}
              checked={valor === i}
              onChange={() => aoMudar(i)}
              className="sr-only"
              aria-label={`${i} de 5: ${ROTULOS[i]}`}
            />
            <Star
              aria-hidden="true"
              style={{ width: 32, height: 32 }}
              className={`transition-transform active:scale-90 ${i <= valor ? 'fill-foreground text-foreground' : 'text-muted-foreground'}`}
            />
          </label>
        ))}
        <span className="ml-2 text-sm text-muted-foreground" aria-live="polite">
          {ROTULOS[valor]}
        </span>
      </div>
    </fieldset>
  )
}
