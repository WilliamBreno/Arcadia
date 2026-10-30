import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Html5Qrcode } from 'html5-qrcode'
import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'

import { ApiError } from '@/lib/api'
import {
  chamar,
  definirModo,
  encerrar,
  gerarLinkTelao,
  lerQRCamarim,
  marcarStatus,
  obterPalco,
  proximo,
  rotuloStatus,
  rotuloTipo,
  type EstadoPalco,
  type ItemFila,
  type ModoTelao,
  type StatusPalco,
} from '@/lib/palco'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Carregando } from '@/components/carregando'

const corStatus: Record<StatusPalco, string> = {
  aguardando: 'bg-muted text-muted-foreground',
  presente: 'bg-secondary text-secondary-foreground',
  no_palco: 'bg-foreground text-background',
  apresentou: 'bg-muted text-foreground',
  ausente: 'bg-destructive/10 text-destructive',
}

function Etiqueta({ status }: { status: StatusPalco }) {
  return (
    <span className={`shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${corStatus[status]}`}>{rotuloStatus[status]}</span>
  )
}

// Operação do concurso no dia (item 8.3): aba Palco (chamar, telão) e aba
// Camarim (chegada dos participantes por QR ou nome). Organizador e equipe
// de check-in usam; o link do telão só o organizador gera.
export default function PalcoCamarim() {
  const { eventoId = '' } = useParams()
  const queryClient = useQueryClient()
  const [aba, setAba] = useState<'palco' | 'camarim'>('palco')
  const [erro, setErro] = useState<string | null>(null)
  const chave = ['palco', eventoId]
  const { data, isLoading, error } = useQuery({ queryKey: chave, queryFn: () => obterPalco(eventoId), refetchInterval: 5000 })

  useEffect(() => {
    document.title = 'Palco e camarim | Evve'
  }, [])

  const agir = async (f: () => Promise<EstadoPalco>) => {
    setErro(null)
    try {
      queryClient.setQueryData(chave, await f())
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível agora.')
    }
  }

  if (isLoading) return <Carregando />
  if (error || !data) {
    return (
      <main className="mx-auto max-w-lg px-4 py-16">
        <h1 className="text-2xl font-bold text-foreground">Sem acesso</h1>
        <p className="mt-2 text-muted-foreground">
          {error instanceof ApiError ? error.message : 'Não foi possível abrir o palco.'}
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-6">
      <h1 className="text-2xl font-semibold text-foreground">Palco e camarim</h1>
      <div role="tablist" aria-label="Seções" className="mt-4 grid grid-cols-2 gap-2">
        {(['palco', 'camarim'] as const).map((a) => (
          <button
            key={a}
            role="tab"
            aria-selected={aba === a}
            onClick={() => setAba(a)}
            className={`rounded-lg border px-3 py-2.5 text-sm font-medium ${aba === a ? 'border-foreground bg-muted text-foreground' : 'border-border text-muted-foreground'}`}
          >
            {a === 'palco' ? 'Palco' : 'Camarim'}
          </button>
        ))}
      </div>
      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}
      {aba === 'palco' ? (
        <AbaPalco eventoId={eventoId} estado={data} agir={agir} />
      ) : (
        <AbaCamarim eventoId={eventoId} estado={data} agir={agir} />
      )}
    </main>
  )
}

type PropsAba = { eventoId: string; estado: EstadoPalco; agir: (f: () => Promise<EstadoPalco>) => Promise<void> }

function AbaPalco({ eventoId, estado, agir }: PropsAba) {
  const atual = estado.fila.find((i) => i.ficha_id === estado.atual_id)
  const proximos = estado.fila.filter(
    (i) => i.ficha_id !== estado.atual_id && (i.status === 'presente' || i.status === 'aguardando'),
  )
  const [link, setLink] = useState<string | null>(null)
  const [erroLink, setErroLink] = useState<string | null>(null)

  const novoLink = async () => {
    setErroLink(null)
    try {
      const { token } = await gerarLinkTelao(eventoId)
      setLink(`${window.location.origin}/telao/${token}`)
    } catch (e) {
      setErroLink(
        e instanceof ApiError && e.status === 404
          ? 'Só o organizador do evento gera o link do telão.'
          : 'Não foi possível gerar.',
      )
    }
  }

  return (
    <div className="mt-5 flex flex-col gap-5">
      <section aria-labelledby="titulo-agora" className="rounded-xl border border-foreground p-4">
        <h2 id="titulo-agora" className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          No palco agora
        </h2>
        {atual ? (
          <div className="mt-2 flex items-center gap-3">
            {atual.foto_referencia && <img src={atual.foto_referencia} alt="" className="size-16 rounded-lg object-cover" />}
            <div className="min-w-0">
              <p className="text-xl font-semibold text-foreground">{atual.nome}</p>
              <p className="text-sm text-muted-foreground">
                {[atual.personagem, atual.obra, rotuloTipo[atual.tipo]].filter(Boolean).join(' · ')}
              </p>
            </div>
          </div>
        ) : (
          <p className="mt-2 text-foreground">Ninguém. {proximos[0] ? `O próximo é ${proximos[0].nome}.` : 'A fila acabou.'}</p>
        )}
        <div className="mt-4 flex flex-wrap gap-2">
          <Button size="lg" onClick={() => agir(() => proximo(eventoId))}>
            {proximos[0] ? `Chamar próximo: ${proximos[0].nome}` : atual ? 'Encerrar (fila acabou)' : 'Fila vazia'}
          </Button>
          {atual && (
            <Button size="lg" variant="outline" onClick={() => agir(() => encerrar(eventoId))}>
              Encerrar apresentação
            </Button>
          )}
        </div>
      </section>

      <section aria-labelledby="titulo-telao" className="flex flex-col gap-2">
        <h2 id="titulo-telao" className="font-semibold text-foreground">
          Telão
        </h2>
        <div role="radiogroup" aria-label="O que o telão mostra" className="grid grid-cols-3 gap-2">
          {(
            [
              ['intervalo', 'Intervalo'],
              ['apresentacao', 'Apresentação'],
              ['resultado', 'Resultado'],
            ] as [ModoTelao, string][]
          ).map(([m, r]) => (
            <button
              key={m}
              role="radio"
              aria-checked={estado.modo === m}
              onClick={() => agir(() => definirModo(eventoId, m))}
              className={`rounded-lg border px-2 py-2 text-sm ${estado.modo === m ? 'border-foreground bg-muted font-medium text-foreground' : 'border-border text-muted-foreground'}`}
            >
              {r}
            </button>
          ))}
        </div>
        <p className="text-xs text-muted-foreground">
          "Resultado" mostra o pódio só depois que o resultado for liberado. Ao chamar alguém, o telão vai sozinho para
          "Apresentação".
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" onClick={novoLink}>
            {estado.telao_ativo ? 'Gerar novo link do telão' : 'Gerar link do telão'}
          </Button>
          {estado.telao_ativo && !link && (
            <span className="text-xs text-muted-foreground">Gerar um novo desliga o anterior.</span>
          )}
        </div>
        {link && (
          <div className="flex flex-col gap-1 rounded-lg bg-muted p-3 text-sm">
            <p className="text-foreground">Abra este link no computador ou na TV do telão (não precisa entrar na conta):</p>
            <code className="break-all text-xs text-foreground">{link}</code>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={() => navigator.clipboard.writeText(link).catch(() => undefined)}>
                Copiar
              </Button>
              <a href={link} target="_blank" rel="noopener" className="self-center text-sm underline underline-offset-4">
                Abrir
              </a>
            </div>
            <p className="text-xs text-muted-foreground">Não compartilhe: quem tiver o link vê o telão. Se vazar, gere outro.</p>
          </div>
        )}
        {erroLink && <p className="text-sm text-destructive">{erroLink}</p>}
      </section>

      <section aria-labelledby="titulo-ordem">
        <h2 id="titulo-ordem" className="mb-2 font-semibold text-foreground">
          Ordem de apresentação
        </h2>
        <ul className="flex flex-col divide-y divide-border rounded-xl border border-border">
          {estado.fila.map((i, k) => (
            <li key={i.ficha_id} className="flex items-center gap-3 px-3 py-2.5">
              <span className="w-6 text-right text-sm tabular-nums text-muted-foreground">{i.ordem ?? k + 1}</span>
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-foreground">{i.nome}</p>
                <p className="truncate text-xs text-muted-foreground">
                  {[i.personagem, rotuloTipo[i.tipo]].filter(Boolean).join(' · ')}
                </p>
              </div>
              <Etiqueta status={i.status} />
              {i.ficha_id !== estado.atual_id && (
                <Button size="sm" variant="ghost" onClick={() => agir(() => chamar(eventoId, i.ficha_id))}>
                  Chamar
                </Button>
              )}
            </li>
          ))}
        </ul>
        {estado.fila.length === 0 && <p className="text-sm text-muted-foreground">Nenhum participante aprovado ainda.</p>}
      </section>
    </div>
  )
}

const SCANNER = 'leitor-camarim'

function AbaCamarim({ eventoId, estado, agir }: PropsAba) {
  const queryClient = useQueryClient()
  const [busca, setBusca] = useState('')
  const [lendo, setLendo] = useState(false)
  const [aviso, setAviso] = useState<{ ok: boolean; texto: string } | null>(null)
  const ocupado = useRef(false)

  useEffect(() => {
    if (!lendo) return
    const leitor = new Html5Qrcode(SCANNER)
    leitor
      .start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: 230 },
        async (texto) => {
          if (ocupado.current) return
          ocupado.current = true
          try {
            const r = await lerQRCamarim(eventoId, texto)
            setAviso({
              ok: true,
              texto: r.chegou_agora ? `${r.participante.nome} chegou ✔` : `${r.participante.nome} já tinha chegado`,
            })
            queryClient.invalidateQueries({ queryKey: ['palco', eventoId] })
          } catch (e) {
            setAviso({ ok: false, texto: e instanceof ApiError ? e.message : 'Não foi possível ler.' })
          } finally {
            window.setTimeout(() => (ocupado.current = false), 1500)
          }
        },
        undefined,
      )
      .catch(() => setAviso({ ok: false, texto: 'Não foi possível acessar a câmera. Confira a permissão do navegador.' }))
    return () => {
      leitor
        .stop()
        .then(() => leitor.clear())
        .catch(() => {})
    }
  }, [lendo, eventoId, queryClient])

  const termo = busca.trim().toLowerCase()
  const lista = estado.fila.filter((i) => !termo || `${i.nome} ${i.personagem ?? ''}`.toLowerCase().includes(termo))
  const presentes = estado.fila.filter(
    (i) => i.status === 'presente' || i.status === 'no_palco' || i.status === 'apresentou',
  ).length

  const botoes = (i: ItemFila) =>
    (
      [
        ['presente', 'Chegou'],
        ['ausente', 'Faltou'],
        ['aguardando', 'Desfazer'],
      ] as [StatusPalco, string][]
    ).filter(([s]) => s !== i.status && !(s === 'aguardando' && i.status === 'aguardando'))

  return (
    <div className="mt-5 flex flex-col gap-4">
      <p role="status" className="text-sm text-foreground">
        <strong>{presentes}</strong> de {estado.fila.length} participantes já chegaram.
      </p>
      <div className="flex flex-col gap-2">
        <Button variant={lendo ? 'outline' : 'default'} onClick={() => setLendo((v) => !v)}>
          {lendo ? 'Fechar câmera' : 'Ler QR do participante'}
        </Button>
        {lendo && <div id={SCANNER} className="w-full overflow-hidden rounded-xl" />}
        {aviso && (
          <p
            role="status"
            className={`rounded-lg p-3 text-sm ${aviso.ok ? 'bg-muted text-foreground' : 'bg-destructive/10 text-destructive'}`}
          >
            {aviso.texto}
          </p>
        )}
      </div>
      <div className="flex flex-col gap-1.5">
        <label htmlFor="busca-camarim" className="text-sm font-medium text-foreground">
          Procurar pelo nome ou personagem
        </label>
        <Input id="busca-camarim" value={busca} onChange={(e) => setBusca(e.target.value)} placeholder="Ex.: Nezuko" />
      </div>
      <ul className="flex flex-col divide-y divide-border rounded-xl border border-border">
        {lista.map((i) => (
          <li key={i.ficha_id} className="flex flex-wrap items-center gap-2 px-3 py-2.5">
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium text-foreground">{i.nome}</p>
              <p className="truncate text-xs text-muted-foreground">
                {[i.personagem, rotuloTipo[i.tipo]].filter(Boolean).join(' · ')}
              </p>
            </div>
            <Etiqueta status={i.status} />
            {i.status !== 'no_palco' &&
              botoes(i).map(([s, r]) => (
                <Button
                  key={s}
                  size="sm"
                  variant={s === 'presente' ? 'outline' : 'ghost'}
                  onClick={() => agir(() => marcarStatus(eventoId, i.ficha_id, s))}
                >
                  {r}
                </Button>
              ))}
          </li>
        ))}
      </ul>
    </div>
  )
}
