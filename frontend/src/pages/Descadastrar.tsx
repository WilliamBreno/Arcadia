import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { VoltarBotao } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'

// Link "deixar de seguir" dos e-mails (item 6.4): funciona sem login. Pede
// um clique (POST) para que leitores de e-mail que abrem links sozinhos não
// descadastrem ninguém por engano.
export default function Descadastrar() {
  const [params] = useSearchParams()
  const [feito, setFeito] = useState<'um' | 'tudo' | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)
  const corpo = { u: params.get('u') ?? '', tipo: params.get('tipo') ?? '', alvo: params.get('alvo') ?? '', t: params.get('t') ?? '' }
  // Item 7.2: link do e-mail de carrinho (não é um "seguir").
  const carrinho = corpo.tipo === 'carrinho'

  useEffect(() => {
    document.title = 'Avisos por e-mail | Evve'
  }, [])

  const enviar = async (tudo: boolean) => {
    setErro(null)
    setEnviando(true)
    try {
      await api('/descadastrar', { method: 'POST', body: { ...corpo, tudo } })
      setFeito(tudo ? 'tudo' : 'um')
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível concluir')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <main className="mx-auto max-w-md px-4 py-16">
      <VoltarBotao fallback="/" rotulo="Ir para o início" />
      <h1 className="text-2xl font-bold text-foreground">Avisos por e-mail</h1>
      {feito && carrinho ? (
        <p className="mt-4 text-foreground" role="status">
          Pronto. Você não vai mais receber lembretes de compra nem avisos de eventos novos. E-mails dos seus ingressos continuam.
        </p>
      ) : carrinho ? (
        <>
          <p className="mt-4 text-foreground">
            Você recebeu um lembrete porque começou uma compra e não terminou. Quer parar de receber esses lembretes?
          </p>
          {erro && <p className="mt-4 text-sm text-destructive" role="alert">{erro}</p>}
          <div className="mt-6 flex flex-col gap-3">
            <Button onClick={() => enviar(true)} disabled={enviando}>
              Não quero receber estes lembretes
            </Button>
          </div>
          <p className="mt-3 text-xs text-muted-foreground">Isso também desliga os avisos de eventos novos. Os e-mails dos seus ingressos continuam.</p>
        </>
      ) : feito ? (
        <p className="mt-4 text-foreground" role="status">
          {feito === 'tudo'
            ? 'Pronto. Você não vai mais receber avisos de eventos novos. Lembretes dos seus ingressos continuam.'
            : 'Pronto. Você deixou de seguir e não recebe mais esses avisos.'}
        </p>
      ) : (
        <>
          <p className="mt-4 text-foreground">Você recebe e-mails de eventos novos porque segue um organizador ou uma categoria no Evve.</p>
          {erro && <p className="mt-4 text-sm text-destructive" role="alert">{erro}</p>}
          <div className="mt-6 flex flex-col gap-3">
            <Button onClick={() => enviar(false)} disabled={enviando}>
              Deixar de seguir este
            </Button>
            <Button variant="outline" onClick={() => enviar(true)} disabled={enviando}>
              Não receber mais nenhum aviso de evento novo
            </Button>
          </div>
        </>
      )}
      <Link to="/" className="mt-8 inline-block text-sm underline underline-offset-4">
        Ir para o início
      </Link>
    </main>
  )
}
