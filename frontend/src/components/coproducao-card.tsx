import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import { formatarBP, lerPercentual, type ResumoCoproducao } from '@/lib/coproducao'

// Item 9.8: o organizador principal divide o repasse do evento com outros
// organizadores. Só o dono vê este card (a API devolve 403 para a equipe).
export function CoproducaoCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const { data, error } = useQuery({
    queryKey: ['org-coproducao', eventoId],
    retry: false,
    queryFn: () => api<ResumoCoproducao>(`/org/eventos/${eventoId}/coproducao`),
  })
  const [email, setEmail] = useState('')
  const [pct, setPct] = useState('')
  const [editando, setEditando] = useState<{ id: number; pct: string } | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)

  if (error instanceof ApiError && error.status === 403) return null
  if (!data) return null

  const executar = async (acao: () => Promise<unknown>, msg: string) => {
    setErro(null)
    setAviso(null)
    try {
      await acao()
      setAviso(msg)
      queryClient.invalidateQueries({ queryKey: ['org-coproducao', eventoId] })
      queryClient.invalidateQueries({ queryKey: ['org-evento-financeiro', eventoId] })
      return true
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
      return false
    }
  }

  const convidar = async () => {
    const bp = lerPercentual(pct)
    if (bp === null) {
      setErro('Informe o percentual (ex.: 30).')
      return
    }
    const ok = await executar(
      () => api(`/org/eventos/${eventoId}/coproducao`, { method: 'POST', body: { email, percentual_bp: bp } }),
      'Convite enviado. A divisão vale quando a pessoa aceitar.',
    )
    if (ok) {
      setEmail('')
      setPct('')
    }
  }

  const propor = (id: number, bp: number) =>
    executar(
      () => api(`/org/eventos/${eventoId}/coproducao/${id}`, { method: 'PUT', body: { percentual_bp: bp } }),
      bp === 0 && !data.tem_venda ? 'Coprodutor removido.' : 'Mudança enviada. Vale quando o coprodutor aceitar.',
    )

  const ativos = data.coprodutores.filter((c) => c.status !== 'recusada')

  return (
    <Card>
      <CardHeader>
        <CardTitle>Coprodução</CardTitle>
        <p className="text-sm text-muted-foreground">
          Divida o repasse com outros organizadores: cada um recebe a sua parte no próprio Pix, calculada sobre o líquido (depois de taxas e
          reembolsos). A pessoa precisa ter perfil de organizador no Evve e aceitar. Depois da primeira venda, qualquer mudança precisa do aceite do
          coprodutor; quando o repasse sai, a divisão não muda mais.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-col gap-2 rounded-lg border border-border p-3 text-sm">
          <div className="flex justify-between">
            <span className="text-foreground">Você (principal)</span>
            <span className="font-medium tabular-nums text-foreground">{formatarBP(data.principal_bp)}</span>
          </div>
          {ativos.map((c) => (
            <div key={c.id} className="flex flex-col gap-2 border-t border-border pt-2">
              <div className="flex flex-wrap justify-between gap-2">
                <span className="text-foreground">
                  {c.organizador_nome}
                  {c.status === 'convidado' && <span className="text-muted-foreground"> · convite enviado ({formatarBP(c.proposto_bp ?? 0)})</span>}
                  {c.status === 'ativa' && c.proposto_bp !== null && (
                    <span className="text-muted-foreground"> · aguardando aceite de {formatarBP(c.proposto_bp)}</span>
                  )}
                </span>
                <span className="font-medium tabular-nums text-foreground">{c.status === 'ativa' ? formatarBP(c.percentual_bp) : '—'}</span>
              </div>
              {!data.travada &&
                (editando?.id === c.id ? (
                  <div className="flex flex-wrap items-end gap-2">
                    <div className="flex flex-col gap-1">
                      <Label htmlFor={`pct-${c.id}`}>Novo percentual</Label>
                      <Input id={`pct-${c.id}`} className="w-28" inputMode="decimal" value={editando.pct} onChange={(e) => setEditando({ ...editando, pct: e.target.value })} />
                    </div>
                    <Button
                      size="sm"
                      onClick={async () => {
                        const bp = lerPercentual(editando.pct)
                        if (bp !== null && (await propor(c.id, bp))) setEditando(null)
                      }}
                    >
                      Propor
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => setEditando(null)}>
                      Cancelar
                    </Button>
                  </div>
                ) : (
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => setEditando({ id: c.id, pct: String((c.proposto_bp ?? c.percentual_bp) / 100).replace('.', ',') })}>
                      Mudar percentual
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => {
                        const msg = data.tem_venda && c.status === 'ativa' ? 'O evento já tem vendas: o coprodutor precisa aceitar sair. Propor 0%?' : `Tirar ${c.organizador_nome} da coprodução?`
                        if (window.confirm(msg)) propor(c.id, 0)
                      }}
                    >
                      Tirar
                    </Button>
                  </div>
                ))}
            </div>
          ))}
        </div>
        {data.coprodutores
          .filter((c) => c.status === 'recusada')
          .map((c) => (
            <p key={c.id} className="text-sm text-muted-foreground">
              {c.organizador_nome} recusou o convite.
            </p>
          ))}

        {data.travada ? (
          <p className="text-sm text-muted-foreground">O repasse deste evento já foi gerado: a divisão está fechada.</p>
        ) : (
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
            <div className="flex flex-1 flex-col gap-1.5">
              <Label htmlFor="copro-email">E-mail do coprodutor</Label>
              <Input id="copro-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="copro-pct">Percentual</Label>
              <Input id="copro-pct" className="sm:w-28" inputMode="decimal" placeholder="30" value={pct} onChange={(e) => setPct(e.target.value)} />
            </div>
            <Button onClick={convidar} disabled={!email || !pct}>
              Convidar
            </Button>
          </div>
        )}
        {aviso && (
          <p role="status" className="text-sm text-foreground">
            {aviso}
          </p>
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
