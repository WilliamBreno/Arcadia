import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, useParams } from 'react-router-dom'
import { z } from 'zod'

import { api, ApiError } from '@/lib/api'
import type { Local } from '@/lib/organizador'
import { MapaPicker } from '@/components/mapa-picker'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

const schema = z.object({
  nome: z.string().min(2, 'Informe o nome do local'),
  logradouro: z.string(),
  numero: z.string(),
  bairro: z.string(),
  cidade: z.string().min(1, 'Informe a cidade'),
  uf: z.string().length(2, 'UF com 2 letras'),
  cep: z.string(),
  capacidade: z.string(),
  observacoes: z.string(),
})

type FormValues = z.infer<typeof schema>

export default function LocalForm() {
  const { id } = useParams()
  const editando = id !== undefined
  const navigate = useNavigate()
  const [erro, setErro] = useState<string | null>(null)
  const [posicao, setPosicao] = useState<{ lat: number | null; lng: number | null }>({ lat: null, lng: null })

  const { data: local } = useQuery({
    queryKey: ['org-local', id],
    queryFn: () => api<Local>(`/org/locais/${id}`),
    enabled: editando,
  })

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  useEffect(() => {
    if (local) {
      reset({ ...local, capacidade: local.capacidade?.toString() ?? '' })
      setPosicao({ lat: local.latitude, lng: local.longitude })
    }
  }, [local, reset])

  const aoEnviar = async (dados: FormValues) => {
    setErro(null)
    const capacidade = dados.capacidade.trim() === '' ? null : Number(dados.capacidade)
    const corpo = {
      ...dados,
      capacidade,
      latitude: posicao.lat,
      longitude: posicao.lng,
    }
    try {
      if (editando) {
        await api(`/org/locais/${id}`, { method: 'PUT', body: corpo })
      } else {
        await api('/org/locais', { method: 'POST', body: corpo })
      }
      navigate('/organizador/locais')
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao salvar local')
    }
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <Card>
        <CardHeader>
          <CardTitle>{editando ? 'Editar local' : 'Novo local'}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(aoEnviar)} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="nome">Nome do local</Label>
              <Input id="nome" {...register('nome')} />
              {errors.nome && <p className="text-sm text-destructive">{errors.nome.message}</p>}
            </div>

            <MapaPicker
              latitude={posicao.lat}
              longitude={posicao.lng}
              onSelecionar={(lat, lng) => setPosicao({ lat, lng })}
            />

            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2 flex flex-col gap-1.5">
                <Label htmlFor="logradouro">Logradouro</Label>
                <Input id="logradouro" {...register('logradouro')} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="numero">Número</Label>
                <Input id="numero" {...register('numero')} />
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="bairro">Bairro</Label>
              <Input id="bairro" {...register('bairro')} />
            </div>

            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2 flex flex-col gap-1.5">
                <Label htmlFor="cidade">Cidade</Label>
                <Input id="cidade" {...register('cidade')} />
                {errors.cidade && <p className="text-sm text-destructive">{errors.cidade.message}</p>}
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="uf">UF</Label>
                <Input id="uf" maxLength={2} {...register('uf')} />
                {errors.uf && <p className="text-sm text-destructive">{errors.uf.message}</p>}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="cep">CEP</Label>
                <Input id="cep" {...register('cep')} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="capacidade">Capacidade</Label>
                <Input id="capacidade" type="number" min={1} {...register('capacidade')} />
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="observacoes">Observações</Label>
              <Textarea id="observacoes" rows={3} {...register('observacoes')} />
            </div>

            {erro && <p className="text-sm text-destructive">{erro}</p>}
            <Button type="submit" disabled={isSubmitting}>
              Salvar
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
