import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { formatarCentavos } from '@/lib/evento'
import type { Extras } from '@/lib/extras'
import { calcularItem, type RegraTaxas } from '@/lib/taxas'

// Item 10.3: pacotes e adicionais na página do evento.
export function ExtrasCompra({
  extras,
  regra,
  qtdAdicionais,
  qtdPacotes,
  aoMudarAdicional,
  aoMudarPacote,
  temIngresso,
  oculto,
}: {
  extras: Extras
  regra: RegraTaxas
  qtdAdicionais: Record<number, number>
  qtdPacotes: Record<number, number>
  aoMudarAdicional: (id: number, qtd: number) => void
  aoMudarPacote: (id: number, qtd: number) => void
  temIngresso: boolean
  oculto: boolean
}) {
  if (oculto || (extras.adicionais.length === 0 && extras.pacotes.length === 0)) return null
  return (
    <>
      {extras.pacotes.length > 0 && (
        <section aria-labelledby="titulo-pacotes" className="mt-8">
          <h2 id="titulo-pacotes" className="mb-3 text-xl font-semibold text-foreground">
            Pacotes
          </h2>
          <div className="flex flex-col gap-3">
            {extras.pacotes.map((p) => {
              const restam = p.quantidade - p.vendidos
              const economia = p.tabela_centavos - p.preco_centavos
              return (
                <Card key={p.id}>
                  <CardContent className="flex items-center justify-between gap-4 py-4">
                    <div className="min-w-0">
                      <p className="font-medium text-foreground">{p.nome}</p>
                      <p className="text-sm text-muted-foreground">{p.itens.map((it) => `${it.quantidade}× ${it.nome}`).join(' + ')}</p>
                      {p.descricao && <p className="text-sm text-muted-foreground">{p.descricao}</p>}
                      <p className="mt-1 text-lg font-semibold tabular-nums text-foreground">{formatarCentavos(p.preco_centavos)}</p>
                      <p className="text-xs text-muted-foreground">
                        {economia > 0 && `Economia de ${formatarCentavos(economia)} · `}+ taxa por item
                        {restam <= 0 ? ' · esgotado' : restam <= 5 ? ` · restam ${restam}` : ''}
                      </p>
                    </div>
                    {restam > 0 && (
                      <Contador
                        rotulo={p.nome}
                        valor={qtdPacotes[p.id] ?? 0}
                        max={Math.min(p.max_por_pedido, restam)}
                        aoMudar={(q) => aoMudarPacote(p.id, q)}
                      />
                    )}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </section>
      )}

      {extras.adicionais.length > 0 && (
        <section aria-labelledby="titulo-adicionais" className="mt-8">
          <h2 id="titulo-adicionais" className="text-xl font-semibold text-foreground">
            Adicionais
          </h2>
          <p className="mb-3 text-sm text-muted-foreground">Junto com o ingresso. Você retira no evento mostrando o código em Meus ingressos.</p>
          <div className="flex flex-col gap-3">
            {extras.adicionais.map((a) => (
              <Card key={a.id}>
                <CardContent className="flex items-center justify-between gap-4 py-4">
                  <div className="min-w-0">
                    <p className="font-medium text-foreground">{a.nome}</p>
                    {a.descricao && <p className="text-sm text-muted-foreground">{a.descricao}</p>}
                    <p className="mt-1 font-semibold tabular-nums text-foreground">{formatarCentavos(calcularItem(regra, a.preco_centavos, false).total)}</p>
                    {a.disponivel <= 0 && <p className="text-xs text-muted-foreground">Esgotado</p>}
                  </div>
                  {a.disponivel > 0 && (
                    <Contador
                      rotulo={a.nome}
                      valor={qtdAdicionais[a.id] ?? 0}
                      max={Math.min(a.max_por_pedido, a.disponivel)}
                      aoMudar={(q) => aoMudarAdicional(a.id, q)}
                    />
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
          {!temIngresso && Object.values(qtdAdicionais).some((q) => q > 0) && (
            <p role="status" className="mt-2 text-sm text-destructive">
              Escolha também um ingresso: adicionais vão junto com ele.
            </p>
          )}
        </section>
      )}
    </>
  )
}

function Contador({ rotulo, valor, max, aoMudar }: { rotulo: string; valor: number; max: number; aoMudar: (q: number) => void }) {
  return (
    <div className="flex shrink-0 items-center gap-2">
      <Button variant="outline" size="icon-sm" type="button" aria-label={`Menos ${rotulo}`} disabled={valor <= 0} onClick={() => aoMudar(Math.max(0, valor - 1))}>
        −
      </Button>
      <span className="w-6 text-center text-sm tabular-nums text-foreground" aria-live="polite">
        {valor}
      </span>
      <Button variant="outline" size="icon-sm" type="button" aria-label={`Mais ${rotulo}`} disabled={valor >= max} onClick={() => aoMudar(Math.min(max, valor + 1))}>
        +
      </Button>
    </div>
  )
}
