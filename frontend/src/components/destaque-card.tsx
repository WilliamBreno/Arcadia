import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { PagarCobranca } from '@/components/pagar-cobranca'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import type { Cobranca, DestaqueView, SituacaoPro } from '@/lib/plataforma'

const diaCurto = (iso: string) => new Date(`${iso}T12:00:00`).toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: '2-digit' })

// Item 10.4: destaque pago na vitrine (por dia, na cidade ou na categoria
// do evento, marcado "Patrocinado").
export function DestaqueCard({ eventoId, publicado }: { eventoId: number; publicado: boolean }) {
  const [alvo, setAlvo] = useState<'cidade' | 'categoria'>('cidade')
  const [dias, setDias] = useState<string[]>([])
  const [cobranca, setCobranca] = useState<Cobranca | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const pro = useQuery({ queryKey: ['org-pro'], queryFn: () => api<SituacaoPro>('/org/pro') })
  const disp = useQuery({
    queryKey: ['org-destaque-disp', eventoId, alvo],
    enabled: publicado,
    retry: false,
    queryFn: () => api<{ vagas: Record<string, number>; preco_dia_centavos: number }>(`/org/eventos/${eventoId}/destaques?alvo=${alvo}`),
  })
  const meus = useQuery({ queryKey: ['org-destaques'], queryFn: () => api<DestaqueView[]>('/org/destaques') })
  const doEvento = (meus.data ?? []).filter((d) => d.evento_id === eventoId)

  const preco = disp.data?.preco_dia_centavos ?? 0
  const incluido = !!pro.data?.pro && !pro.data.destaque_incluido_usado && dias.length > 0
  const total = Math.max(0, dias.length - (incluido ? 1 : 0)) * preco

  const comprar = async () => {
    setErro(null)
    try {
      const c = await api<Cobranca>('/org/destaques', { method: 'POST', body: { evento_id: eventoId, alvo_tipo: alvo, dias: [...dias].sort() } })
      setDias([])
      if (c.status === 'paga') {
        meus.refetch()
        pro.refetch()
        disp.refetch()
      } else setCobranca(c)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível reservar.')
      disp.refetch()
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Destaque na vitrine</CardTitle>
        <p className="text-sm text-muted-foreground">
          Seu evento aparece no topo da página da cidade ou da categoria, marcado como "Patrocinado", nos dias que você escolher. Até{' '}
          {pro.data?.precos.destaque_vagas_dia ?? 3} eventos por lugar e dia. Se o evento for cancelado, os dias que ainda não aconteceram voltam.
          {pro.data?.pro ? ' No Pro, 1 dia a cada 30 sai de graça.' : ''}
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {!publicado ? (
          <p className="text-sm text-muted-foreground">Publique o evento para comprar destaque.</p>
        ) : cobranca ? (
          <PagarCobranca
            cobranca={cobranca}
            aoPagar={() => {
              meus.refetch()
              disp.refetch()
            }}
            aoFechar={() => setCobranca(null)}
          />
        ) : (
          <>
            <div role="radiogroup" aria-label="Onde destacar" className="flex flex-wrap gap-2">
              {(['cidade', 'categoria'] as const).map((a) => (
                <Button
                  key={a}
                  size="sm"
                  role="radio"
                  aria-checked={alvo === a}
                  variant={alvo === a ? 'default' : 'outline'}
                  onClick={() => {
                    setAlvo(a)
                    setDias([])
                  }}
                >
                  {a === 'cidade' ? 'Na cidade' : 'Na categoria'}
                </Button>
              ))}
            </div>
            {disp.error ? (
              <p className="text-sm text-muted-foreground">{disp.error instanceof ApiError ? disp.error.message : 'Indisponível.'}</p>
            ) : (
              <fieldset>
                <legend className="mb-2 text-sm font-medium text-foreground">Dias ({formatarCentavos(preco)} cada)</legend>
                <div className="grid grid-cols-3 gap-2 sm:grid-cols-5">
                  {Object.entries(disp.data?.vagas ?? {})
                    .sort(([a], [b]) => a.localeCompare(b))
                    .map(([dia, vagas]) => {
                      const marcado = dias.includes(dia)
                      return (
                        <button
                          key={dia}
                          type="button"
                          disabled={vagas <= 0 && !marcado}
                          aria-pressed={marcado}
                          onClick={() => setDias(marcado ? dias.filter((d) => d !== dia) : [...dias, dia])}
                          className={`flex flex-col items-center rounded-lg border px-2 py-2 text-xs transition-colors disabled:opacity-40 ${
                            marcado ? 'border-primary bg-primary text-primary-foreground' : 'border-border text-foreground hover:border-primary'
                          }`}
                        >
                          <span className="font-medium capitalize">{diaCurto(dia)}</span>
                          <span className={marcado ? '' : 'text-muted-foreground'}>{vagas > 0 ? `${vagas} vaga(s)` : 'lotado'}</span>
                        </button>
                      )
                    })}
                </div>
              </fieldset>
            )}
            {dias.length > 0 && (
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="text-sm text-foreground">
                  {dias.length} dia(s){incluido ? ' · 1 incluído no Pro' : ''} · <strong>{formatarCentavos(total)}</strong>
                </p>
                <Button onClick={comprar}>{total === 0 ? 'Ativar destaque' : 'Continuar para o pagamento'}</Button>
              </div>
            )}
            {!pro.data?.pro && (
              <p className="text-xs text-muted-foreground">
                Com o{' '}
                <Link to="/organizador/pro" className="underline underline-offset-4">
                  Evve Pro
                </Link>
                , 1 dia de destaque a cada 30 é por nossa conta.
              </p>
            )}
          </>
        )}
        {doEvento.length > 0 && (
          <ul className="flex flex-col gap-1 text-sm">
            {doEvento.map((d) => (
              <li key={d.id} className="flex justify-between gap-3">
                <span className="text-foreground">
                  {diaCurto(d.dia.slice(0, 10))} · {d.alvo_tipo === 'cidade' ? 'cidade' : 'categoria'}
                  {d.incluido_pro ? ' · incluído no Pro' : ''}
                </span>
                <span className="text-muted-foreground">{d.status === 'pago' ? 'ativo' : d.status === 'devolvido' ? 'devolvido' : 'aguardando pagamento'}</span>
              </li>
            ))}
          </ul>
        )}
        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
