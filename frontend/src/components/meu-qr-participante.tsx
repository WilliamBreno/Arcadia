import { useQuery } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'

import { api } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { rotuloStatus, rotuloTipo, type StatusPalco } from '@/lib/palco'

type MeuQR = { ficha_id: number; tipo: string; personagem?: string; payload: string; status: StatusPalco }

// QR do participante para o camarim (item 8.3). Aparece na página do evento
// para quem tem inscrição aprovada como participante.
export function MeuQRParticipante({ slug }: { slug: string }) {
  const { usuario } = useAuth()
  const { data } = useQuery({
    queryKey: ['meu-qr-participante', slug, usuario?.id],
    enabled: !!usuario,
    queryFn: () => api<MeuQR[]>(`/eventos/${slug}/meu-qr-participante`).catch(() => [] as MeuQR[]),
  })
  if (!data || data.length === 0) return null
  return (
    <section id="meu-qr" aria-labelledby="titulo-meu-qr" className="mt-8 scroll-mt-20 rounded-xl border border-border p-4 sm:p-5">
      <h2 id="titulo-meu-qr" className="text-xl font-semibold text-foreground">
        Seu QR de participante
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">Mostre na entrada do camarim. É diferente do ingresso da plateia.</p>
      <div className="mt-4 flex flex-wrap gap-6">
        {data.map((q) => (
          <div key={q.ficha_id} className="flex flex-col items-center gap-2">
            <div className="rounded-lg bg-white p-3">
              <QRCodeSVG value={q.payload} size={180} level="M" aria-label="QR de participante" />
            </div>
            <p className="text-sm font-medium text-foreground">{q.personagem || rotuloTipo[q.tipo] || 'Apresentação'}</p>
            <p className="text-xs text-muted-foreground">{rotuloStatus[q.status]}</p>
          </div>
        ))}
      </div>
    </section>
  )
}
