import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router-dom'
import { z } from 'zod'

import { api, ApiError } from '@/lib/api'
import type { Evento } from '@/lib/evento'
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
  const [erro, setErro] = useState<string | null>(null)

  const { organizador, carregando } = useOrganizador()
  const {
    register,
    handleSubmit,
    control,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { tipo_acesso: 'ingresso', categoria: '' } })
  const tipoAcesso = watch('tipo_acesso')

  const aoEnviar = async (dados: FormValues) => {
    setErro(null)
    try {
      const evento = await api<Evento>('/org/eventos', { method: 'POST', body: dados })
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
