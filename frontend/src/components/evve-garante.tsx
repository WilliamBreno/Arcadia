import { ShieldCheck } from 'lucide-react'
import { Link } from 'react-router-dom'

type Props = {
  garantiaHabilitada: boolean
  organizadorVerificado: boolean
  eventoComecou: boolean
}

// Selo "Evve Garante" (item 6.3): só lista proteções que o sistema
// realmente cumpre neste evento — nada de promessa genérica.
export function EvveGarante({ garantiaHabilitada, organizadorVerificado, eventoComecou }: Props) {
  const itens = [
    'Evento cancelado: você recebe tudo de volta automaticamente, inclusive a taxa.',
    garantiaHabilitada && !eventoComecou ? 'Garantia de vaga: cancele até o início do evento e receba tudo de volta.' : null,
    'Arrependimento: até 7 dias depois da compra, se faltarem mais de 48 horas para o evento.',
    organizadorVerificado ? 'Organizador verificado: documento e dados de recebimento conferidos.' : null,
    'Ingresso no seu nome, com QR Code único, entregue na hora.',
  ].filter(Boolean) as string[]

  return (
    <section aria-labelledby="titulo-garante" className="mt-6 rounded-xl bg-muted p-4 sm:p-5">
      <h2 id="titulo-garante" className="flex items-center gap-2 font-semibold text-foreground">
        <ShieldCheck className="size-5" aria-hidden="true" />
        Evve Garante
      </h2>
      <ul className="mt-2 flex flex-col gap-1.5 text-sm text-foreground">
        {itens.map((t) => (
          <li key={t} className="text-pretty">
            {t}
          </li>
        ))}
      </ul>
      <Link to="/quanto-custa#garante" className="mt-3 inline-block text-sm font-medium text-foreground underline underline-offset-4">
        Como funciona
      </Link>
    </section>
  )
}
