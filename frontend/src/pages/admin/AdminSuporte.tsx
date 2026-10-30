import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { AdminNav } from '@/components/admin-nav'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api, ApiError } from '@/lib/api'
import type { StatusPlataforma } from '@/lib/ajuda'

type Contato = { id: number; nome: string; email: string; assunto_rotulo: string; pedido: string; mensagem: string; status: string; criado_em: string; com_conta: boolean }

// Item 9.9: avisos da página de status e mensagens da central de ajuda.
export default function AdminSuporte() {
  const queryClient = useQueryClient()
  const [filtro, setFiltro] = useState('aberto')
  const contatos = useQuery({ queryKey: ['admin-suporte', filtro], queryFn: () => api<Contato[]>(`/admin/suporte${filtro ? `?status=${filtro}` : ''}`) })
  const status = useQuery({ queryKey: ['status-plataforma'], queryFn: () => api<StatusPlataforma>('/status', { semAuth: true }) })
  const [aviso, setAviso] = useState({ titulo: '', mensagem: '', nivel: 'aviso', componente: 'geral' })
  const [erro, setErro] = useState<string | null>(null)

  const executar = async (acao: () => Promise<unknown>) => {
    setErro(null)
    try {
      await acao()
      queryClient.invalidateQueries({ queryKey: ['admin-suporte'] })
      queryClient.invalidateQueries({ queryKey: ['status-plataforma'] })
      queryClient.invalidateQueries({ queryKey: ['avisos-status'] })
      return true
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro')
      return false
    }
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <AdminNav />
      <h1 className="mb-6 text-2xl font-semibold text-foreground">Suporte e status</h1>

      <section aria-labelledby="titulo-avisos-admin" className="flex flex-col gap-3 rounded-xl border border-border p-4">
        <h2 id="titulo-avisos-admin" className="text-lg font-semibold text-foreground">
          Avisos de incidente
        </h2>
        <p className="text-sm text-muted-foreground">Aparecem em /status e numa faixa no topo do site até você encerrar.</p>
        {status.data?.avisos.map((a) => (
          <div key={a.id} className="flex items-center justify-between gap-3 rounded-lg bg-muted px-3 py-2 text-sm">
            <span className="text-foreground">
              <strong>{a.titulo}</strong> · {a.nivel} · {a.componente}
            </span>
            <Button size="sm" variant="outline" onClick={() => executar(() => api(`/admin/avisos-status/${a.id}/encerrar`, { method: 'POST' }))}>
              Encerrar
            </Button>
          </div>
        ))}
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="flex flex-col gap-1.5 sm:col-span-3">
            <Label htmlFor="aviso-titulo">Título</Label>
            <Input id="aviso-titulo" maxLength={120} value={aviso.titulo} onChange={(e) => setAviso({ ...aviso, titulo: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="aviso-nivel">Nível</Label>
            <select id="aviso-nivel" className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground" value={aviso.nivel} onChange={(e) => setAviso({ ...aviso, nivel: e.target.value })}>
              <option value="aviso">Aviso (instável)</option>
              <option value="falha">Falha (fora do ar)</option>
              <option value="manutencao">Manutenção</option>
            </select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="aviso-comp">Parte afetada</Label>
            <select id="aviso-comp" className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground" value={aviso.componente} onChange={(e) => setAviso({ ...aviso, componente: e.target.value })}>
              <option value="geral">Geral</option>
              <option value="site">Site</option>
              <option value="pagamentos">Pagamentos</option>
              <option value="emails">E-mails</option>
              <option value="checkin">Check-in</option>
              <option value="repasses">Repasses</option>
            </select>
          </div>
          <div className="flex flex-col gap-1.5 sm:col-span-3">
            <Label htmlFor="aviso-msg">Mensagem (opcional)</Label>
            <Textarea id="aviso-msg" rows={2} maxLength={1000} value={aviso.mensagem} onChange={(e) => setAviso({ ...aviso, mensagem: e.target.value })} />
          </div>
        </div>
        <Button
          className="self-start"
          disabled={!aviso.titulo.trim()}
          onClick={async () => {
            if (await executar(() => api('/admin/avisos-status', { method: 'POST', body: aviso }))) setAviso({ ...aviso, titulo: '', mensagem: '' })
          }}
        >
          Publicar aviso
        </Button>
      </section>

      <section aria-labelledby="titulo-msgs" className="mt-8">
        <div className="mb-3 flex items-center justify-between gap-3">
          <h2 id="titulo-msgs" className="text-lg font-semibold text-foreground">
            Mensagens
          </h2>
          <select aria-label="Filtrar mensagens" className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground" value={filtro} onChange={(e) => setFiltro(e.target.value)}>
            <option value="aberto">Abertas</option>
            <option value="respondido">Respondidas</option>
            <option value="">Todas</option>
          </select>
        </div>
        {contatos.data?.length === 0 && <p className="text-sm text-muted-foreground">Nenhuma mensagem.</p>}
        <ul className="flex flex-col gap-3">
          {contatos.data?.map((m) => (
            <li key={m.id} className="rounded-xl border border-border p-4 text-sm">
              <p className="text-xs text-muted-foreground">
                #{m.id} · {new Date(m.criado_em).toLocaleString('pt-BR')} · {m.assunto_rotulo}
                {m.pedido ? ` · pedido ${m.pedido}` : ''} · {m.com_conta ? 'com conta' : 'sem conta'}
              </p>
              <p className="mt-1 font-medium text-foreground">
                {m.nome} ·{' '}
                <a href={`mailto:${m.email}?subject=${encodeURIComponent(`Protocolo #${m.id}`)}`} className="underline underline-offset-4">
                  {m.email}
                </a>
              </p>
              <p className="mt-2 whitespace-pre-wrap text-foreground">{m.mensagem}</p>
              {m.status === 'aberto' && (
                <Button size="sm" variant="outline" className="mt-3" onClick={() => executar(() => api(`/admin/suporte/${m.id}/respondido`, { method: 'POST' }))}>
                  Marcar como respondida
                </Button>
              )}
            </li>
          ))}
        </ul>
      </section>
      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}
    </main>
  )
}
