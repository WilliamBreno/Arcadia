import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { z } from 'zod'

import { ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/hooks/use-auth'
import { EntrarComCodigo } from '@/components/entrar-com-codigo'

const schema = z.object({
  email: z.string().email('E-mail inválido'),
  senha: z.string().min(1, 'Informe sua senha'),
})

type FormValues = z.infer<typeof schema>

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [erro, setErro] = useState<string | null>(null)
  const [comCodigo, setComCodigo] = useState(false)
  const destino = (location.state as { de?: string } | null)?.de ?? '/'

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  const aoEnviar = async (dados: FormValues) => {
    setErro(null)
    try {
      await login(dados.email, dados.senha)
      navigate(destino, { replace: true })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao entrar')
    }
  }

  if (comCodigo) {
    return (
      <main className="mx-auto flex max-w-sm flex-col justify-center gap-3 px-4 py-24">
        <EntrarComCodigo titulo="Entrar com código" aoEntrar={() => navigate(destino, { replace: true })} />
        <button type="button" onClick={() => setComCodigo(false)} className="text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground">
          Entrar com senha
        </button>
      </main>
    )
  }

  return (
    <main className="mx-auto flex max-w-sm flex-col justify-center px-4 py-24">
      <Card>
        <CardHeader>
          <CardTitle>Entrar</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(aoEnviar)} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email">E-mail</Label>
              <Input id="email" type="email" autoComplete="email" {...register('email')} />
              {errors.email && <p className="text-sm text-destructive">{errors.email.message}</p>}
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="senha">Senha</Label>
              <Input id="senha" type="password" autoComplete="current-password" {...register('senha')} />
              {errors.senha && <p className="text-sm text-destructive">{errors.senha.message}</p>}
            </div>
            {erro && <p className="text-sm text-destructive">{erro}</p>}
            <Button type="submit" disabled={isSubmitting} className="w-full">
              Entrar
            </Button>
          </form>
          <Button type="button" variant="outline" className="mt-3 w-full" onClick={() => setComCodigo(true)}>
            Entrar sem senha, com código no e-mail
          </Button>
          <p className="mt-4 text-center text-sm text-muted-foreground">
            Não tem conta?{' '}
            <Link to="/cadastro" className="text-primary underline-offset-4 hover:underline">
              Cadastre-se
            </Link>
          </p>
        </CardContent>
      </Card>
    </main>
  )
}
