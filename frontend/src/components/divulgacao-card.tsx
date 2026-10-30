import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type Pixels = { meta: string; ga4: string; tiktok: string }

// Divulgação do evento (item 7.5): pixels de anúncio e widget de compra.
export function DivulgacaoCard({ eventoId, slug }: { eventoId: number; slug: string }) {
  const queryClient = useQueryClient()
  const { data: perfil } = useQuery({ queryKey: ['org-perfil'], queryFn: () => api<{ slug: string }>('/org/perfil') })
  const organizadorSlug = perfil?.slug
  const { data } = useQuery({ queryKey: ['org-pixels', eventoId], queryFn: () => api<Pixels>(`/org/eventos/${eventoId}/pixels`) })
  const [p, setP] = useState<Pixels>({ meta: '', ga4: '', tiktok: '' })
  const [salvando, setSalvando] = useState(false)
  const [msg, setMsg] = useState<{ ok: boolean; texto: string } | null>(null)
  const [copiado, setCopiado] = useState<string | null>(null)

  useEffect(() => {
    if (data) setP(data)
  }, [data])

  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null)
    setSalvando(true)
    try {
      const novo = await api<Pixels>(`/org/eventos/${eventoId}/pixels`, { method: 'PUT', body: p })
      setP(novo)
      queryClient.setQueryData(['org-pixels', eventoId], novo)
      setMsg({ ok: true, texto: 'Pixels salvos.' })
    } catch (err) {
      setMsg({ ok: false, texto: err instanceof ApiError ? err.message : 'Não foi possível salvar.' })
    } finally {
      setSalvando(false)
    }
  }

  const origem = window.location.origin
  const codigoWidget = `<iframe src="${origem}/widget/e/${slug}" title="Ingressos" style="width:100%;max-width:420px;height:360px;border:0" loading="lazy"></iframe>`
  const linkBio = organizadorSlug ? `${origem}/o/${organizadorSlug}` : ''

  const copiar = async (texto: string, qual: string) => {
    try {
      await navigator.clipboard.writeText(texto)
      setCopiado(qual)
      window.setTimeout(() => setCopiado(null), 2000)
    } catch {
      setCopiado(null)
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Divulgação</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <form onSubmit={salvar} className="flex flex-col gap-3">
          <div>
            <p className="font-medium text-foreground">Pixels de anúncio</p>
            <p className="text-sm text-muted-foreground">
              Meça seus anúncios: visualização da página, início da compra e compra concluída (valor e quantidade). Funcionam só
              nas páginas deste evento e só para quem aceitar os cookies. Nenhum dado pessoal do comprador é enviado. Na Meta, deixe desligada a "correspondência avançada automática" no Gerenciador de Eventos.
            </p>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="px-meta">Meta (Facebook/Instagram)</Label>
              <Input
                id="px-meta"
                inputMode="numeric"
                value={p.meta}
                onChange={(e) => setP({ ...p, meta: e.target.value })}
                placeholder="1234567890123456"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="px-ga4">Google Analytics 4</Label>
              <Input id="px-ga4" value={p.ga4} onChange={(e) => setP({ ...p, ga4: e.target.value })} placeholder="G-ABC123XYZ9" />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="px-tiktok">TikTok</Label>
              <Input
                id="px-tiktok"
                value={p.tiktok}
                onChange={(e) => setP({ ...p, tiktok: e.target.value })}
                placeholder="C4ABCDEFGHIJ12345678"
              />
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Button type="submit" disabled={salvando}>
              {salvando ? 'Salvando…' : 'Salvar pixels'}
            </Button>
            {msg && (
              <p role="status" className={`text-sm ${msg.ok ? 'text-foreground' : 'text-destructive'}`}>
                {msg.texto}
              </p>
            )}
          </div>
        </form>

        <div className="flex flex-col gap-2 border-t border-border pt-4">
          <p className="font-medium text-foreground">Widget de compra para o seu site</p>
          <p className="text-sm text-muted-foreground">
            Cole este código na página do seu site. Ele mostra os ingressos com o preço final; a compra abre no Evve.
          </p>
          <Textarea
            readOnly
            rows={3}
            value={codigoWidget}
            onFocus={(e) => e.currentTarget.select()}
            className="font-mono text-xs"
          />
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" size="sm" onClick={() => copiar(codigoWidget, 'widget')}>
              {copiado === 'widget' ? 'Código copiado' : 'Copiar código'}
            </Button>
            <a
              href={`/widget/e/${slug}`}
              target="_blank"
              rel="noopener"
              className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground"
            >
              Ver o widget
            </a>
          </div>
        </div>

        {linkBio && (
          <div className="flex flex-col gap-2 border-t border-border pt-4">
            <p className="font-medium text-foreground">Link para a bio</p>
            <p className="text-sm text-muted-foreground">
              A sua página no Evve, com todos os seus eventos. Ideal para a bio do Instagram e do TikTok.
            </p>
            <div className="flex flex-wrap items-center gap-2">
              <code className="break-all rounded-md bg-muted px-2 py-1 text-xs text-foreground">{linkBio}</code>
              <Button type="button" variant="outline" size="sm" onClick={() => copiar(linkBio, 'bio')}>
                {copiado === 'bio' ? 'Link copiado' : 'Copiar link'}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
