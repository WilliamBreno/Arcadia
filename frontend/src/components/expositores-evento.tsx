import { useQuery, useQueryClient } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { Desistencia, ResumoPagamento } from '@/components/minha-inscricao'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuth } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import {
  expositoresDoEvento,
  meuEspaco,
  pagarEspaco,
  renomearCredencial,
  rotuloStatusExpositor,
  type Credencial,
  type MeuEspaco,
} from '@/lib/expositor'

const caixa = 'mt-8 scroll-mt-20 rounded-xl border border-border p-4 sm:p-5'

// Seção "Expositores" da página do evento (item 8.6): mapa das mesas, quem
// expõe e o convite para se candidatar. Só aparece se o evento tem espaços.
export function ExpositoresDoEvento({ slug, timezone }: { slug: string; timezone?: string }) {
  const { usuario } = useAuth()
  const { data } = useQuery({ queryKey: ['expositores', slug], queryFn: () => expositoresDoEvento(slug).catch(() => null) })
  const { data: meu } = useQuery({
    queryKey: ['meu-espaco', slug, usuario?.id],
    enabled: !!usuario,
    retry: false,
    queryFn: () => meuEspaco(slug).catch((e) => (e instanceof ApiError && e.status === 404 ? null : Promise.reject(e))),
  })
  if (!data || (data.mesas.length === 0 && data.tipos.length === 0)) return null

  const setores = [...new Set(data.mesas.map((m) => m.setor))]
  const menorPreco = data.tipos.length ? Math.min(...data.tipos.map((t) => t.preco_centavos)) : 0

  return (
    <>
      {meu && <MeuEspacoBloco slug={slug} meu={meu} timezone={timezone} />}
      <section id="expositores" aria-labelledby="titulo-expositores" className={caixa}>
        <h2 id="titulo-expositores" className="text-xl font-semibold text-foreground">
          Expositores
        </h2>

        {data.expositores.length > 0 ? (
          <ul className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
            {data.expositores.map((x) => (
              <li key={x.mesa} className="rounded-lg border border-border p-3">
                <p className="flex items-baseline justify-between gap-2">
                  <span className="font-medium text-foreground">{x.marca}</span>
                  <span className="shrink-0 text-xs tabular-nums text-muted-foreground">mesa {x.mesa}</span>
                </p>
                {x.categoria && <p className="text-xs text-muted-foreground">{x.categoria}</p>}
                {x.descricao && <p className="mt-1 text-sm text-foreground text-pretty">{x.descricao}</p>}
                {x.instagram && (
                  <a
                    href={`https://instagram.com/${x.instagram}`}
                    target="_blank"
                    rel="noopener noreferrer nofollow"
                    className="mt-1 inline-block text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground"
                  >
                    @{x.instagram}
                  </a>
                )}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-1 text-sm text-muted-foreground">Os expositores confirmados aparecem aqui.</p>
        )}

        {setores.length > 0 && (
          <div className="mt-5">
            <h3 className="text-sm font-medium text-foreground">Mapa das mesas</h3>
            <p className="mt-0.5 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1.5">
                <span className="inline-block size-3 rounded-sm border border-border" aria-hidden="true" /> livre
              </span>
              <span className="inline-flex items-center gap-1.5">
                <span className="inline-block size-3 rounded-sm bg-foreground" aria-hidden="true" /> ocupada
              </span>
            </p>
            {setores.map((setor) => (
              <div key={setor} className="mt-3">
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Setor {setor}</p>
                <ul className="mt-1 grid gap-1.5" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(4.5rem, 1fr))' }}>
                  {data.mesas
                    .filter((m) => m.setor === setor)
                    .map((m) => (
                      <li
                        key={m.codigo}
                        title={m.marca ? `${m.codigo} · ${m.marca}` : `${m.codigo} · ${m.ocupada ? 'ocupada' : 'livre'} · ${m.tipo}`}
                        aria-label={m.marca ? `Mesa ${m.codigo}, ${m.marca}` : `Mesa ${m.codigo}, ${m.ocupada ? 'ocupada' : 'livre'}`}
                        className={`flex min-h-12 flex-col justify-center rounded-md border px-1.5 py-1 text-center text-xs ${
                          m.ocupada ? 'border-foreground bg-foreground text-background' : 'border-border text-foreground'
                        }`}
                      >
                        <span className="font-semibold tabular-nums">{m.codigo}</span>
                        {m.marca && <span className="truncate opacity-80">{m.marca}</span>}
                      </li>
                    ))}
                </ul>
              </div>
            ))}
          </div>
        )}

        {data.aberto && !meu && (
          <div className="mt-5 flex flex-wrap items-center justify-between gap-3 rounded-lg bg-muted p-3">
            <p className="text-sm text-foreground">
              Quer expor? {data.tipos.map((t) => `${t.nome}: ${t.preco_centavos ? formatarCentavos(t.preco_centavos) : 'gratuito'}`).join(' · ')}
              {menorPreco > 0 && ' + taxa'}. Você só paga se for aprovado.
            </p>
            <Link to={`/e/${slug}/expositor`} className="text-sm font-medium text-primary underline-offset-4 hover:underline">
              Quero expor
            </Link>
          </div>
        )}
      </section>
    </>
  )
}

function quando(iso: string | null | undefined, fuso?: string) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short', timeZone: fuso || 'America/Maceio' })
}

function MeuEspacoBloco({ slug, meu, timezone }: { slug: string; meu: MeuEspaco; timezone?: string }) {
  const queryClient = useQueryClient()
  const recarregar = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ['meu-espaco', slug] }),
      queryClient.invalidateQueries({ queryKey: ['expositores', slug] }),
    ])
  const v = meu.valores
  return (
    <section id="meu-espaco" aria-labelledby="titulo-meu-espaco" className={caixa}>
      <h2 id="titulo-meu-espaco" className="text-xl font-semibold text-foreground">
        Seu espaço · {meu.marca}
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">
        {rotuloStatusExpositor[meu.status]}
        {meu.mesa && ` · mesa ${meu.mesa}`} · {meu.tipo.nome}
      </p>

      {meu.status === 'pendente' && (
        <p className="mt-2 text-sm text-foreground">Candidatura enviada. Você recebe um e-mail quando o organizador decidir.</p>
      )}
      {meu.status === 'rejeitado' && (
        <p className="mt-2 text-sm text-foreground">
          A candidatura não foi aprovada{meu.motivo_rejeicao ? `: ${meu.motivo_rejeicao}` : '.'}{' '}
          <Link to={`/e/${slug}/expositor`} className="underline underline-offset-4">
            Candidatar de novo
          </Link>
        </p>
      )}
      {meu.status === 'desistiu' && <p className="mt-2 text-sm text-foreground">Você desistiu do espaço.</p>}

      {meu.status === 'aguardando_pagamento' && v && (
        <ResumoPagamento
          valores={v}
          rotuloItem={`${meu.tipo.nome} (mesa ${meu.mesa})`}
          aviso={
            <>
              Aprovado! Para garantir a mesa, pague{v.prazo_em ? <> até <strong>{quando(v.prazo_em, timezone)}</strong></> : null}. Sem o pagamento, a mesa é
              liberada.
            </>
          }
          pagar={(g) => pagarEspaco(slug, g)}
        />
      )}

      {meu.status === 'aprovado' && (
        <div className="mt-3 flex flex-col gap-4">
          <p className="text-sm text-foreground">
            Espaço confirmado{v?.pago ? <> e pago ({formatarCentavos(v.total_pago_centavos)})</> : null}. Mostre a credencial de cada pessoa na entrada; ela
            vale para entrar e sair durante o evento.
          </p>
          <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {meu.credenciais.map((c, i) => (
              <CredencialCartao key={c.id} credencial={c} indice={i} aoSalvar={recarregar} />
            ))}
          </ul>
          {v?.pago && v.item_pago_id && (
            <Desistencia itemId={v.item_pago_id} rotulo="Desistir do espaço" manter="Manter o espaço" aoConcluir={recarregar} />
          )}
        </div>
      )}
    </section>
  )
}

function CredencialCartao({ credencial: c, indice, aoSalvar }: { credencial: Credencial; indice: number; aoSalvar: () => Promise<unknown> }) {
  const [nome, setNome] = useState(c.nome)
  const [salvando, setSalvando] = useState(false)
  const mudou = nome.trim() !== c.nome
  const salvar = async () => {
    setSalvando(true)
    try {
      await renomearCredencial(c.id, nome)
      await aoSalvar()
    } finally {
      setSalvando(false)
    }
  }
  return (
    <li className="flex flex-col items-center gap-2 rounded-lg border border-border p-3">
      <div className="rounded-md bg-white p-2">
        <QRCodeSVG value={c.payload} size={150} level="M" aria-label={`QR da credencial ${indice + 1}`} />
      </div>
      <label htmlFor={`cred-${c.id}`} className="sr-only">
        Nome na credencial {indice + 1}
      </label>
      <div className="flex w-full gap-2">
        <Input id={`cred-${c.id}`} placeholder={indice === 0 ? 'Responsável' : 'Nome do ajudante'} value={nome} onChange={(e) => setNome(e.target.value)} />
        {mudou && (
          <Button size="sm" variant="outline" onClick={salvar} disabled={salvando}>
            Salvar
          </Button>
        )}
      </div>
      <p className="text-xs text-muted-foreground">{c.entrou_em ? `Entrou em ${new Date(c.entrou_em).toLocaleString('pt-BR')}` : 'Ainda não entrou'}</p>
    </li>
  )
}
