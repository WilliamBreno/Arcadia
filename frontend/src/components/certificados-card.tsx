import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api, ApiError } from '@/lib/api'
import { rotuloFuncaoCert, type CertificadoPublico, type ConfigCertificado, type ResultadoPesquisa } from '@/lib/pos-evento'

type Resumo = { config: ConfigCertificado; resumo: { total: number; por_funcao: Record<string, number>; evento_terminou: boolean; lista: CertificadoPublico[] } }

// Item 9.7: certificados automáticos e resultado da pesquisa pós-evento.
export function CertificadosCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['org-certificados', eventoId], queryFn: () => api<Resumo>(`/org/eventos/${eventoId}/certificados`) })
  const pesquisa = useQuery({ queryKey: ['org-pesquisa', eventoId], queryFn: () => api<ResultadoPesquisa>(`/org/eventos/${eventoId}/pesquisa`) })
  const [cfg, setCfg] = useState<ConfigCertificado | null>(null)
  const [horas, setHoras] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)
  const [verLista, setVerLista] = useState(false)

  useEffect(() => {
    if (data && !cfg) {
      setCfg(data.config)
      setHoras(data.config.carga_horaria_min ? String(data.config.carga_horaria_min / 60).replace('.', ',') : '')
    }
  }, [data, cfg])

  if (!cfg || !data) return null

  const salvar = async () => {
    setErro(null)
    setAviso(null)
    const h = Number(horas.replace(',', '.') || '0')
    if (Number.isNaN(h) || h < 0) {
      setErro('Carga horária inválida.')
      return
    }
    try {
      await api(`/org/eventos/${eventoId}/certificados/config`, { method: 'PUT', body: { ...cfg, carga_horaria_min: Math.round(h * 60) } })
      setAviso('Certificados salvos.')
      queryClient.invalidateQueries({ queryKey: ['org-certificados', eventoId] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
    }
  }

  const emitir = async () => {
    setErro(null)
    setAviso(null)
    try {
      const r = await api<{ emitidos: number }>(`/org/eventos/${eventoId}/certificados/emitir`, { method: 'POST' })
      setAviso(r.emitidos > 0 ? `${r.emitidos} certificado(s) novo(s).` : 'Nenhum certificado novo: todos já foram emitidos.')
      queryClient.invalidateQueries({ queryKey: ['org-certificados', eventoId] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível emitir.')
    }
  }

  const grupo = (chave: 'publico' | 'participantes' | 'jurados' | 'equipe', rotulo: string) => (
    <label className="flex items-center gap-2 text-sm text-foreground">
      <input type="checkbox" checked={cfg[chave]} onChange={(e) => setCfg({ ...cfg, [chave]: e.target.checked })} />
      {rotulo}
    </label>
  )
  const p = pesquisa.data

  return (
    <Card>
      <CardHeader>
        <CardTitle>Certificados e pesquisa</CardTitle>
        <p className="text-sm text-muted-foreground">
          No dia seguinte ao evento, quem participou recebe um e-mail com a pesquisa (nota de 0 a 10, anônima) e, se você ligar, o certificado em PDF
          com código de validação.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <section aria-labelledby="titulo-cert-config" className="flex flex-col gap-3">
          <h3 id="titulo-cert-config" className="text-sm font-semibold text-foreground">
            Certificados
          </h3>
          <label className="flex items-center gap-2 text-sm font-medium text-foreground">
            <input type="checkbox" checked={cfg.ativo} onChange={(e) => setCfg({ ...cfg, ativo: e.target.checked })} />
            Emitir certificados neste evento
          </label>
          {cfg.ativo && (
            <>
              <fieldset className="flex flex-col gap-2">
                <legend className="mb-1 text-sm text-muted-foreground">Quem recebe (só quem esteve lá)</legend>
                {grupo('publico', 'Público que fez check-in (no nome do ingresso)')}
                {grupo('participantes', 'Participantes do concurso aprovados (com a colocação)')}
                {grupo('jurados', 'Jurados')}
                {grupo('equipe', 'Equipe de check-in')}
              </fieldset>
              <div className="grid gap-3 sm:grid-cols-3">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="cert-horas">Carga horária (horas)</Label>
                  <Input id="cert-horas" inputMode="decimal" placeholder="4" value={horas} onChange={(e) => setHoras(e.target.value)} />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="cert-assina">Quem assina</Label>
                  <Input id="cert-assina" placeholder="Nome" value={cfg.assinatura_nome} onChange={(e) => setCfg({ ...cfg, assinatura_nome: e.target.value })} />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="cert-cargo">Cargo</Label>
                  <Input id="cert-cargo" placeholder="Coordenação" value={cfg.assinatura_cargo} onChange={(e) => setCfg({ ...cfg, assinatura_cargo: e.target.value })} />
                </div>
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="cert-texto">Texto extra (opcional)</Label>
                <Textarea id="cert-texto" rows={2} maxLength={400} placeholder="Ex.: conteúdo da oficina" value={cfg.texto} onChange={(e) => setCfg({ ...cfg, texto: e.target.value })} />
                <p className="text-xs text-muted-foreground">Sem assinatura, sai o nome público do organizador.</p>
              </div>
            </>
          )}
          <div className="flex flex-wrap gap-2">
            <Button onClick={salvar}>Salvar</Button>
            {data.config.ativo && data.resumo.evento_terminou && (
              <Button variant="outline" onClick={emitir}>
                Emitir agora
              </Button>
            )}
          </div>
          {data.resumo.total > 0 && (
            <div className="text-sm text-foreground">
              <p>
                {data.resumo.total} certificado(s) emitido(s):{' '}
                {Object.entries(data.resumo.por_funcao)
                  .map(([f, n]) => `${n} ${rotuloFuncaoCert[f as CertificadoPublico['funcao']].toLowerCase()}`)
                  .join(', ')}
                .
              </p>
              <Button variant="ghost" size="sm" className="mt-1 px-0" onClick={() => setVerLista(!verLista)} aria-expanded={verLista}>
                {verLista ? 'Esconder a lista' : 'Ver a lista'}
              </Button>
              {verLista && (
                <ul className="mt-2 flex max-h-72 flex-col divide-y divide-border overflow-y-auto rounded-lg border border-border">
                  {data.resumo.lista.map((c) => (
                    <li key={c.codigo} className="flex justify-between gap-2 px-3 py-2">
                      <span className="min-w-0 truncate">
                        {c.nome} <span className="text-muted-foreground">· {rotuloFuncaoCert[c.funcao]}</span>
                      </span>
                      <Link to={`/certificado/${c.codigo}`} className="shrink-0 font-mono text-xs underline underline-offset-4">
                        {c.codigo}
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </section>

        <section aria-labelledby="titulo-pesquisa-org" className="flex flex-col gap-3">
          <h3 id="titulo-pesquisa-org" className="text-sm font-semibold text-foreground">
            Pesquisa pós-evento
          </h3>
          {!p || p.respostas < p.minimo ? (
            <p className="text-sm text-muted-foreground">
              {p?.respostas ? `${p.respostas} resposta(s) até agora. ` : ''}O resultado aparece a partir de {p?.minimo ?? 3} respostas, para ninguém ser
              reconhecido pela nota.
            </p>
          ) : (
            <>
              <div className="flex flex-wrap items-end gap-6">
                <div>
                  <p className="text-sm text-muted-foreground">NPS</p>
                  <p className="text-4xl font-semibold tabular-nums text-foreground">{p.nps}</p>
                </div>
                <p className="text-sm text-muted-foreground">
                  {p.respostas} respostas · {p.promotores} recomendariam (9–10), {p.neutros} neutras (7–8), {p.detratores} não recomendariam (0–6)
                </p>
              </div>
              <div aria-hidden="true" className="flex h-24 items-end gap-1 border-b border-border">
                {p.distribuicao.map((n, nota) => {
                  const max = Math.max(...p.distribuicao, 1)
                  return (
                    <div key={nota} className="flex h-full flex-1 flex-col items-center justify-end gap-1" title={`Nota ${nota}: ${n}`}>
                      <div className="w-full rounded-t-sm bg-primary" style={{ height: n ? `${Math.max((n / max) * 100, 4)}%` : '0%' }} />
                    </div>
                  )
                })}
              </div>
              <div aria-hidden="true" className="-mt-2 flex gap-1 text-center text-xs text-muted-foreground tabular-nums">
                {p.distribuicao.map((_, nota) => (
                  <span key={nota} className="flex-1">
                    {nota}
                  </span>
                ))}
              </div>
              <table className="sr-only">
                <caption>Respostas por nota</caption>
                <tbody>
                  {p.distribuicao.map((n, nota) => (
                    <tr key={nota}>
                      <th>Nota {nota}</th>
                      <td>{n}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {p.comentarios.length > 0 && (
                <ul className="flex max-h-80 flex-col gap-2 overflow-y-auto">
                  {p.comentarios.map((c, i) => (
                    <li key={i} className="rounded-lg bg-muted px-3 py-2 text-sm">
                      <span className="font-medium tabular-nums text-foreground">{c.nota}</span>
                      <span className="text-muted-foreground"> · {c.data}</span>
                      <p className="mt-1 text-foreground">{c.comentario}</p>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
        </section>

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
