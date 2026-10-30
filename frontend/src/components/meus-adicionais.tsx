import { useQuery } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import type { Grupo, MeuAdicional } from '@/lib/extras'

// Item 10.3: adicionais para retirar no evento e grupos que a pessoa criou.
export function MeusAdicionais() {
  const { data } = useQuery({ queryKey: ['meus-adicionais'], queryFn: () => api<MeuAdicional[]>('/me/adicionais') })
  const grupos = useQuery({ queryKey: ['meus-grupos'], queryFn: () => api<Grupo[]>('/me/grupos') })
  const [aberto, setAberto] = useState<number | null>(null)

  return (
    <>
      {data && data.length > 0 && (
        <section aria-labelledby="titulo-meus-adicionais" className="mt-10">
          <h2 id="titulo-meus-adicionais" className="mb-1 text-lg font-semibold text-foreground">
            Adicionais para retirar
          </h2>
          <p className="mb-3 text-sm text-muted-foreground">Mostre o código na retirada, no evento.</p>
          <ul className="flex flex-col gap-3">
            {data.map((a) => (
              <li key={a.id} className="rounded-xl border border-border p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="font-medium text-foreground">{a.nome}</p>
                    <p className="text-sm text-muted-foreground">
                      {a.evento_titulo} · <span className="font-mono">{a.codigo}</span>
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {a.status === 'utilizado' ? `Entregue${a.entregue_em ? ` em ${new Date(a.entregue_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })}` : ''}` : 'Para retirar'}
                    </p>
                  </div>
                  {a.status === 'pago' && (
                    <Button size="sm" variant="outline" aria-expanded={aberto === a.id} onClick={() => setAberto(aberto === a.id ? null : a.id)}>
                      {aberto === a.id ? 'Esconder' : 'Mostrar código'}
                    </Button>
                  )}
                </div>
                {aberto === a.id && (
                  <div className="mt-3 flex justify-center rounded-lg bg-white p-3">
                    <QRCodeSVG value={a.qr} size={180} aria-label={`QR para retirar ${a.nome}`} />
                  </div>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}
      {grupos.data && grupos.data.length > 0 && (
        <section aria-labelledby="titulo-meus-grupos" className="mt-10">
          <h2 id="titulo-meus-grupos" className="mb-3 text-lg font-semibold text-foreground">
            Grupos que você criou
          </h2>
          <ul className="flex flex-col gap-2">
            {grupos.data.map((g) => (
              <li key={g.token}>
                <Link to={`/grupo/${g.token}`} className="flex justify-between gap-3 rounded-lg border border-border px-4 py-3 text-sm hover:bg-muted">
                  <span className="min-w-0 truncate text-foreground">{g.evento_titulo}</span>
                  <span className="shrink-0 text-muted-foreground">
                    {g.pagas}/{g.quantidade} pagas · {g.status === 'aberto' ? 'aberto' : 'fechado'}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}
    </>
  )
}
