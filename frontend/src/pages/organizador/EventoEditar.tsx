import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useParams } from 'react-router-dom'
import { z } from 'zod'

import { Link } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { adicionarStaff, listarStaff, removerStaff, type Staff } from '@/lib/checkin'
import { formatarCentavos, type Evento, type TipoIngresso } from '@/lib/evento'
import type { Local } from '@/lib/organizador'
import { CortesiasCard } from '@/components/cortesias-card'
import { SolicitacoesCard } from '@/components/solicitacoes-card'
import { CuponsCard } from '@/components/cupons-card'
import { obterFinanceiroEvento } from '@/lib/financeiro'
import { obterVendas } from '@/lib/vendas'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

function paraDatetimeLocal(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function paraISO(datetimeLocal: string): string | null {
  if (!datetimeLocal) return null
  return new Date(datetimeLocal).toISOString()
}

const schemaBasico = z.object({
  titulo: z.string().min(2),
  categoria: z.string(),
  descricao: z.string(),
  classificacao_etaria: z.string(),
  tipo_acesso: z.enum(['ingresso', 'cadastro']),
  modo_participantes: z.enum(['nenhum', 'convite', 'inscricao_aberta', 'ambos']),
  visibilidade: z.enum(['publico', 'nao_listado', 'privado']),
  local_id: z.string(),
  inicio_em: z.string(),
  fim_em: z.string(),
  aprovacao_manual: z.boolean(),
})

type FormBasico = z.infer<typeof schemaBasico>

export default function EventoEditar() {
  const { id } = useParams()
  const queryClient = useQueryClient()
  const [erro, setErro] = useState<string | null>(null)
  const [problemasPublicacao, setProblemasPublicacao] = useState<string[] | null>(null)

  const { data: evento } = useQuery({
    queryKey: ['org-evento', id],
    queryFn: () => api<Evento>(`/org/eventos/${id}`),
  })
  const { data: locais } = useQuery({
    queryKey: ['org-locais'],
    queryFn: () => api<Local[]>('/org/locais'),
  })
  const { data: tiposIngresso } = useQuery({
    queryKey: ['org-evento-ingressos', id],
    queryFn: () => api<TipoIngresso[]>(`/org/eventos/${id}/ingressos`),
    enabled: evento !== undefined,
  })

  const {
    register,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = useForm<FormBasico>({ resolver: zodResolver(schemaBasico) })

  useEffect(() => {
    if (evento) {
      reset({
        ...evento,
        local_id: evento.local_id?.toString() ?? '',
        inicio_em: paraDatetimeLocal(evento.inicio_em),
        fim_em: paraDatetimeLocal(evento.fim_em),
      })
    }
  }, [evento, reset])

  const salvarBasico = async (dados: FormBasico) => {
    setErro(null)
    try {
      await api(`/org/eventos/${id}`, {
        method: 'PUT',
        body: {
          ...dados,
          local_id: dados.local_id ? Number(dados.local_id) : null,
          inicio_em: paraISO(dados.inicio_em),
          fim_em: paraISO(dados.fim_em),
        },
      })
      queryClient.invalidateQueries({ queryKey: ['org-evento', id] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao salvar')
    }
  }

  const publicar = async () => {
    setErro(null)
    setProblemasPublicacao(null)
    try {
      await api(`/org/eventos/${id}/publicar`, { method: 'POST' })
      queryClient.invalidateQueries({ queryKey: ['org-evento', id] })
    } catch (e) {
      if (e instanceof ApiError && e.status === 422) {
        const corpo = e.corpo as { problemas?: string[] } | undefined
        setProblemasPublicacao(corpo?.problemas ?? [e.message])
      } else {
        setErro(e instanceof ApiError ? e.message : 'Erro ao publicar')
      }
    }
  }

  if (!evento) {
    return <p className="p-8 text-center text-muted-foreground">Carregando…</p>
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <h1 className="mb-6 text-2xl font-semibold text-foreground">
        {evento.titulo} <span className="text-sm font-normal text-muted-foreground">({evento.status})</span>
      </h1>

      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Informações básicas</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(salvarBasico)} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="titulo">Título</Label>
              <Input id="titulo" {...register('titulo')} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="descricao">Descrição</Label>
              <Textarea id="descricao" rows={4} {...register('descricao')} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="categoria">Categoria</Label>
                <Input id="categoria" {...register('categoria')} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="classificacao_etaria">Classificação etária</Label>
                <Input id="classificacao_etaria" placeholder="Livre, 16 anos…" {...register('classificacao_etaria')} />
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="local_id">Local</Label>
              <select
                id="local_id"
                className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
                {...register('local_id')}
              >
                <option value="">Evento online (sem local)</option>
                {locais?.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.nome} — {l.cidade}/{l.uf}
                  </option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="inicio_em">Início</Label>
                <Input id="inicio_em" type="datetime-local" {...register('inicio_em')} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="fim_em">Término</Label>
                <Input id="fim_em" type="datetime-local" {...register('fim_em')} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="tipo_acesso">Tipo de acesso</Label>
                <select
                  id="tipo_acesso"
                  className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
                  {...register('tipo_acesso')}
                >
                  <option value="ingresso">Venda de ingressos</option>
                  <option value="cadastro">Cadastro</option>
                </select>
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="visibilidade">Visibilidade</Label>
                <select
                  id="visibilidade"
                  className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
                  {...register('visibilidade')}
                >
                  <option value="publico">Público</option>
                  <option value="nao_listado">Não listado</option>
                  <option value="privado">Privado</option>
                </select>
              </div>
            </div>

            <label className="flex items-center gap-2 text-sm text-foreground">
              <input type="checkbox" {...register('aprovacao_manual')} />
              Aprovação manual da plateia (o público solicita e você aprova antes de poder comprar; sem vaga vira lista de espera)
            </label>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="modo_participantes">Participantes (concurso)</Label>
              <select
                id="modo_participantes"
                className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
                {...register('modo_participantes')}
              >
                <option value="nenhum">Sem participantes</option>
                <option value="convite">Só por convite</option>
                <option value="inscricao_aberta">Inscrição aberta</option>
                <option value="ambos">Convite + inscrição aberta</option>
              </select>
            </div>

            {erro && <p className="text-sm text-destructive">{erro}</p>}
            <Button type="submit" disabled={isSubmitting}>
              Salvar
            </Button>
          </form>
        </CardContent>
      </Card>

      <TiposIngressoCard eventoId={Number(id)} tipos={tiposIngresso ?? []} />

      <VendasCard eventoId={Number(id)} />

      <FinanceiroCard eventoId={Number(id)} />

      <CuponsCard eventoId={Number(id)} />

      <CortesiasCard eventoId={Number(id)} tipos={tiposIngresso ?? []} />

      <ConvitesCard eventoId={Number(id)} />

      {evento.aprovacao_manual && <SolicitacoesCard eventoId={Number(id)} />}

      <ParticipantesCard eventoId={Number(id)} />

      <StaffCard eventoId={Number(id)} />

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>Publicação</CardTitle>
        </CardHeader>
        <CardContent>
          {evento.status === 'publicado' ? (
            <p className="text-sm text-muted-foreground">Evento publicado em {new Date(evento.publicado_em!).toLocaleString('pt-BR')}.</p>
          ) : (
            <>
              {problemasPublicacao && (
                <ul className="mb-4 list-inside list-disc text-sm text-destructive">
                  {problemasPublicacao.map((p) => (
                    <li key={p}>{p}</li>
                  ))}
                </ul>
              )}
              <Button onClick={publicar}>Publicar evento</Button>
            </>
          )}
        </CardContent>
      </Card>

      {evento.status === 'publicado' && <CancelarEventoCard eventoId={Number(id)} />}
    </main>
  )
}

function TiposIngressoCard({ eventoId, tipos }: { eventoId: number; tipos: TipoIngresso[] }) {
  const queryClient = useQueryClient()
  const [novo, setNovo] = useState({ nome: '', preco: '0', quantidade: '10', lote: '', meia: false })
  const [erro, setErro] = useState<string | null>(null)

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-evento-ingressos', String(eventoId)] })

  const adicionar = async () => {
    setErro(null)
    try {
      await api(`/org/eventos/${eventoId}/ingressos`, {
        method: 'POST',
        body: {
          nome: novo.nome,
          preco_centavos: Math.round(Number(novo.preco) * 100),
          quantidade: Number(novo.quantidade),
          lote_grupo: novo.lote,
          meia_entrada: novo.meia,
          ordem: tipos.length + 1,
          ativo: true,
        },
      })
      setNovo({ nome: '', preco: '0', quantidade: '10', lote: '', meia: false })
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao adicionar tipo de ingresso')
    }
  }

  const excluir = async (tipoId: number) => {
    await api(`/org/eventos/${eventoId}/ingressos/${tipoId}`, { method: 'DELETE' })
    invalidar()
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Ingressos</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {tipos.length === 0 && <p className="text-sm text-muted-foreground">Nenhum tipo de ingresso ainda.</p>}
        {tipos.map((t) => (
          <div key={t.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <div>
              <p className="text-sm font-medium text-foreground">{t.nome}</p>
              <p className="text-xs text-muted-foreground">
                {formatarCentavos(t.preco_centavos)} · {t.quantidade} unidades
                {t.lote_grupo ? ` · lote "${t.lote_grupo}" (ordem ${t.ordem})` : ''}
                {t.meia_entrada ? ' · meia-entrada' : ''}
              </p>
            </div>
            <Button variant="destructive" size="sm" onClick={() => excluir(t.id)}>
              Excluir
            </Button>
          </div>
        ))}

        <label className="flex items-center gap-2 border-t border-border pt-4 text-sm text-foreground">
          <input type="checkbox" checked={novo.meia} onChange={(e) => setNovo({ ...novo, meia: e.target.checked })} />
          Este é um ingresso de meia-entrada (máx. 40% do total; a portaria confere o documento)
        </label>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="novo-lote">Grupo de lotes (opcional)</Label>
          <Input
            id="novo-lote"
            placeholder="Ex.: inteira — ingressos com o mesmo grupo viram lotes em sequência (esgotou ou passou a data, abre o próximo)"
            value={novo.lote}
            onChange={(e) => setNovo({ ...novo, lote: e.target.value })}
          />
        </div>
        <div className="grid grid-cols-[2fr_1fr_1fr_auto] items-end gap-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="novo-nome">Nome</Label>
            <Input id="novo-nome" value={novo.nome} onChange={(e) => setNovo({ ...novo, nome: e.target.value })} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="novo-preco">Preço (R$)</Label>
            <Input
              id="novo-preco"
              type="number"
              min="0"
              step="0.01"
              value={novo.preco}
              onChange={(e) => setNovo({ ...novo, preco: e.target.value })}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="novo-qtd">Quantidade</Label>
            <Input
              id="novo-qtd"
              type="number"
              min="1"
              value={novo.quantidade}
              onChange={(e) => setNovo({ ...novo, quantidade: e.target.value })}
            />
          </div>
          <Button type="button" onClick={adicionar} disabled={!novo.nome}>
            Adicionar
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}

function FinanceiroCard({ eventoId }: { eventoId: number }) {
  const { data } = useQuery({
    queryKey: ['org-evento-financeiro', eventoId],
    queryFn: () => obterFinanceiroEvento(eventoId),
  })

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Financeiro</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 text-sm">
        {data && (
          <>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Receita bruta (preço dos ingressos)</span>
              <span className="text-foreground">{formatarCentavos(data.bruto_centavos)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Taxa do processador de pagamento</span>
              <span className="text-foreground">− {formatarCentavos(data.taxa_processador_centavos)}</span>
            </div>
            <div className="flex justify-between border-t border-border pt-2 font-medium">
              <span className="text-foreground">Líquido a receber</span>
              <span className="text-foreground">{formatarCentavos(data.liquido_centavos)}</span>
            </div>
            <p className="text-xs text-muted-foreground">
              {data.repasse_status
                ? `Repasse: ${data.repasse_status}`
                : data.liberar_em
                  ? `Repasse previsto a partir de ${new Date(data.liberar_em).toLocaleDateString('pt-BR')} (valores parciais até lá).`
                  : 'Defina a data do evento para prever o repasse.'}
            </p>
          </>
        )}
      </CardContent>
    </Card>
  )
}

function VendasCard({ eventoId }: { eventoId: number }) {
  const { data: vendas } = useQuery({
    queryKey: ['org-evento-vendas', eventoId],
    queryFn: () => obterVendas(eventoId),
  })

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Vendas</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="rounded-lg border border-border px-3 py-2">
            <p className="text-xs text-muted-foreground">Ingressos vendidos</p>
            <p className="text-lg font-semibold text-foreground">{vendas?.total_vendido ?? '—'}</p>
          </div>
          <div className="rounded-lg border border-border px-3 py-2">
            <p className="text-xs text-muted-foreground">Receita bruta</p>
            <p className="text-lg font-semibold text-foreground">
              {vendas ? formatarCentavos(vendas.receita_centavos) : '—'}
            </p>
          </div>
        </div>

        {vendas && vendas.por_tipo.length > 0 && (
          <div className="flex flex-col gap-2">
            {vendas.por_tipo.map((t) => (
              <div key={t.tipo_ingresso_id} className="flex items-center justify-between text-sm">
                <span className="text-foreground">{t.tipo_ingresso_nome}</span>
                <span className="text-muted-foreground">
                  {t.quantidade} · {formatarCentavos(t.receita_centavos)}
                </span>
              </div>
            ))}
          </div>
        )}

        <div className="flex flex-col gap-2 border-t border-border pt-4">
          {vendas?.itens.length === 0 && <p className="text-sm text-muted-foreground">Nenhuma venda ainda.</p>}
          {vendas?.itens.map((i) => (
            <div key={i.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
              <div>
                <p className="text-sm font-medium text-foreground">{i.titular_nome}</p>
                <p className="text-xs text-muted-foreground">
                  {i.tipo_ingresso_nome} · {i.codigo}
                </p>
              </div>
              <div className="text-right">
                <p className="text-sm text-foreground">{formatarCentavos(i.preco_centavos)}</p>
                <p className="text-xs text-muted-foreground">{i.status === 'utilizado' ? 'já entrou' : 'pago'}</p>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

type Convite = {
  id: number
  tipo: 'jurado' | 'participante_especial'
  token?: string
  max_usos: number | null
  usos: number
  revogado_em: string | null
}

const rotuloTipoConvite: Record<Convite['tipo'], string> = {
  jurado: 'Jurado',
  participante_especial: 'Participante especial',
}

function ConvitesCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [tipo, setTipo] = useState<Convite['tipo']>('jurado')
  const [ultimoLink, setUltimoLink] = useState<string | null>(null)
  const [erro, setErro] = useState<string | null>(null)

  const { data: convites } = useQuery({
    queryKey: ['org-evento-convites', eventoId],
    queryFn: () => api<Convite[]>(`/org/eventos/${eventoId}/convites`),
  })

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-evento-convites', eventoId] })

  const gerar = async () => {
    setErro(null)
    try {
      const convite = await api<Convite>(`/org/eventos/${eventoId}/convites`, { method: 'POST', body: { tipo } })
      setUltimoLink(`${window.location.origin}/convite/${convite.token}`)
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao gerar convite')
    }
  }

  const revogar = async (conviteId: number) => {
    await api(`/org/eventos/${eventoId}/convites/${conviteId}`, { method: 'DELETE' })
    invalidar()
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Convites (jurados e participantes especiais)</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex items-end gap-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tipo-convite">Tipo</Label>
            <select
              id="tipo-convite"
              className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
              value={tipo}
              onChange={(e) => setTipo(e.target.value as Convite['tipo'])}
            >
              <option value="jurado">Jurado</option>
              <option value="participante_especial">Participante especial</option>
            </select>
          </div>
          <Button type="button" onClick={gerar}>
            Gerar link
          </Button>
        </div>

        {ultimoLink && (
          <p className="rounded-lg border border-border p-2 text-sm break-all text-muted-foreground">{ultimoLink}</p>
        )}
        {erro && <p className="text-sm text-destructive">{erro}</p>}

        <div className="flex flex-col gap-2">
          {convites?.length === 0 && <p className="text-sm text-muted-foreground">Nenhum convite gerado ainda.</p>}
          {convites?.map((c) => (
            <div key={c.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
              <div>
                <p className="text-sm font-medium text-foreground">{rotuloTipoConvite[c.tipo]}</p>
                <p className="text-xs text-muted-foreground">
                  {c.usos} uso(s){c.max_usos ? ` de ${c.max_usos}` : ''}
                  {c.revogado_em ? ' · revogado' : ''}
                </p>
              </div>
              {!c.revogado_em && (
                <Button variant="destructive" size="sm" onClick={() => revogar(c.id)}>
                  Revogar
                </Button>
              )}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

type Ficha = {
  id: number
  papel: 'jurado' | 'participante'
  nome: string
  nome_artistico: string
  status: 'rascunho' | 'pendente' | 'aprovado' | 'rejeitado' | 'lista_espera' | 'desistiu'
  motivo_rejeicao?: string
  tipo_apresentacao: string | null
}

const rotuloStatusFicha: Record<Ficha['status'], string> = {
  rascunho: 'Rascunho',
  pendente: 'Pendente',
  aprovado: 'Aprovado',
  rejeitado: 'Rejeitado',
  lista_espera: 'Lista de espera',
  desistiu: 'Desistiu',
}

function ParticipantesCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()

  const { data: fichas } = useQuery({
    queryKey: ['org-evento-participantes', eventoId],
    queryFn: () => api<Ficha[]>(`/org/eventos/${eventoId}/participantes`),
  })

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-evento-participantes', eventoId] })

  const aprovar = async (fichaId: number) => {
    await api(`/org/eventos/${eventoId}/participantes/${fichaId}/aprovar`, { method: 'POST' })
    invalidar()
  }

  const rejeitar = async (fichaId: number) => {
    const motivo = window.prompt('Motivo da rejeição (opcional):') ?? ''
    await api(`/org/eventos/${eventoId}/participantes/${fichaId}/rejeitar`, { method: 'POST', body: { motivo } })
    invalidar()
  }

  const pendentes = fichas?.filter((f) => f.status === 'pendente') ?? []
  const outras = fichas?.filter((f) => f.status !== 'pendente') ?? []

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Participantes e jurados</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {fichas?.length === 0 && <p className="text-sm text-muted-foreground">Ninguém se inscreveu ou aceitou convite ainda.</p>}

        {pendentes.length > 0 && (
          <div className="flex flex-col gap-2">
            <p className="text-sm font-medium text-foreground">Aguardando aprovação</p>
            {pendentes.map((f) => (
              <div key={f.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {f.nome} {f.nome_artistico && `(${f.nome_artistico})`}
                  </p>
                  <p className="text-xs text-muted-foreground">{f.tipo_apresentacao ?? f.papel}</p>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" onClick={() => aprovar(f.id)}>
                    Aprovar
                  </Button>
                  <Button variant="destructive" size="sm" onClick={() => rejeitar(f.id)}>
                    Rejeitar
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}

        {outras.length > 0 && (
          <div className="flex flex-col gap-2">
            <p className="text-sm font-medium text-foreground">Demais</p>
            {outras.map((f) => (
              <div key={f.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {f.nome} {f.nome_artistico && `(${f.nome_artistico})`}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {f.papel} · {f.tipo_apresentacao ?? ''}
                    {f.status === 'rejeitado' && f.motivo_rejeicao ? ` · ${f.motivo_rejeicao}` : ''}
                  </p>
                </div>
                <span className="rounded-full border border-border px-2.5 py-0.5 text-xs text-muted-foreground">
                  {rotuloStatusFicha[f.status]}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function StaffCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [email, setEmail] = useState('')
  const [erro, setErro] = useState<string | null>(null)

  const { data: staff } = useQuery({
    queryKey: ['org-evento-staff', eventoId],
    queryFn: () => listarStaff(eventoId),
  })

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['org-evento-staff', eventoId] })

  const adicionar = async () => {
    setErro(null)
    try {
      await adicionarStaff(eventoId, email)
      setEmail('')
      invalidar()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao adicionar staff')
    }
  }

  const remover = async (staffItem: Staff) => {
    await removerStaff(eventoId, staffItem.usuario_id)
    invalidar()
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Staff e check-in</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Pessoas adicionadas aqui (já precisam ter uma conta na Arcadia) podem operar o leitor de QR na portaria.
        </p>

        <Link to={`/checkin/${eventoId}`} className={buttonVariants({ variant: 'outline' })}>
          Abrir leitor de check-in
        </Link>

        <div className="flex flex-col gap-2">
          {staff?.length === 0 && <p className="text-sm text-muted-foreground">Nenhum staff adicionado ainda.</p>}
          {staff?.map((s) => (
            <div key={s.usuario_id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
              <div>
                <p className="text-sm font-medium text-foreground">{s.nome}</p>
                <p className="text-xs text-muted-foreground">{s.email}</p>
              </div>
              <Button variant="destructive" size="sm" onClick={() => remover(s)}>
                Remover
              </Button>
            </div>
          ))}
        </div>

        <div className="flex items-end gap-2 border-t border-border pt-4">
          <div className="flex flex-1 flex-col gap-1.5">
            <Label htmlFor="staff-email">E-mail</Label>
            <Input id="staff-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <Button type="button" onClick={adicionar} disabled={!email}>
            Adicionar
          </Button>
        </div>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
      </CardContent>
    </Card>
  )
}

function CancelarEventoCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  const cancelar = async () => {
    const motivo = window.prompt('Motivo do cancelamento (todos os ingressos pagos serão reembolsados integralmente):')
    if (!motivo || motivo.trim().length < 3) return
    if (!confirm('Tem certeza? Essa ação não pode ser desfeita e reembolsa todos os compradores.')) return

    setErro(null)
    setEnviando(true)
    try {
      const resp = await api<{ sucessos: number; falhas: number[] }>(`/org/eventos/${eventoId}/cancelar`, {
        method: 'POST',
        body: { motivo },
      })
      if (resp.falhas.length > 0) {
        alert(`Evento cancelado. ${resp.sucessos} reembolso(s) concluído(s), ${resp.falhas.length} falharam — fale com o suporte.`)
      }
      queryClient.invalidateQueries({ queryKey: ['org-evento', String(eventoId)] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao cancelar evento')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <Card className="mt-6 border-destructive/30">
      <CardHeader>
        <CardTitle className="text-destructive">Cancelar evento</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <p className="text-sm text-muted-foreground">
          Cancela o evento e reembolsa integralmente (preço + taxa) todos os ingressos já pagos. Não pode ser desfeito.
        </p>
        {erro && <p className="text-sm text-destructive">{erro}</p>}
        <Button variant="destructive" onClick={cancelar} disabled={enviando} className="w-fit">
          {enviando ? 'Cancelando…' : 'Cancelar evento'}
        </Button>
      </CardContent>
    </Card>
  )
}
