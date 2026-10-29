import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Props = {
  titulo?: string
  aoEntrar: () => void
  // Para onde volta quem preferir entrar com senha.
  voltarPara?: string
}

// Entrar sem senha (item 7.2): e-mail, código de 6 dígitos e, se a conta
// não existe, o nome (o ingresso é nominal). Usado direto no checkout.
export function EntrarComCodigo({ titulo = 'Entre para continuar', aoEntrar, voltarPara }: Props) {
  const { entrarComCodigo } = useAuth()
  const [etapa, setEtapa] = useState<'email' | 'codigo'>('email')
  const [email, setEmail] = useState('')
  const [codigo, setCodigo] = useState('')
  const [nome, setNome] = useState('')
  const [precisaNome, setPrecisaNome] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const [podeReenviarEm, setPodeReenviarEm] = useState(0)
  const [agora, setAgora] = useState(() => Date.now())
  const campoCodigo = useRef<HTMLInputElement>(null)
  const campoNome = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (etapa !== 'codigo') return
    const t = window.setInterval(() => setAgora(Date.now()), 1000)
    return () => window.clearInterval(t)
  }, [etapa])

  const pedirCodigo = async (e?: React.FormEvent) => {
    e?.preventDefault()
    setErro(null)
    setEnviando(true)
    try {
      await api('/auth/codigo', { method: 'POST', body: { email: email.trim() }, semAuth: true })
      setEtapa('codigo')
      setCodigo('')
      setPodeReenviarEm(Date.now() + 30_000)
      window.setTimeout(() => campoCodigo.current?.focus(), 50)
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Não foi possível enviar o código.')
    } finally {
      setEnviando(false)
    }
  }

  const confirmar = async (e: React.FormEvent) => {
    e.preventDefault()
    if (precisaNome && nome.trim().split(/\s+/).length < 2) {
      setErro('Informe nome e sobrenome: o ingresso sai no seu nome.')
      return
    }
    setErro(null)
    setEnviando(true)
    try {
      await entrarComCodigo(email.trim(), codigo.trim(), precisaNome ? nome : undefined)
      aoEntrar()
    } catch (err) {
      if (err instanceof ApiError && (err.corpo as { precisa_nome?: boolean } | undefined)?.precisa_nome) {
        setPrecisaNome(true)
        window.setTimeout(() => campoNome.current?.focus(), 50)
      } else {
        setErro(err instanceof ApiError ? err.message : 'Não foi possível entrar.')
      }
    } finally {
      setEnviando(false)
    }
  }

  const espera = Math.max(0, Math.ceil((podeReenviarEm - agora) / 1000))

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-border p-4">
      <div>
        <h3 className="font-semibold text-foreground">{titulo}</h3>
        <p className="text-sm text-muted-foreground">
          {etapa === 'email' ? 'Sem senha: mandamos um código para o seu e-mail.' : `Enviamos um código de 6 dígitos para ${email.trim()}.`}
        </p>
      </div>

      {etapa === 'email' ? (
        <form onSubmit={pedirCodigo} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="codigo-email">E-mail</Label>
            <Input
              id="codigo-email"
              type="email"
              autoComplete="email"
              inputMode="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="voce@exemplo.com"
            />
          </div>
          <Button type="submit" size="lg" disabled={enviando}>
            {enviando ? 'Enviando…' : 'Receber código'}
          </Button>
        </form>
      ) : (
        <form onSubmit={confirmar} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="codigo-digitos">Código</Label>
            <Input
              ref={campoCodigo}
              id="codigo-digitos"
              inputMode="numeric"
              autoComplete="one-time-code"
              pattern="[0-9]{6}"
              maxLength={6}
              required
              value={codigo}
              onChange={(e) => setCodigo(e.target.value.replace(/\D/g, ''))}
              className="text-center font-mono text-lg tracking-widest"
              placeholder="000000"
            />
          </div>
          {precisaNome && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="codigo-nome">Seu nome completo</Label>
              <Input
                ref={campoNome}
                id="codigo-nome"
                autoComplete="name"
                required
                value={nome}
                onChange={(e) => setNome(e.target.value)}
                placeholder="Como aparece no seu documento"
              />
              <p className="text-xs text-muted-foreground">É a sua primeira vez aqui: criamos a conta com este nome.</p>
            </div>
          )}
          <Button type="submit" size="lg" disabled={enviando || codigo.length !== 6}>
            {enviando ? 'Entrando…' : precisaNome ? 'Criar conta e continuar' : 'Entrar e continuar'}
          </Button>
          <div className="flex flex-wrap justify-between gap-2 text-sm">
            <button
              type="button"
              className="text-muted-foreground underline underline-offset-4 hover:text-foreground"
              onClick={() => {
                setEtapa('email')
                setPrecisaNome(false)
                setErro(null)
              }}
            >
              Trocar e-mail
            </button>
            <button
              type="button"
              disabled={espera > 0 || enviando}
              onClick={() => pedirCodigo()}
              className="text-muted-foreground underline underline-offset-4 hover:text-foreground disabled:no-underline disabled:opacity-60"
            >
              {espera > 0 ? `Reenviar em ${espera}s` : 'Reenviar código'}
            </button>
          </div>
        </form>
      )}

      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
      {voltarPara && (
        <p className="text-xs text-muted-foreground">
          Prefere usar senha?{' '}
          <Link to="/login" state={{ de: voltarPara }} className="underline underline-offset-4 hover:text-foreground">
            Entrar com senha
          </Link>
        </p>
      )}
    </div>
  )
}
