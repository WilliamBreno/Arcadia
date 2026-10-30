import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { Carregando } from '@/components/carregando'
import { Button } from '@/components/ui/button'
import { VoltarLink } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { formatarBP, type Coproducao } from '@/lib/coproducao'
import { formatarCentavos } from '@/lib/evento'
import { obterFinanceiroEvento } from '@/lib/financeiro'

// Item 9.8: convites e coproduções de outros organizadores, com a parte
// de cada evento.
export default function Coproducoes() {
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({ queryKey: ['org-coproducoes'], queryFn: () => api<Coproducao[]>('/org/coproducoes') })
  const [erro, setErro] = useState<string | null>(null)

  const executar = async (acao: () => Promise<unknown>) => {
    setErro(null)
    try {
      await acao()
      queryClient.invalidateQueries({ queryKey: ['org-coproducoes'] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível responder.')
    }
  }

  if (isLoading) return <Carregando />
  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-muted-foreground">Só o dono do perfil de organizador responde a coproduções.</p>
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar ao painel</VoltarLink>
      <h1 className="mb-2 text-2xl font-semibold text-foreground">Coproduções</h1>
      <p className="mb-6 text-sm text-muted-foreground">
        Eventos de outros organizadores em que você recebe uma parte do repasse, no seu Pix. O repasse aparece em Repasses quando o evento termina.
      </p>
      {data?.length === 0 && <p className="text-muted-foreground">Nenhuma coprodução por enquanto.</p>}
      <ul className="flex flex-col gap-3">
        {data?.map((c) => (
          <li key={c.id} className="flex flex-col gap-3 rounded-xl border border-border p-4">
            <div>
              <p className="font-medium text-foreground">{c.evento_titulo}</p>
              <p className="text-sm text-muted-foreground">
                de {c.principal_nome}
                {c.evento_inicio_em ? ` · ${new Date(c.evento_inicio_em).toLocaleDateString('pt-BR')}` : ''}
              </p>
            </div>
            {c.status === 'ativa' && <ParteDoEvento eventoId={c.evento_id} percentual={c.percentual_bp} />}
            {c.proposto_bp !== null && (
              <div className="flex flex-col gap-2 rounded-lg bg-muted p-3 text-sm">
                <p className="text-foreground">
                  {c.status === 'convidado'
                    ? `Convite para receber ${formatarBP(c.proposto_bp)} do repasse líquido.`
                    : c.proposto_bp === 0
                      ? 'O organizador principal propôs tirar você da coprodução.'
                      : `O organizador principal propôs mudar sua parte de ${formatarBP(c.percentual_bp)} para ${formatarBP(c.proposto_bp)}.`}
                </p>
                <div className="flex gap-2">
                  <Button size="sm" onClick={() => executar(() => api(`/org/coproducoes/${c.id}/responder`, { method: 'POST', body: { aceitar: true } }))}>
                    Aceitar
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => executar(() => api(`/org/coproducoes/${c.id}/responder`, { method: 'POST', body: { aceitar: false } }))}>
                    Recusar
                  </Button>
                </div>
              </div>
            )}
            {c.status === 'ativa' && c.proposto_bp === null && c.pode_sair && (
              <Button
                size="sm"
                variant="ghost"
                className="self-start"
                onClick={() => {
                  if (window.confirm('Sair desta coprodução?')) executar(() => api(`/org/coproducoes/${c.id}/sair`, { method: 'POST' }))
                }}
              >
                Sair da coprodução
              </Button>
            )}
          </li>
        ))}
      </ul>
      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}
    </main>
  )
}

function ParteDoEvento({ eventoId, percentual }: { eventoId: number; percentual: number }) {
  const { data } = useQuery({ queryKey: ['org-evento-financeiro', eventoId], queryFn: () => obterFinanceiroEvento(eventoId) })
  return (
    <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
      <dt className="text-muted-foreground">Sua parte</dt>
      <dd className="text-right font-medium tabular-nums text-foreground">{formatarBP(percentual)}</dd>
      {data && (
        <>
          <dt className="text-muted-foreground">Bruto da sua parte</dt>
          <dd className="text-right tabular-nums text-foreground">{formatarCentavos(data.bruto_centavos)}</dd>
          <dt className="text-muted-foreground">Líquido da sua parte</dt>
          <dd className="text-right tabular-nums text-foreground">{formatarCentavos(data.liquido_centavos)}</dd>
          {data.liberar_em && (
            <>
              <dt className="text-muted-foreground">Repasse a partir de</dt>
              <dd className="text-right tabular-nums text-foreground">{new Date(data.liberar_em).toLocaleDateString('pt-BR')}</dd>
            </>
          )}
        </>
      )}
    </dl>
  )
}
