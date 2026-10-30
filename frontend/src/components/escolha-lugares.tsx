import { X } from 'lucide-react'

import { MapaLugares } from '@/components/mapa-lugares'
import { Button } from '@/components/ui/button'
import { formatarCentavos } from '@/lib/evento'
import { rotuloLugar, type Escolhas, type GrupoMapa, type SetorMapa } from '@/lib/lugares'

type TipoVenda = { id: number; nome: string; preco_centavos: number; ativo: boolean; esgotado?: boolean }

// Escolha das cadeiras na página do evento (item 9.5). Cada cadeira tocada
// entra com o primeiro tipo à venda do setor; se o setor aceita mais de um
// (inteira e meia), a pessoa troca na lista abaixo do mapa.
export function EscolhaLugares({
  setores,
  tipos,
  escolhas,
  aoMudar,
}: {
  setores: SetorMapa[]
  tipos: TipoVenda[]
  escolhas: Escolhas
  aoMudar: (e: Escolhas) => void
}) {
  const aVenda = (setor: SetorMapa) => tipos.filter((t) => setor.tipo_ingresso_ids.includes(t.id) && t.ativo && !t.esgotado)

  const tocar = (setor: SetorMapa, grupo: GrupoMapa, lugarId: number) => {
    const opcoes = aVenda(setor)
    if (opcoes.length === 0) return
    // Mesa inteira: as cadeiras entram e saem juntas.
    const ids = grupo.mesa_inteira ? grupo.lugares.map((l) => l.id) : [lugarId]
    const proximo = { ...escolhas }
    if (escolhas[lugarId]) {
      for (const id of ids) delete proximo[id]
    } else {
      if (grupo.mesa_inteira && grupo.lugares.some((l) => l.ocupado)) return
      for (const id of ids) proximo[id] = opcoes[0].id
    }
    aoMudar(proximo)
  }

  const escolhidas = setores.flatMap((setor) =>
    setor.grupos.flatMap((g) =>
      g.lugares.filter((l) => escolhas[l.id]).map((l) => ({ setor, grupo: g, lugar: l, tipoId: escolhas[l.id] })),
    ),
  )

  return (
    <section aria-labelledby="titulo-mapa" className="mt-6 flex flex-col gap-4 rounded-xl border border-border p-4">
      <div>
        <h2 id="titulo-mapa" className="text-lg font-semibold text-foreground">
          Escolha seus lugares
        </h2>
        <p className="text-sm text-muted-foreground">Toque nas cadeiras livres. Elas ficam guardadas para você enquanto paga.</p>
      </div>
      <MapaLugares setores={setores} selecionados={new Set(Object.keys(escolhas).map(Number))} aoTocar={tocar} />
      {escolhidas.length > 0 && (
        <ul className="flex flex-col gap-2" aria-label="Lugares escolhidos">
          {escolhidas.map(({ setor, grupo, lugar, tipoId }) => {
            const opcoes = aVenda(setor)
            return (
              <li key={lugar.id} className="flex flex-wrap items-center gap-2 rounded-lg bg-muted px-3 py-2 text-sm">
                <span className="min-w-0 flex-1 font-medium text-foreground">{rotuloLugar(setor, grupo.grupo, lugar.numero)}</span>
                {opcoes.length > 1 ? (
                  <select
                    aria-label={`Ingresso para ${rotuloLugar(setor, grupo.grupo, lugar.numero)}`}
                    className="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground"
                    value={tipoId}
                    onChange={(e) => aoMudar({ ...escolhas, [lugar.id]: Number(e.target.value) })}
                  >
                    {opcoes.map((t) => (
                      <option key={t.id} value={t.id}>
                        {t.nome} · {formatarCentavos(t.preco_centavos)}
                      </option>
                    ))}
                  </select>
                ) : (
                  <span className="text-muted-foreground">{opcoes.find((t) => t.id === tipoId)?.nome}</span>
                )}
                {!grupo.mesa_inteira && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={`Tirar ${rotuloLugar(setor, grupo.grupo, lugar.numero)}`}
                    onClick={() => {
                      const proximo = { ...escolhas }
                      delete proximo[lugar.id]
                      aoMudar(proximo)
                    }}
                  >
                    <X className="size-4" aria-hidden="true" />
                  </Button>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
