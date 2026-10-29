import { SeloVerificado } from '@/components/selo-verificado'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useLocation, useNavigate } from 'react-router-dom'
import { z } from 'zod'

import { api, ApiError } from '@/lib/api'
import { useOrganizador } from '@/hooks/use-organizador'
import { VoltarLink } from '@/components/voltar-link'
import type { Organizador } from '@/lib/organizador'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Carregando } from '@/components/carregando'
import { CamposPix } from '@/components/campos-pix'

const schemaCriar = z.object({
  nome_publico: z.string().min(2, 'Informe o nome'),
  tipo_pessoa: z.enum(['pf', 'pj']),
  documento: z.string().min(11, 'Documento inválido'),
  tipo_chave_pix: z.string(),
  chave_pix: z.string(),
})

const schemaEditar = z.object({
  nome_publico: z.string().min(2, 'Informe o nome'),
  descricao: z.string(),
  logo_url: z.string(),
  chave_pix: z.string(),
  tipo_chave_pix: z.string(),
  instagram: z.string(),
  site: z.string(),
})

type FormCriar = z.infer<typeof schemaCriar>
type FormEditar = z.infer<typeof schemaEditar>

export default function Perfil() {
  const queryClient = useQueryClient()
  const { organizador, carregando: isLoading } = useOrganizador()
  const navigate = useNavigate()
  const location = useLocation()
  const voltarPara = (location.state as { voltarPara?: string } | null)?.voltarPara
  const [erro, setErro] = useState<string | null>(null)

  if (isLoading) {
    return <Carregando />
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <VoltarLink to={organizador ? '/organizador/eventos' : (voltarPara ?? '/')}>
        {organizador ? 'Voltar para meus eventos' : 'Voltar'}
      </VoltarLink>
      <h1 className="mb-2 text-2xl font-semibold text-foreground">
        {organizador ? 'Perfil de organizador' : 'Seja um organizador'}
      </h1>
      {!organizador && (
        <div className="mb-6 flex flex-col gap-2 text-sm text-muted-foreground">
          <p>Crie seu perfil para publicar eventos, vender ingressos ou receber inscrições e acompanhar tudo pelo painel.</p>
          <ol className="list-decimal pl-5">
            <li>Preencha os dados abaixo (leva menos de um minuto).</li>
            <li>Crie seu primeiro evento e configure ingressos ou inscrições.</li>
            <li>Publique e divulgue o link. Os repasses chegam depois do evento.</li>
          </ol>
        </div>
      )}
      {erro && <p className="mb-4 text-sm text-destructive">{erro}</p>}
      {organizador ? (
        <FormularioEditar
          organizador={organizador}
          onSalvar={() => queryClient.invalidateQueries({ queryKey: ['org-perfil'] })}
          onErro={setErro}
        />
      ) : (
        <FormularioCriar
          onCriado={async () => {
            await queryClient.invalidateQueries({ queryKey: ['org-perfil'] })
            navigate(voltarPara ?? '/organizador/eventos', { replace: true })
          }}
          onErro={setErro}
        />
      )}
    </main>
  )
}

function FormularioCriar({ onCriado, onErro }: { onCriado: () => void | Promise<void>; onErro: (e: string | null) => void }) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormCriar>({ resolver: zodResolver(schemaCriar), defaultValues: { tipo_pessoa: 'pf', tipo_chave_pix: '', chave_pix: '' } })

  const aoEnviar = async (dados: FormCriar) => {
    onErro(null)
    try {
      await api('/org/perfil', { method: 'POST', body: dados })
      await onCriado()
    } catch (e) {
      onErro(e instanceof ApiError ? e.message : 'Erro ao criar perfil')
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Criar perfil</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(aoEnviar)} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="nome_publico">Nome público</Label>
            <Input id="nome_publico" {...register('nome_publico')} />
            <p className="text-xs text-muted-foreground">Nome da sua marca, produtora ou grupo, exibido nas páginas dos eventos.</p>
            {errors.nome_publico && <p className="text-sm text-destructive">{errors.nome_publico.message}</p>}
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tipo_pessoa">Tipo de pessoa</Label>
            <select
              id="tipo_pessoa"
              className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
              {...register('tipo_pessoa')}
            >
              <option value="pf">Pessoa física (CPF)</option>
              <option value="pj">Pessoa jurídica (CNPJ)</option>
            </select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="documento">Documento</Label>
            <Input id="documento" placeholder="CPF ou CNPJ" {...register('documento')} />
            <p className="text-xs text-muted-foreground">Usado para identificar você nos repasses. Não aparece publicamente.</p>
            {errors.documento && <p className="text-sm text-destructive">{errors.documento.message}</p>}
          </div>
          <fieldset className="flex flex-col gap-2 rounded-lg border border-border p-3">
            <legend className="px-1 text-sm font-medium text-foreground">Pix para receber (pode preencher depois)</legend>
            <CamposPix register={register} tipoSelecionado={watch('tipo_chave_pix')} />
          </fieldset>
          <Button type="submit" disabled={isSubmitting}>
            Criar perfil
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

function FormularioEditar({
  organizador,
  onSalvar,
  onErro,
}: {
  organizador: Organizador
  onSalvar: () => void
  onErro: (e: string | null) => void
}) {
  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormEditar>({ resolver: zodResolver(schemaEditar), defaultValues: organizador })
  const [salvo, setSalvo] = useState(false)
  const [erroPerto, setErroPerto] = useState<string | null>(null)

  useEffect(() => reset(organizador), [organizador, reset])
  // Link "Cadastrar Pix" do painel leva direto para o bloco do Pix.
  useEffect(() => {
    if (window.location.hash === '#pix') document.getElementById('pix')?.scrollIntoView({ block: 'center' })
  }, [])

  const aoEnviar = async (dados: FormEditar) => {
    onErro(null)
    setSalvo(false)
    setErroPerto(null)
    try {
      await api('/org/perfil', { method: 'PUT', body: dados })
      setSalvo(true)
      onSalvar()
    } catch (e) {
      // Mostra perto do botão: no celular o topo da página fica fora da tela.
      setErroPerto(e instanceof ApiError ? e.message : 'Erro ao salvar')
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {organizador.nome_publico} <span className="text-sm font-normal text-muted-foreground">({organizador.status})</span>
        </CardTitle>
        {organizador.verificado ? (
          <SeloVerificado className="mt-1 self-start" />
        ) : (
          <p className="mt-1 text-xs text-muted-foreground">
            Aguardando verificação: a plataforma confere seu documento e se a chave Pix está no seu nome. Até lá, e nos seus 3
            primeiros repasses, 10% das vendas no cartão ficam retidas por 60 dias como reserva contra contestações. Trocar a
            chave Pix pede uma nova verificação.
          </p>
        )}
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(aoEnviar)} className="flex flex-col gap-4">
          <section id="pix" className="flex flex-col gap-3 rounded-lg border border-border p-3" aria-labelledby="titulo-pix">
            <div className="flex items-center justify-between gap-2">
              <h2 id="titulo-pix" className="font-medium text-foreground">Dados de recebimento (Pix)</h2>
              <span className={`shrink-0 whitespace-nowrap rounded-full px-2 py-0.5 text-xs ${organizador.chave_pix ? 'bg-muted text-foreground' : 'bg-foreground text-background'}`}>
                {organizador.chave_pix ? 'Cadastrada' : 'Falta cadastrar'}
              </span>
            </div>
            <CamposPix register={register} tipoSelecionado={watch('tipo_chave_pix')} />
          </section>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="nome_publico">Nome público</Label>
            <Input id="nome_publico" {...register('nome_publico')} />
            {errors.nome_publico && <p className="text-sm text-destructive">{errors.nome_publico.message}</p>}
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="descricao">Descrição</Label>
            <Textarea id="descricao" rows={3} {...register('descricao')} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="instagram">Instagram</Label>
            <Input id="instagram" {...register('instagram')} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="site">Site</Label>
            <Input id="site" {...register('site')} />
          </div>
          {salvo && <p className="text-sm text-muted-foreground" role="status">Perfil salvo.</p>}
          {erroPerto && <p className="text-sm text-destructive" role="alert">{erroPerto}</p>}
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Salvando…' : 'Salvar'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
