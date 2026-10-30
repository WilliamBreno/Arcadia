import { Html5Qrcode } from 'html5-qrcode'
import { useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { VoltarBotao } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { lerCredencial } from '@/lib/expositor'
import { BuscaAdicionais } from '@/components/busca-adicionais'
import { lerCredencialEvento, portaoSalvo, portoesCheckin, rotuloFuncao, salvarPortao, validarImportado } from '@/lib/dia-evento'
import { buscarCheckin, resumoCheckin, validarCheckin, type ItemBusca, type ResumoCheckin, type ValidarCheckinResposta } from '@/lib/checkin'
import {
  baixarPacote,
  enfileirar,
  lerPacote,
  listarFila,
  salvarPacote,
  sincronizarFila,
  validarOffline,
  type Pacote,
  type ResultadoSync,
} from '@/lib/offline'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const SCANNER_ID = 'leitor-qr-checkin'

const corPorResultado: Record<ValidarCheckinResposta['resultado'], string> = {
  valido: 'bg-green-600 border-green-700',
  ja_utilizado: 'bg-yellow-500 border-yellow-600',
  cancelado: 'bg-red-600 border-red-700',
  outro_evento: 'bg-red-600 border-red-700',
  nao_encontrado: 'bg-red-600 border-red-700',
  qr_expirado: 'bg-orange-500 border-orange-600',
  fora_da_sessao: 'bg-orange-500 border-orange-600',
  em_revenda: 'bg-orange-500 border-orange-600',
  requer_internet: 'bg-orange-500 border-orange-600',
  expositor: 'bg-green-600 border-green-700',
  credencial: 'bg-green-600 border-green-700',
  adicional: 'bg-blue-600 border-blue-700',
  adicional_entregue: 'bg-yellow-500 border-yellow-600',
  credencial_invalida: 'bg-red-600 border-red-700',
}

const tituloPorResultado: Record<ValidarCheckinResposta['resultado'], string> = {
  valido: '✅ Entrada liberada',
  ja_utilizado: '⚠️ Já utilizado',
  cancelado: '❌ Ingresso cancelado',
  outro_evento: '❌ Ingresso de outro evento',
  nao_encontrado: '❌ Código não encontrado',
  qr_expirado: '⏱️ QR desatualizado — peça para abrir o ingresso de novo',
  fora_da_sessao: '⛔ Ingresso não vale para esta sessão/horário',
  em_revenda: '⛔ Ingresso sendo revendido agora — peça para retirar da revenda ou aguardar',
  requer_internet: '📶 QR dinâmico: precisa de internet para validar',
  expositor: '✅ Expositor liberado',
  credencial: '✅ Credencial liberada',
  adicional: '🎁 Entregar o adicional',
  adicional_entregue: '⚠️ Adicional já entregue',
  credencial_invalida: '❌ Credencial inválida ou desativada',
}

export default function Checkin() {
  const { eventoId } = useParams()
  const id = Number(eventoId)

  const [acessoLiberado, setAcessoLiberado] = useState<boolean | null>(null)
  const [erroAcesso, setErroAcesso] = useState<string | null>(null)
  const [erroCamera, setErroCamera] = useState<string | null>(null)
  const [resultado, setResultado] = useState<ValidarCheckinResposta | null>(null)
  const [resumo, setResumo] = useState<ResumoCheckin | null>(null)
  const [busca, setBusca] = useState('')
  const [itensBusca, setItensBusca] = useState<ItemBusca[]>([])
  const [processando, setProcessando] = useState(false)

  const [online, setOnline] = useState(navigator.onLine)
  const [pacoteInfo, setPacoteInfo] = useState<{ geradoEm: string; total: number } | null>(null)
  const [pendentes, setPendentes] = useState(0)
  const [msgOffline, setMsgOffline] = useState<string | null>(null)
  const [conflitos, setConflitos] = useState<{ nome: string; codigo: string }[]>([])
  const [modoOffline, setModoOffline] = useState(false)
  // Item 9.6: portão deste aparelho (vai junto em cada entrada).
  const [portoes, setPortoes] = useState<string[]>([])
  const [portao, setPortao] = useState(() => portaoSalvo(id))
  const portaoRef = useRef(portao)
  portaoRef.current = portao

  const leitorRef = useRef<Html5Qrcode | null>(null)
  const processandoRef = useRef(false)
  const pacoteRef = useRef<Pacote | null>(null)

  const carregarLocal = async () => {
    const p = await lerPacote(id)
    pacoteRef.current = p ?? null
    setPacoteInfo(p ? { geradoEm: p.gerado_em, total: p.ingressos.length } : null)
    setPendentes((await listarFila(id)).length)
  }

  const baixar = async () => {
    setMsgOffline(null)
    try {
      const p = await baixarPacote(id)
      await salvarPacote(p)
      await carregarLocal()
      setMsgOffline(`Pacote salvo: ${p.ingressos.length} ingresso(s). Já pode ficar sem internet.`)
    } catch (e) {
      setMsgOffline(e instanceof ApiError ? e.message : 'Sem conexão para baixar o pacote.')
    }
  }

  const sincronizar = async () => {
    setMsgOffline('Sincronizando…')
    try {
      const r = await sincronizarFila(id)
      const ruins = r.resultados.filter((x: ResultadoSync) => x.resultado !== 'aceito')
      setConflitos(ruins.map((x) => ({ nome: r.nomes[x.entrada_id] ?? '', codigo: `${x.codigo} (${x.resultado})` })))
      setMsgOffline(r.enviados === 0 ? 'Nada pendente.' : `${r.enviados - ruins.length} entrada(s) confirmada(s), ${ruins.length} com problema.`)
      await carregarLocal()
      atualizarResumo()
    } catch {
      setMsgOffline('Sem conexão — as entradas continuam guardadas no aparelho.')
    }
  }

  useEffect(() => {
    carregarLocal()
    const ligou = () => {
      setOnline(true)
      sincronizar()
    }
    const caiu = () => setOnline(false)
    window.addEventListener('online', ligou)
    window.addEventListener('offline', caiu)
    if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {})
    return () => {
      window.removeEventListener('online', ligou)
      window.removeEventListener('offline', caiu)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  const atualizarResumo = async () => {
    try {
      setResumo(await resumoCheckin(id))
    } catch {
      // contador é só informativo, ignora falha silenciosamente
    }
  }

  // Confirma acesso (staff ou organizador do evento) antes de ligar a câmera.
  useEffect(() => {
    resumoCheckin(id)
      .then((r) => {
        setResumo(r)
        setAcessoLiberado(true)
        portoesCheckin(id)
          .then((p) => setPortoes(p.portoes))
          .catch(() => {})
      })
      .catch(async (e) => {
        // sem rede + pacote salvo no aparelho: abre em modo offline
        if (!(e instanceof ApiError) && (await lerPacote(id))) {
          setModoOffline(true)
          setAcessoLiberado(true)
          return
        }
        setAcessoLiberado(false)
        setErroAcesso(e instanceof ApiError ? e.message : 'Sem conexão e sem pacote offline baixado neste aparelho.')
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  useEffect(() => {
    if (!acessoLiberado) return

    const leitor = new Html5Qrcode(SCANNER_ID)
    leitorRef.current = leitor

    leitor
      .start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: 250 },
        (textoDecodificado) => processarLeitura(textoDecodificado),
        undefined,
      )
      .catch(() => setErroCamera('Não foi possível acessar a câmera. Confira a permissão do navegador.'))

    return () => {
      leitor
        .stop()
        .then(() => leitor.clear())
        .catch(() => {})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [acessoLiberado])

  const processarLeitura = async (texto: string) => {
    if (processandoRef.current) return
    texto = texto.trim()
    if (!texto) return
    const separador = texto.indexOf(':')
    const liberarLeitor = () =>
      setTimeout(() => {
        setResultado(null)
        processandoRef.current = false
        setProcessando(false)
        leitorRef.current?.resume()
      }, 2500)

    processandoRef.current = true
    setProcessando(true)
    leitorRef.current?.pause(true)

    // Item 8.6: credencial de expositor (EVVEX:...), entra e sai à vontade.
    if (texto.startsWith('EVVEX:')) {
      try {
        const r = await lerCredencial(id, texto)
        setResultado({
          resultado: 'expositor',
          titular_nome: r.nome || 'Credencial sem nome',
          tipo_ingresso_nome: `${r.marca}${r.mesa ? ` · mesa ${r.mesa}` : ''}${r.primeira_vez ? '' : ' · reentrada'}`,
        })
      } catch {
        setResultado({ resultado: 'credencial_invalida' })
      }
      setTimeout(() => {
        setResultado(null)
        processandoRef.current = false
        setProcessando(false)
        leitorRef.current?.resume()
      }, 2500)
      return
    }

    // Item 10.3: adicional (EVVEA:...) — marca entregue, não dá entrada.
    if (texto.startsWith('EVVEA:')) {
      try {
        const r = await api<{ nome: string; titular: string; primeira_vez: boolean; entregue_em: string | null }>(
          `/checkin/eventos/${id}/adicionais/entregar`,
          { method: 'POST', body: { texto } },
        )
        setResultado({
          resultado: r.primeira_vez ? 'adicional' : 'adicional_entregue',
          titular_nome: r.titular,
          tipo_ingresso_nome: r.nome,
          utilizado_em: !r.primeira_vez && r.entregue_em ? new Date(r.entregue_em).toLocaleTimeString('pt-BR') : undefined,
        })
      } catch (e) {
        setResultado({ resultado: e instanceof ApiError && e.status === 409 ? 'cancelado' : 'nao_encontrado' })
      }
      liberarLeitor()
      return
    }

    // Item 9.6: credencial impressa (EVVEC:...) de equipe, jurado e imprensa.
    if (texto.startsWith('EVVEC:')) {
      try {
        const r = await lerCredencialEvento(id, texto, portaoRef.current)
        setResultado({ resultado: 'credencial', titular_nome: r.nome, tipo_ingresso_nome: `${rotuloFuncao[r.funcao]}${r.primeira_vez ? '' : ' · reentrada'}` })
      } catch {
        setResultado({ resultado: 'credencial_invalida' })
      }
      liberarLeitor()
      return
    }

    // QR de outra plataforma (ingresso importado): o texto inteiro é o código.
    const importado = async () => {
      const r = await validarImportado(id, texto, portaoRef.current)
      setResultado(r)
      if (r.resultado === 'valido') atualizarResumo()
    }
    if (separador === -1) {
      try {
        await importado()
      } catch (e) {
        setResultado({ resultado: e instanceof ApiError ? 'nao_encontrado' : 'requer_internet' })
      }
      liberarLeitor()
      return
    }

    const codigo = texto.slice(0, separador)
    const qrToken = texto.slice(separador + 1)

    try {
      const resp = await validarCheckin(id, codigo, qrToken, portaoRef.current)
      if (resp.resultado === 'nao_encontrado') await importado()
      else setResultado(resp)
      if (resp.resultado === 'valido') atualizarResumo()
    } catch (e) {
      if (e instanceof ApiError) {
        setResultado({ resultado: 'nao_encontrado' })
      } else if (pacoteRef.current) {
        // sem rede: valida no aparelho com o pacote e guarda na fila
        const d = await validarOffline(pacoteRef.current, await listarFila(id), codigo, qrToken)
        if (d.entrada) {
          await enfileirar(d.entrada)
          setPendentes((n) => n + 1)
        }
        setResultado({ ...d.resposta, tipo_ingresso_nome: `${d.resposta.tipo_ingresso_nome ?? ''} · OFFLINE`.replace(/^ · /, '') })
      } else {
        setResultado({ resultado: 'requer_internet' })
      }
    }

    setTimeout(() => {
      setResultado(null)
      processandoRef.current = false
      setProcessando(false)
      leitorRef.current?.resume()
    }, 2500)
  }

  const buscarManual = async (valor: string) => {
    setBusca(valor)
    if (valor.trim().length < 2) {
      setItensBusca([])
      return
    }
    try {
      setItensBusca(await buscarCheckin(id, valor))
    } catch {
      setItensBusca([])
    }
  }

  if (acessoLiberado === null) {
    return <p className="p-8 text-center text-muted-foreground">Verificando acesso…</p>
  }

  if (!acessoLiberado) {
    return (
      <main className="mx-auto max-w-md px-4 py-12 text-center">
        <p className="text-destructive">{erroAcesso}</p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-md px-4 py-8">
      <VoltarBotao fallback="/meus-eventos" />
      <div className="mb-4 flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold text-foreground">Check-in</h1>
        {/* Item 8.3: a mesma equipe opera o camarim e o palco. */}
        <span className="flex gap-3">
          {/* Item 9.4: venda na porta. */}
          <Link to={`/porta/${id}`} className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground">
            Vender na porta
          </Link>
          <Link to={`/ao-vivo/${id}`} className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground">
            Ao vivo
          </Link>
          <Link to={`/palco/${id}`} className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground">
            Palco e camarim
          </Link>
        </span>
      </div>

      <div className="mb-4 flex items-center justify-between rounded-lg border border-border px-4 py-2">
        <span className="text-sm text-muted-foreground">Entradas</span>
        <span className="text-lg font-semibold text-foreground">
          {resumo ? `${resumo.total_utilizados} / ${resumo.total_pagos}` : '—'}
        </span>
      </div>

      {portoes.length > 0 && (
        <div className="mb-4 flex flex-col gap-1.5">
          <label htmlFor="portao" className="text-sm font-medium text-foreground">
            Portão deste aparelho
          </label>
          <select
            id="portao"
            className="h-10 rounded-md border border-input bg-background px-3 text-sm text-foreground"
            value={portao}
            onChange={(e) => {
              setPortao(e.target.value)
              salvarPortao(id, e.target.value)
            }}
          >
            <option value="">Escolha o portão</option>
            {portoes.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
      )}

      {erroCamera && <p className="mb-2 text-sm text-destructive">{erroCamera}</p>}

      <div className="relative overflow-hidden rounded-lg border border-border">
        <div id={SCANNER_ID} className="w-full" />

        {resultado && (
          <div
            className={`absolute inset-0 flex flex-col items-center justify-center gap-2 border-4 p-4 text-center text-white ${corPorResultado[resultado.resultado]}`}
          >
            <p className="text-lg font-bold">{tituloPorResultado[resultado.resultado]}</p>
            {resultado.titular_nome && <p className="text-base">{resultado.titular_nome}</p>}
            {resultado.tipo_ingresso_nome && <p className="text-sm opacity-90">{resultado.tipo_ingresso_nome}</p>}
            {resultado.lugar && <p className="text-lg font-semibold">{resultado.lugar}</p>}
            {resultado.meia_entrada && resultado.resultado === 'valido' && (
              <p className="rounded bg-white px-3 py-1 text-base font-bold text-red-700">MEIA-ENTRADA — conferir documento</p>
            )}
            {resultado.utilizado_em && <p className="text-sm opacity-90">às {resultado.utilizado_em}</p>}
          </div>
        )}
      </div>

      {processando && !resultado && <p className="mt-2 text-center text-sm text-muted-foreground">Validando…</p>}

      <div className="mt-6">
        <Input placeholder="Buscar por nome, e-mail ou código" value={busca} onChange={(e) => buscarManual(e.target.value)} />
        {itensBusca.length > 0 && (
          <div className="mt-2 flex flex-col gap-2">
            {itensBusca.map((i) => (
              <div key={i.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
                <div>
                  <p className="text-sm font-medium text-foreground">{i.titular_nome}</p>
                  <p className="text-xs text-muted-foreground">
                    {i.tipo_ingresso_nome}
                    {i.meia_entrada ? ' (meia — conferir documento)' : ''} · {i.codigo}
                    {i.lugar ? ` · ${i.lugar}` : ''}
                  </p>
                </div>
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs ${
                    i.status === 'utilizado' ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800'
                  }`}
                >
                  {i.status === 'utilizado' ? 'já entrou' : 'pago'}
                </span>
              </div>
            ))}
          </div>
        )}
        <BuscaAdicionais eventoId={id} busca={busca} />
      </div>

      <div className="mt-6 flex flex-col gap-2 rounded-lg border border-border p-3 text-sm">
        <div className="flex items-center justify-between">
          <span className="font-medium text-foreground">Modo offline</span>
          <span className={online && !modoOffline ? 'text-green-600' : 'text-orange-600'}>{online && !modoOffline ? 'online' : 'sem internet'}</span>
        </div>
        <p className="text-xs text-muted-foreground">
          {pacoteInfo
            ? `Pacote salvo neste aparelho: ${pacoteInfo.total} ingresso(s), baixado em ${new Date(pacoteInfo.geradoEm).toLocaleString('pt-BR')}.`
            : 'Baixe o pacote ANTES do evento, com internet, para poder validar sem sinal. Não vale para evento com QR rotativo.'}
        </p>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={baixar} disabled={!online}>
            {pacoteInfo ? 'Atualizar pacote' : 'Baixar pacote offline'}
          </Button>
          <Button variant="outline" size="sm" onClick={sincronizar} disabled={!online || pendentes === 0}>
            Sincronizar ({pendentes})
          </Button>
        </div>
        {msgOffline && <p className="text-xs text-foreground">{msgOffline}</p>}
        {conflitos.length > 0 && (
          <div className="rounded border border-destructive p-2 text-xs text-destructive">
            <p className="font-medium">Entradas com problema (já tinham entrado antes ou inválidas):</p>
            {conflitos.map((c, i) => (
              <p key={i}>
                {c.nome} — {c.codigo}
              </p>
            ))}
          </div>
        )}
        <p className="text-xs text-muted-foreground">
          Atenção: dois aparelhos offline podem aceitar o mesmo ingresso; o conflito só aparece ao sincronizar.
        </p>
      </div>

      <Button variant="outline" className="mt-4 w-full" onClick={atualizarResumo}>
        Atualizar contador
      </Button>
    </main>
  )
}
