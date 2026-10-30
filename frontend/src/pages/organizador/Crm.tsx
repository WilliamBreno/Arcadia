import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { VoltarLink } from '@/components/voltar-link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api, ApiError, baixarArquivo } from '@/lib/api'
import type { Evento } from '@/lib/evento'

type Resumo = {
  contatos: number
  enviados_mes: number
  cota_mes: number
  cota_restante: number
  campanhas_hoje: number
  campanhas_por_dia: number
  campanhas: { assunto: string; segmento: string; evento_titulo: string; destinatarios: number; enviada_em: string }[]
}

const rotuloSegmento: Record<string, string> = { todos: 'Todos os contatos', compraram: 'Compraram no evento', foram: 'Foram ao evento (check-in)' }
const select = 'h-8 w-full rounded-lg border border-border bg-background px-2.5 text-sm'

// CRM do organizador (item 9.3): e-mail para quem segue o organizador,
// com segmentos por evento, prévia, teste e exportação para Mailchimp/Brevo.
export default function Crm() {
  const queryClient = useQueryClient()
  const { data: r } = useQuery({ queryKey: ['org-crm'], queryFn: () => api<Resumo>('/org/crm') })
  const { data: eventos } = useQuery({ queryKey: ['org-eventos'], queryFn: () => api<Evento[]>('/org/eventos') })
  const [segmento, setSegmento] = useState('todos')
  const [eventoId, setEventoId] = useState('')
  const [assunto, setAssunto] = useState('')
  const [corpo, setCorpo] = useState('')
  const [msg, setMsg] = useState<{ ok: boolean; texto: string } | null>(null)
  const [enviando, setEnviando] = useState(false)

  const precisaEvento = segmento !== 'todos'
  const eventoEscolhido = precisaEvento ? eventoId || String(eventos?.[0]?.id ?? '') : ''
  const { data: previa } = useQuery({
    queryKey: ['org-crm-previa', segmento, eventoEscolhido],
    enabled: !precisaEvento || !!eventoEscolhido,
    queryFn: () =>
      api<{ destinatarios: number }>(`/org/crm/previa?segmento=${segmento}${eventoEscolhido ? `&evento_id=${eventoEscolhido}` : ''}`),
  })

  const enviar = async (teste: boolean) => {
    if (!teste && !confirm(`Enviar "${assunto}" para ${previa?.destinatarios ?? 0} pessoa(s)? Não dá para desfazer.`)) return
    setMsg(null)
    setEnviando(true)
    try {
      const res = await api<{ destinatarios: number }>('/org/crm/campanhas', {
        method: 'POST',
        body: { assunto, corpo, segmento, evento_id: eventoEscolhido ? Number(eventoEscolhido) : null, teste },
      })
      setMsg({ ok: true, texto: teste ? 'Teste enviado para o seu e-mail.' : `Campanha enviada para ${res.destinatarios} pessoa(s).` })
      if (!teste) {
        setAssunto('')
        setCorpo('')
        queryClient.invalidateQueries({ queryKey: ['org-crm'] })
      }
    } catch (e) {
      setMsg({ ok: false, texto: e instanceof ApiError ? e.message : 'Não foi possível enviar agora.' })
    } finally {
      setEnviando(false)
    }
  }

  const podeEnviar = assunto.trim() && corpo.trim() && !enviando
  const limiteDia = r ? r.campanhas_hoje >= r.campanhas_por_dia : false

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar ao painel</VoltarLink>
      <h1 className="text-2xl font-semibold text-foreground">Contatos e e-mail</h1>
      <p className="mt-2 text-sm text-muted-foreground text-pretty">
        Sua base são as pessoas que seguem você na Evve, inclusive quem marcou "quero receber novidades" na compra. Só elas recebem, e todo e-mail
        tem o link para deixar de seguir.
      </p>

      {r && (
        <dl className="mt-6 grid grid-cols-3 gap-3">
          {[
            ['Contatos', String(r.contatos)],
            ['Enviados no mês', `${r.enviados_mes} de ${r.cota_mes}`],
            ['Campanhas hoje', `${r.campanhas_hoje} de ${r.campanhas_por_dia}`],
          ].map(([t, v]) => (
            <div key={t} className="rounded-xl border border-border p-3">
              <dt className="text-xs text-muted-foreground">{t}</dt>
              <dd className="mt-1 text-lg font-semibold tabular-nums text-foreground">{v}</dd>
            </div>
          ))}
        </dl>
      )}

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>Nova campanha</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="seg">Para quem</Label>
              <select id="seg" className={select} value={segmento} onChange={(e) => setSegmento(e.target.value)}>
                {Object.entries(rotuloSegmento).map(([v, t]) => (
                  <option key={v} value={v}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
            {precisaEvento && (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="seg-evento">Evento</Label>
                <select id="seg-evento" className={select} value={eventoEscolhido} onChange={(e) => setEventoId(e.target.value)}>
                  {eventos?.map((e) => (
                    <option key={e.id} value={e.id}>
                      {e.titulo}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>
          <p className="text-sm text-foreground" aria-live="polite">
            {previa ? `${previa.destinatarios} pessoa(s) recebem.` : 'Calculando…'}
          </p>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="assunto">Assunto</Label>
            <Input id="assunto" maxLength={120} value={assunto} onChange={(e) => setAssunto(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="corpo">Mensagem</Label>
            <Textarea id="corpo" rows={8} maxLength={5000} value={corpo} onChange={(e) => setCorpo(e.target.value)} />
            <p className="text-xs text-muted-foreground">
              Texto simples. Deixe uma linha em branco entre parágrafos; links que começam com https:// ficam clicáveis. A Evve põe "Olá, (nome)!" no
              começo e o link para deixar de seguir no fim.
            </p>
          </div>
          {msg && (
            <p role={msg.ok ? 'status' : 'alert'} className={`text-sm ${msg.ok ? 'text-foreground' : 'text-destructive'}`}>
              {msg.texto}
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" disabled={!podeEnviar} onClick={() => enviar(true)}>
              Enviar teste para mim
            </Button>
            <Button disabled={!podeEnviar || limiteDia || !previa?.destinatarios} onClick={() => enviar(false)}>
              Enviar para {previa?.destinatarios ?? 0} pessoa(s)
            </Button>
          </div>
          {limiteDia && <p className="text-xs text-muted-foreground">Você já mandou {r?.campanhas_por_dia} campanhas hoje. Amanhã libera de novo.</p>}
        </CardContent>
      </Card>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>Levar para o Mailchimp ou Brevo</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-sm text-muted-foreground text-pretty">
            Baixe os contatos (e-mail, nome e os eventos em que cada um comprou, como tags) num arquivo que o Mailchimp e o Brevo importam direto. Só
            vêm pessoas que aceitaram receber novidades suas. Lá, mantenha o descadastro funcionando.
          </p>
          <div>
            <Button variant="outline" onClick={() => baixarArquivo('/org/crm/contatos.csv', 'contatos-evve.csv')}>
              Baixar contatos (CSV)
            </Button>
          </div>
        </CardContent>
      </Card>

      {r && r.campanhas.length > 0 && (
        <section aria-labelledby="t-historico" className="mt-6">
          <h2 id="t-historico" className="text-lg font-semibold text-foreground">
            Enviadas
          </h2>
          <ul className="mt-2 flex flex-col divide-y divide-border rounded-xl border border-border">
            {r.campanhas.map((c, i) => (
              <li key={i} className="flex flex-wrap items-baseline justify-between gap-2 px-4 py-2 text-sm">
                <span className="min-w-0 text-foreground">
                  {c.assunto}{' '}
                  <span className="text-xs text-muted-foreground">
                    · {rotuloSegmento[c.segmento]}
                    {c.evento_titulo && `: ${c.evento_titulo}`}
                  </span>
                </span>
                <span className="text-xs text-muted-foreground">
                  {c.destinatarios} pessoa(s) · {new Date(c.enviada_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}
    </main>
  )
}
