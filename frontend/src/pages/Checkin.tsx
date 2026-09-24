import { Html5Qrcode } from 'html5-qrcode'
import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'

import { ApiError } from '@/lib/api'
import { buscarCheckin, resumoCheckin, validarCheckin, type ItemBusca, type ResumoCheckin, type ValidarCheckinResposta } from '@/lib/checkin'
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
}

const tituloPorResultado: Record<ValidarCheckinResposta['resultado'], string> = {
  valido: '✅ Entrada liberada',
  ja_utilizado: '⚠️ Já utilizado',
  cancelado: '❌ Ingresso cancelado',
  outro_evento: '❌ Ingresso de outro evento',
  nao_encontrado: '❌ Código não encontrado',
  qr_expirado: '⏱️ QR desatualizado — peça para abrir o ingresso de novo',
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

  const leitorRef = useRef<Html5Qrcode | null>(null)
  const processandoRef = useRef(false)

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
      })
      .catch((e) => {
        setAcessoLiberado(false)
        setErroAcesso(e instanceof ApiError ? e.message : 'Erro ao verificar acesso')
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
    const separador = texto.indexOf(':')
    if (separador === -1) return

    processandoRef.current = true
    setProcessando(true)
    leitorRef.current?.pause(true)

    const codigo = texto.slice(0, separador)
    const qrToken = texto.slice(separador + 1)

    try {
      const resp = await validarCheckin(id, codigo, qrToken)
      setResultado(resp)
      if (resp.resultado === 'valido') atualizarResumo()
    } catch {
      setResultado({ resultado: 'nao_encontrado' })
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
      <h1 className="mb-4 text-xl font-semibold text-foreground">Check-in</h1>

      <div className="mb-4 flex items-center justify-between rounded-lg border border-border px-4 py-2">
        <span className="text-sm text-muted-foreground">Entradas</span>
        <span className="text-lg font-semibold text-foreground">
          {resumo ? `${resumo.total_utilizados} / ${resumo.total_pagos}` : '—'}
        </span>
      </div>

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
      </div>

      <Button variant="outline" className="mt-6 w-full" onClick={atualizarResumo}>
        Atualizar contador
      </Button>
    </main>
  )
}
