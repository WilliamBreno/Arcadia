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

const schemaCriar = z.object({
  nome_publico: z.string().min(2, 'Informe o nome'),
  tipo_pessoa: z.enum(['pf', 'pj']),
  documento: z.string().min(11, 'Documento inválido'),
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
    formState: { errors, isSubmitting },
  } = useForm<FormCriar>({ resolver: zodResolver(schemaCriar), defaultValues: { tipo_pessoa: 'pf' } })

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
    formState: { errors, isSubmitting },
  } = useForm<FormEditar>({ resolver: zodResolver(schemaEditar), defaultValues: organizador })

  useEffect(() => reset(organizador), [organizador, reset])

  const aoEnviar = async (dados: FormEditar) => {
    onErro(null)
    try {
      await api('/org/perfil', { method: 'PUT', body: dados })
      onSalvar()
    } catch (e) {
      onErro(e instanceof ApiError ? e.message : 'Erro ao salvar')
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {organizador.nome_publico} <span className="text-sm font-normal text-muted-foreground">({organizador.status})</span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(aoEnviar)} className="flex flex-col gap-4">
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
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tipo_chave_pix">Tipo de chave Pix</Label>
              <Input id="tipo_chave_pix" placeholder="email, cpf, telefone…" {...register('tipo_chave_pix')} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="chave_pix">Chave Pix</Label>
              <Input id="chave_pix" {...register('chave_pix')} />
            </div>
          </div>
          <Button type="submit" disabled={isSubmitting}>
            Salvar
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
