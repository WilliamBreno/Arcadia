import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { z } from 'zod'

import { api, ApiError } from '@/lib/api'
import type { Evento } from '@/lib/evento'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const schema = z.object({
  titulo: z.string().min(2, 'Informe o título'),
  categoria: z.string(),
  tipo_acesso: z.enum(['ingresso', 'cadastro']),
})

type FormValues = z.infer<typeof schema>

export default function EventoNovo() {
  const navigate = useNavigate()
  const [erro, setErro] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { tipo_acesso: 'ingresso' } })

  const aoEnviar = async (dados: FormValues) => {
    setErro(null)
    try {
      const evento = await api<Evento>('/org/eventos', { method: 'POST', body: dados })
      navigate(`/organizador/eventos/${evento.id}`, { replace: true })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao criar evento')
    }
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
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
              <Input id="categoria" placeholder="Games e Geek, música…" {...register('categoria')} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tipo_acesso">Tipo de acesso</Label>
              <select
                id="tipo_acesso"
                className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
                {...register('tipo_acesso')}
              >
                <option value="ingresso">Venda de ingressos</option>
                <option value="cadastro">Cadastro (gratuito ou pago sem taxa de ingresso)</option>
              </select>
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
