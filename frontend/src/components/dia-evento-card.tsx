import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import { rotuloFuncao, type CredencialImpressa } from '@/lib/dia-evento'

type ResumoImportados = { total: number; entraram: number; origens: { origem: string; total: number; entraram: number }[] }

// Item 9.6: tudo do dia do evento num card: painel ao vivo, portões,
// ingressos de outras plataformas e credenciais impressas.
export function DiaDoEventoCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)

  const portoes = useQuery({
    queryKey: ['org-portoes', eventoId],
    queryFn: () => api<{ portoes: string[] }>(`/org/eventos/${eventoId}/portoes`),
  })
  const [textoPortoes, setTextoPortoes] = useState('')
  const carregou = useRef(false)
  useEffect(() => {
    if (portoes.data && !carregou.current) {
      carregou.current = true
      setTextoPortoes(portoes.data.portoes.join(', '))
    }
  }, [portoes.data])

  const importados = useQuery({
    queryKey: ['org-importados', eventoId],
    queryFn: () => api<ResumoImportados>(`/org/eventos/${eventoId}/importados`),
  })
  const credenciais = useQuery({
    queryKey: ['org-credenciais', eventoId],
    queryFn: () => api<{ credenciais: CredencialImpressa[] }>(`/org/eventos/${eventoId}/credenciais`),
  })

  const [origem, setOrigem] = useState('')
  const arquivo = useRef<HTMLInputElement>(null)
  const [enviando, setEnviando] = useState(false)
  const [nova, setNova] = useState({ nome: '', funcao: 'imprensa' })

  const executar = async (acao: () => Promise<unknown>, chave: string, msg?: string) => {
    setErro(null)
    setAviso(null)
    try {
      await acao()
      queryClient.invalidateQueries({ queryKey: [chave, eventoId] })
      if (msg) setAviso(msg)
      return true
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
      return false
    }
  }

  const importar = async (f?: File) => {
    if (!f) return
    setEnviando(true)
    setErro(null)
    setAviso(null)
    try {
      const csv = await f.text()
      const r = await api<{ importados: number; repetidos: number; sem_codigo: number }>(`/org/eventos/${eventoId}/importados`, {
        method: 'POST',
        body: { csv, origem },
      })
      setAviso(`${r.importados} ingresso(s) importado(s). ${r.repetidos} repetido(s) e ${r.sem_codigo} linha(s) sem código ignorados.`)
      queryClient.invalidateQueries({ queryKey: ['org-importados', eventoId] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível ler o arquivo.')
    } finally {
      setEnviando(false)
      if (arquivo.current) arquivo.current.value = ''
    }
  }

  const lista = credenciais.data?.credenciais ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle>Dia do evento</CardTitle>
        <p className="text-sm text-muted-foreground">Painel ao vivo, portões, ingressos vendidos em outras plataformas e credenciais impressas.</p>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <div className="flex flex-wrap gap-2">
          <Link to={`/ao-vivo/${eventoId}`} className={buttonVariants({ size: 'sm' })}>
            Abrir painel ao vivo
          </Link>
          <Link to={`/checkin/${eventoId}`} className={buttonVariants({ size: 'sm', variant: 'outline' })}>
            Abrir check-in
          </Link>
        </div>

        <section aria-labelledby="titulo-portoes-org" className="flex flex-col gap-2">
          <h3 id="titulo-portoes-org" className="text-sm font-semibold text-foreground">
            Portões
          </h3>
          <Label htmlFor="portoes-org" className="font-normal text-muted-foreground">
            Separe por vírgula. Cada aparelho do check-in escolhe o seu, e o painel mostra as entradas por portão.
          </Label>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Input id="portoes-org" placeholder="Principal, VIP, Imprensa" value={textoPortoes} onChange={(e) => setTextoPortoes(e.target.value)} />
            <Button
              type="button"
              variant="outline"
              onClick={() =>
                executar(
                  () => api(`/org/eventos/${eventoId}/portoes`, { method: 'PUT', body: { portoes: textoPortoes.split(',') } }),
                  'org-portoes',
                  'Portões salvos.',
                )
              }
            >
              Salvar portões
            </Button>
          </div>
        </section>

        <section aria-labelledby="titulo-importar" className="flex flex-col gap-2">
          <h3 id="titulo-importar" className="text-sm font-semibold text-foreground">
            Ingressos de outras plataformas
          </h3>
          <p className="text-sm text-muted-foreground">
            Vendeu também no Sympla, Even3 ou Eventbrite? Exporte a lista de participantes em CSV e importe aqui: o leitor do Evve passa a aceitar o
            QR original. Só a coluna do código é obrigatória (nome, e-mail e tipo ajudam a portaria). Esses ingressos só valem na entrada: não
            entram em vendas nem no repasse, e o Evve não manda e-mail para essas pessoas.
          </p>
          {importados.data && importados.data.total > 0 && (
            <p className="text-sm text-foreground">
              {importados.data.total} importado(s), {importados.data.entraram} já entraram
              {importados.data.origens.length > 0 && ` (${importados.data.origens.map((o) => `${o.origem || 'sem origem'}: ${o.total}`).join(', ')})`}.
            </p>
          )}
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
            <div className="flex flex-1 flex-col gap-1.5">
              <Label htmlFor="origem-importacao">Plataforma de origem</Label>
              <Input id="origem-importacao" placeholder="Sympla" value={origem} onChange={(e) => setOrigem(e.target.value)} />
            </div>
            <Button type="button" disabled={enviando} onClick={() => arquivo.current?.click()}>
              {enviando ? 'Importando…' : 'Escolher arquivo CSV'}
            </Button>
            <input ref={arquivo} type="file" accept=".csv,text/csv" className="sr-only" tabIndex={-1} aria-hidden="true" onChange={(e) => importar(e.target.files?.[0])} />
          </div>
          {importados.data && importados.data.total > 0 && (
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                size="sm"
                variant="ghost"
                onClick={() => {
                  if (window.confirm('Apagar os importados que ainda não entraram?')) {
                    executar(() => api(`/org/eventos/${eventoId}/importados`, { method: 'DELETE' }), 'org-importados', 'Importados que não entraram foram apagados.')
                  }
                }}
              >
                Apagar os que não entraram
              </Button>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                onClick={() => {
                  if (window.confirm('Apagar todos os importados, inclusive o registro de quem entrou?')) {
                    executar(() => api(`/org/eventos/${eventoId}/importados?todos=1`, { method: 'DELETE' }), 'org-importados', 'Todos os importados foram apagados.')
                  }
                }}
              >
                Apagar todos
              </Button>
            </div>
          )}
        </section>

        <section aria-labelledby="titulo-credenciais" className="flex flex-col gap-3">
          <h3 id="titulo-credenciais" className="text-sm font-semibold text-foreground">
            Credenciais impressas
          </h3>
          <p className="text-sm text-muted-foreground">
            Crachá com nome, função e QR. A equipe e os jurados entram sozinhos (e saem quando deixam o evento); expositores confirmados também. Imprensa
            e convidados você cadastra aqui. A credencial entra e sai à vontade no check-in.
          </p>
          <form
            className="flex flex-col gap-2 sm:flex-row sm:items-end"
            onSubmit={async (e) => {
              e.preventDefault()
              const ok = await executar(() => api(`/org/eventos/${eventoId}/credenciais`, { method: 'POST', body: nova }), 'org-credenciais')
              if (ok) setNova({ ...nova, nome: '' })
            }}
          >
            <div className="flex flex-1 flex-col gap-1.5">
              <Label htmlFor="cred-nome">Nome</Label>
              <Input id="cred-nome" value={nova.nome} onChange={(e) => setNova({ ...nova, nome: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="cred-funcao">Função</Label>
              <select
                id="cred-funcao"
                className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground"
                value={nova.funcao}
                onChange={(e) => setNova({ ...nova, funcao: e.target.value })}
              >
                <option value="imprensa">Imprensa</option>
                <option value="convidado">Convidado</option>
              </select>
            </div>
            <Button type="submit" variant="outline" disabled={!nova.nome.trim()}>
              Adicionar
            </Button>
          </form>
          {lista.length > 0 && (
            <ul className="flex flex-col divide-y divide-border rounded-lg border border-border">
              {lista.map((c) => (
                <li key={`${c.funcao}-${c.id}`} className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
                  <span className="min-w-0">
                    <span className="font-medium text-foreground">{c.nome || 'Sem nome'}</span>{' '}
                    <span className="text-muted-foreground">
                      · {rotuloFuncao[c.funcao]}
                      {c.detalhe ? ` · ${c.detalhe}` : ''}
                      {c.entrou_em ? ' · já entrou' : ''}
                    </span>
                  </span>
                  {c.manual && (
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => executar(() => api(`/org/eventos/${eventoId}/credenciais/${c.id}`, { method: 'DELETE' }), 'org-credenciais')}
                    >
                      Desativar
                    </Button>
                  )}
                </li>
              ))}
            </ul>
          )}
          <div className="flex flex-wrap gap-2">
            <Link to={`/organizador/eventos/${eventoId}/credenciais/imprimir`} className={buttonVariants({ size: 'sm', variant: 'outline' })}>
              Imprimir crachás (A4)
            </Link>
            <Link to={`/organizador/eventos/${eventoId}/credenciais/imprimir?formato=etiqueta`} className={buttonVariants({ size: 'sm', variant: 'outline' })}>
              Imprimir etiquetas (62 mm)
            </Link>
          </div>
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
