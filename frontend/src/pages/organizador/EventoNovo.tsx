import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { z } from 'zod'

import { api, ApiError } from '@/lib/api'
import type { Evento } from '@/lib/evento'
import { CapaEvento } from '@/components/capa-evento'
import { CriacaoRapida, type LoteSugerido, type Rascunho } from '@/components/criacao-rapida'
import { Textarea } from '@/components/ui/textarea'
import { formatarCentavos } from '@/lib/evento'
import { CategoriaSelect } from '@/components/categoria-select'
import { Carregando } from '@/components/carregando'
import { VoltarLink } from '@/components/voltar-link'
import { useOrganizador } from '@/hooks/use-organizador'
import { DESCRICAO_TIPO_ACESSO } from '@/lib/organizador'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const schema = z.object({
  titulo: z.string().min(2, 'Informe o título'),
  categoria: z.string().min(2, 'Escolha ou digite uma categoria'),
  tipo_acesso: z.enum(['ingresso', 'cadastro']),
})

type FormValues = z.infer<typeof schema>

export default function EventoNovo() {
  const navigate = useNavigate()
  // Título digitado na página "Quanto custa" (item 6.3) chega pela URL.
  const [params] = useSearchParams()
  const [erro, setErro] = useState<string | null>(null)
  const [capa, setCapa] = useState('')
  // Item 10.5: o rascunho (link ou IA) preenche estes campos para revisão.
  const [descricao, setDescricao] = useState('')
  const [inicio, setInicio] = useState('')
  const [fim, setFim] = useState('')
  const [onde, setOnde] = useState('')
  const [lotes, setLotes] = useState<LoteSugerido[]>([])
  const [lotesMarcados, setLotesMarcados] = useState<boolean[]>([])
  const [dicas, setDicas] = useState<string[]>([])

  const { organizador, carregando } = useOrganizador()
  const {
    register,
    handleSubmit,
    control,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { titulo: params.get('titulo')?.slice(0, 200) ?? '', tipo_acesso: params.get('tipo') === 'cadastro' ? 'cadastro' : 'ingresso', categoria: '' } })
  const tipoAcesso = watch('tipo_acesso')

  const paraLocal = (iso?: string | null) => {
    if (!iso) return ''
    const d = new Date(iso)
    return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
  }
  const preencher = (r: Rascunho) => {
    if (r.titulo) setValue('titulo', r.titulo.slice(0, 200), { shouldValidate: true })
    if (r.categoria) setValue('categoria', r.categoria)
    if (r.descricao) setDescricao(r.descricao)
    if (r.inicio_em) setInicio(paraLocal(r.inicio_em))
    if (r.fim_em) setFim(paraLocal(r.fim_em))
    if (r.onde) setOnde(r.onde)
    if (r.lotes) {
      setLotes(r.lotes)
      setLotesMarcados(r.lotes.map(() => true))
    }
    if (r.dicas) setDicas(r.dicas)
  }

  const aoEnviar = async (dados: FormValues) => {
    setErro(null)
    try {
      const evento = await api<Evento>('/org/eventos', {
        method: 'POST',
        body: {
          ...dados,
          capa_url: capa,
          descricao,
          inicio_em: inicio ? new Date(inicio).toISOString() : undefined,
          fim_em: fim ? new Date(fim).toISOString() : undefined,
        },
      })
      // Lotes sugeridos que o organizador manteve marcados viram tipos de ingresso.
      for (const [i, l] of lotes.entries()) {
        if (!lotesMarcados[i]) continue
        await api(`/org/eventos/${evento.id}/ingressos`, {
          method: 'POST',
          body: { nome: l.nome, preco_centavos: l.preco_centavos, quantidade: l.quantidade, min_por_pedido: 1, max_por_pedido: 10, ativo: true },
        }).catch(() => undefined)
      }
      navigate(`/organizador/eventos/${evento.id}`, { replace: true })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar evento')
    }
  }

  if (carregando) return <Carregando />

  if (!organizador) {
    return (
      <main className="mx-auto max-w-lg px-4 py-12">
        <VoltarLink to="/">Voltar ao início</VoltarLink>
        <Card>
          <CardHeader>
            <CardTitle>Primeiro, crie seu perfil de organizador</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <p className="text-muted-foreground">
              Para criar eventos você precisa de um perfil de organizador: o nome que o público vê e os dados para receber
              os repasses. Leva menos de um minuto.
            </p>
            <Link
              to="/organizador/perfil"
              state={{ voltarPara: '/organizador/eventos/novo' }}
              className={buttonVariants()}
            >
              Criar perfil de organizador
            </Link>
          </CardContent>
        </Card>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <VoltarLink to="/organizador/eventos">Voltar para meus eventos</VoltarLink>
      <Card>
        <CardHeader>
          <CardTitle>Novo evento</CardTitle>
        </CardHeader>
        <CardContent>
          <CriacaoRapida titulo={watch('titulo') ?? ''} categoria={watch('categoria') ?? ''} aoPreencher={preencher} />
          <form onSubmit={handleSubmit(aoEnviar)} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="titulo">Título</Label>
              <Input id="titulo" {...register('titulo')} />
              {errors.titulo && <p className="text-sm text-destructive">{errors.titulo.message}</p>}
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="categoria">Categoria</Label>
              <Controller
                control={control}
                name="categoria"
                render={({ field }) => <CategoriaSelect id="categoria" value={field.value} onChange={field.onChange} />}
              />
              {errors.categoria && <p className="text-sm text-destructive">{errors.categoria.message}</p>}
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tipo_acesso">Tipo de acesso</Label>
              <select
                id="tipo_acesso"
                className="h-8 w-full rounded-lg border border-border bg-background px-2.5 text-sm"
                {...register('tipo_acesso')}
              >
                <option value="ingresso">Venda de ingressos</option>
                <option value="cadastro">Cadastro / inscrição (gratuito ou com valor)</option>
              </select>
              <p className="text-xs text-muted-foreground">{DESCRICAO_TIPO_ACESSO[tipoAcesso]}</p>
            </div>
            {(descricao || inicio || lotes.length > 0) && (
              <>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="descricao-novo">Descrição</Label>
                  <Textarea id="descricao-novo" rows={8} value={descricao} onChange={(e) => setDescricao(e.target.value)} />
                </div>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="inicio-novo">Início</Label>
                    <Input id="inicio-novo" type="datetime-local" value={inicio} onChange={(e) => setInicio(e.target.value)} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="fim-novo">Fim</Label>
                    <Input id="fim-novo" type="datetime-local" value={fim} onChange={(e) => setFim(e.target.value)} />
                  </div>
                </div>
                {onde && <p className="text-sm text-muted-foreground">Local na página original: {onde}. Cadastre o local na próxima tela.</p>}
                {lotes.length > 0 && (
                  <fieldset className="flex flex-col gap-2 rounded-lg border border-border p-3">
                    <legend className="px-1 text-sm font-medium text-foreground">Lotes sugeridos (criados ao continuar)</legend>
                    {lotes.map((l, i) => (
                      <label key={i} className="flex items-start gap-2 text-sm text-foreground">
                        <input
                          type="checkbox"
                          className="mt-0.5"
                          checked={!!lotesMarcados[i]}
                          onChange={(e) => setLotesMarcados(lotesMarcados.map((m, j) => (j === i ? e.target.checked : m)))}
                        />
                        <span>
                          <strong>{l.nome}</strong> · {formatarCentavos(l.preco_centavos)} · {l.quantidade} ingressos
                          {l.observacao && <span className="block text-xs text-muted-foreground">{l.observacao}</span>}
                        </span>
                      </label>
                    ))}
                  </fieldset>
                )}
                {dicas.length > 0 && (
                  <div className="rounded-lg bg-muted p-3 text-sm">
                    <p className="font-medium text-foreground">Dicas para vender mais</p>
                    <ul className="mt-1 list-disc pl-5 text-muted-foreground">
                      {dicas.map((d) => (
                        <li key={d}>{d}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </>
            )}
            <CapaEvento valor={capa} aoMudar={setCapa} />
            {erro && <p className="text-sm text-destructive">{erro}</p>}
            <Button type="submit" disabled={isSubmitting}>
              Continuar
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
