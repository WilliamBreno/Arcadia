import { valeParaTipo, type Pergunta, type RespostasCheckout } from '@/lib/perguntas'
import { Input } from '@/components/ui/input'

type Props = {
  perguntas: Pergunta[]
  ingressos: { id: number; nome: string }[]
  quantidades: Record<number, number>
  respostas: RespostasCheckout
  aoMudar: (r: RespostasCheckout) => void
  organizador?: string
}

// Perguntas do organizador, respondidas uma vez por ingresso (item 7.6).
// Cada pergunta mostra para que o organizador pede a informação.
export function PerguntasCheckout({ perguntas, ingressos, quantidades, respostas, aoMudar, organizador }: Props) {
  const blocos = ingressos
    .flatMap((t) =>
      Array.from({ length: quantidades[t.id] ?? 0 }, (_, i) => ({
        tipo: t,
        i,
        perguntas: perguntas.filter((p) => valeParaTipo(p, t.id)),
      })),
    )
    .filter((b) => b.perguntas.length > 0)
  if (blocos.length === 0) return null

  const mudar = (chave: string, pid: number, valor: string) =>
    aoMudar({ ...respostas, [chave]: { ...respostas[chave], [pid]: valor } })
  const varios = blocos.length > 1

  return (
    <section aria-labelledby="titulo-perguntas" className="mt-4 flex flex-col gap-4 rounded-xl border border-border p-4">
      <div>
        <h3 id="titulo-perguntas" className="font-semibold text-foreground">
          Perguntas {organizador ? `de ${organizador}` : 'do organizador'}
        </h3>
        <p className="text-xs text-muted-foreground">
          As respostas vão só para o organizador e são apagadas 90 dias depois do evento.
        </p>
      </div>
      {blocos.map(({ tipo, i, perguntas: ps }) => {
        const chave = `${tipo.id}:${i}`
        return (
          <fieldset key={chave} className="flex flex-col gap-3">
            {varios && (
              <legend className="mb-1 text-sm font-medium text-foreground">
                Ingresso {i + 1} · {tipo.nome}
              </legend>
            )}
            {ps.map((p) => {
              const id = `pq-${chave}-${p.id}`
              const valor = respostas[chave]?.[p.id] ?? ''
              return (
                <div key={p.id} className="flex flex-col gap-1">
                  <label htmlFor={id} className="text-sm text-foreground">
                    {p.texto}
                    {p.obrigatoria ? (
                      <span className="text-destructive"> *</span>
                    ) : (
                      <span className="text-muted-foreground"> (opcional)</span>
                    )}
                  </label>
                  {p.tipo === 'opcao' ? (
                    <select
                      id={id}
                      value={valor}
                      required={p.obrigatoria}
                      onChange={(e) => mudar(chave, p.id, e.target.value)}
                      className="h-9 rounded-lg border border-border bg-background px-2 text-sm text-foreground"
                    >
                      <option value="">Escolha…</option>
                      {p.opcoes.map((o) => (
                        <option key={o} value={o}>
                          {o}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <Input
                      id={id}
                      value={valor}
                      maxLength={300}
                      required={p.obrigatoria}
                      onChange={(e) => mudar(chave, p.id, e.target.value)}
                    />
                  )}
                  <p className="text-xs text-muted-foreground">Para quê: {p.finalidade}</p>
                </div>
              )
            })}
          </fieldset>
        )
      })}
    </section>
  )
}
