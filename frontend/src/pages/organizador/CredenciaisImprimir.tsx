import { useQuery } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { useParams, useSearchParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { Button } from '@/components/ui/button'
import { VoltarBotao } from '@/components/voltar-link'
import { api } from '@/lib/api'
import { rotuloFuncao, type CredencialImpressa } from '@/lib/dia-evento'
import type { Evento } from '@/lib/evento'

// Item 9.6: crachás para imprimir pelo navegador. A4 = 8 por folha
// (2 × 4, 9 × 6,5 cm); etiqueta = uma por página, 62 × 40 mm (térmica).
export default function CredenciaisImprimir() {
  const id = Number(useParams().id)
  const [params] = useSearchParams()
  const etiqueta = params.get('formato') === 'etiqueta'
  const { data: evento } = useQuery({ queryKey: ['org-evento', id], queryFn: () => api<Evento>(`/org/eventos/${id}`) })
  const { data, isLoading } = useQuery({
    queryKey: ['org-credenciais', id],
    queryFn: () => api<{ credenciais: CredencialImpressa[] }>(`/org/eventos/${id}/credenciais`),
  })
  if (isLoading) return <Carregando />
  const lista = data?.credenciais ?? []

  return (
    <main className="mx-auto max-w-4xl px-4 py-8 print:m-0 print:max-w-none print:p-0">
      <style>
        {(etiqueta ? '@page { size: 62mm 40mm; margin: 0 }' : '@page { size: A4; margin: 10mm }') +
          // Na impressão só os crachás aparecem (sem cabeçalho, rodapé e avisos do site).
          ' @media print { body * { visibility: hidden } #area-impressao, #area-impressao * { visibility: visible } #area-impressao { position: absolute; left: 0; top: 0; width: 100% } }'}
      </style>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3 print:hidden">
        <div>
          <VoltarBotao fallback={`/organizador/eventos/${id}`} />
          <h1 className="text-xl font-semibold text-foreground">{etiqueta ? 'Etiquetas 62 mm' : 'Crachás A4'}</h1>
          <p className="text-sm text-muted-foreground">
            {lista.length} credencial(is). {etiqueta ? 'Use a impressora térmica com etiqueta de 62 × 40 mm.' : '8 por folha; corte nas linhas.'} Na janela de
            impressão, deixe a escala em 100%.
          </p>
        </div>
        <Button onClick={() => window.print()} disabled={lista.length === 0}>
          Imprimir
        </Button>
      </div>
      {lista.length === 0 && <p className="text-muted-foreground">Nenhuma credencial ainda. Adicione equipe, jurados, imprensa ou convidados.</p>}

      {etiqueta ? (
        <div id="area-impressao" className="flex flex-col gap-4 print:gap-0">
          {lista.map((c) => (
            <div
              key={`${c.funcao}-${c.id}`}
              className="flex items-center overflow-hidden border border-dashed border-border bg-white text-black print:break-after-page print:border-0"
              style={{ width: '62mm', height: '40mm', gap: '3mm', padding: '2mm' }}
            >
              <QRCodeSVG value={c.qr} size={110} level="M" aria-label={`QR de ${c.nome}`} />
              <div className="min-w-0">
                <p className="font-bold uppercase leading-tight" style={{ fontSize: '9pt' }}>{rotuloFuncao[c.funcao]}</p>
                <p className="mt-1 line-clamp-3 font-semibold leading-tight" style={{ fontSize: '11pt' }}>{c.nome || 'Sem nome'}</p>
                {c.detalhe && <p className="mt-1 line-clamp-2 leading-tight" style={{ fontSize: '8pt' }}>{c.detalhe}</p>}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div id="area-impressao" className="grid grid-cols-1 gap-4 sm:grid-cols-2 print:grid-cols-2 print:gap-0">
          {lista.map((c) => (
            <div
              key={`${c.funcao}-${c.id}`}
              className="flex flex-col items-center justify-center gap-2 border border-dashed border-gray-400 bg-white p-4 text-center text-black print:break-inside-avoid"
              style={{ minHeight: '65mm' }}
            >
              <p className="text-xs font-medium uppercase tracking-wide">{evento?.titulo}</p>
              <p className="rounded bg-black px-3 py-0.5 text-sm font-bold uppercase text-white">{rotuloFuncao[c.funcao]}</p>
              <p className="text-lg font-semibold leading-tight">{c.nome || 'Sem nome'}</p>
              {c.detalhe && <p className="text-xs">{c.detalhe}</p>}
              <QRCodeSVG value={c.qr} size={120} level="M" aria-label={`QR de ${c.nome}`} />
            </div>
          ))}
        </div>
      )}
    </main>
  )
}
