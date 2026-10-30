import type { GrupoMapa, SetorMapa } from '@/lib/lugares'

// Desenho do mapa (item 9.5). Fileiras com o palco no topo; mesas em
// cartões com as cadeiras. Sem aoTocar, é só para ver (painel).
export function MapaLugares({
  setores,
  selecionados,
  aoTocar,
}: {
  setores: SetorMapa[]
  selecionados?: Set<number>
  aoTocar?: (setor: SetorMapa, grupo: GrupoMapa, lugarId: number) => void
}) {
  const temFileiras = setores.some((s) => s.formato === 'fileiras')
  return (
    <div className="flex flex-col gap-5">
      <Legenda interativo={!!aoTocar} />
      {temFileiras && (
        <div className="rounded-md bg-muted py-1.5 text-center text-xs font-medium uppercase tracking-wide text-muted-foreground">Palco</div>
      )}
      {setores.map((setor) => (
        <section key={setor.id} aria-label={`Setor ${setor.nome}`} className="flex flex-col gap-2">
          <div className="flex items-baseline justify-between gap-2">
            <h3 className="text-sm font-semibold text-foreground">{setor.nome}</h3>
            <span className="text-xs text-muted-foreground tabular-nums">
              {setor.livres} de {setor.total} livres
            </span>
          </div>
          {setor.formato === 'fileiras' ? (
            <div className="overflow-x-auto pb-1">
              <div className="flex w-max min-w-full flex-col items-center gap-1.5">
                {setor.grupos.map((g) => (
                  <div key={g.grupo} className="flex items-center gap-1.5">
                    <span className="w-6 text-right text-xs font-medium text-muted-foreground">{g.grupo}</span>
                    {g.lugares.map((l) => (
                      <Cadeira
                        key={l.id}
                        rotulo={`Fileira ${g.grupo}, lugar ${l.numero}`}
                        numero={l.numero}
                        ocupado={l.ocupado}
                        escolhido={!!selecionados?.has(l.id)}
                        aoTocar={aoTocar && (() => aoTocar(setor, g, l.id))}
                      />
                    ))}
                    <span className="w-6 text-xs font-medium text-muted-foreground">{g.grupo}</span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              {setor.grupos.map((g) => (
                <div key={g.grupo} className="flex flex-col items-center gap-2 rounded-xl border border-border p-3">
                  <span className="text-sm font-medium text-foreground">Mesa {g.grupo}</span>
                  <div className="flex flex-wrap justify-center gap-1.5">
                    {g.lugares.map((l) => (
                      <Cadeira
                        key={l.id}
                        rotulo={`Mesa ${g.grupo}, cadeira ${l.numero}`}
                        numero={l.numero}
                        ocupado={l.ocupado}
                        escolhido={!!selecionados?.has(l.id)}
                        aoTocar={aoTocar && (() => aoTocar(setor, g, l.id))}
                      />
                    ))}
                  </div>
                  <span className="text-xs text-muted-foreground">{g.mesa_inteira ? `Mesa inteira (${g.lugares.length} lugares)` : 'Cadeira avulsa'}</span>
                </div>
              ))}
            </div>
          )}
        </section>
      ))}
    </div>
  )
}

function Cadeira({
  rotulo,
  numero,
  ocupado,
  escolhido,
  aoTocar,
}: {
  rotulo: string
  numero: number
  ocupado: boolean
  escolhido: boolean
  aoTocar?: () => void
}) {
  const estado = ocupado ? 'ocupado' : escolhido ? 'escolhido' : 'livre'
  const cor =
    estado === 'ocupado'
      ? 'border-transparent bg-muted text-muted-foreground/50'
      : estado === 'escolhido'
        ? 'border-primary bg-primary text-primary-foreground'
        : 'border-border bg-background text-foreground hover:border-primary'
  const base = `flex size-8 shrink-0 items-center justify-center rounded-md border text-xs tabular-nums transition-colors ${cor}`
  if (!aoTocar) {
    return (
      <span className={base} title={`${rotulo}: ${estado}`}>
        {numero}
      </span>
    )
  }
  return (
    <button
      type="button"
      className={`${base} focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:cursor-not-allowed`}
      disabled={ocupado}
      aria-pressed={escolhido}
      aria-label={`${rotulo}, ${estado}`}
      onClick={aoTocar}
    >
      {numero}
    </button>
  )
}

function Legenda({ interativo }: { interativo: boolean }) {
  return (
    <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
      <span className="flex items-center gap-1.5">
        <span className="size-4 rounded border border-border bg-background" aria-hidden="true" /> Livre
      </span>
      {interativo && (
        <span className="flex items-center gap-1.5">
          <span className="size-4 rounded bg-primary" aria-hidden="true" /> Sua escolha
        </span>
      )}
      <span className="flex items-center gap-1.5">
        <span className="size-4 rounded bg-muted" aria-hidden="true" /> Ocupado
      </span>
    </div>
  )
}
