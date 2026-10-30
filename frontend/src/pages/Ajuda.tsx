import { ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useAuth } from '@/hooks/use-auth'
import { api, ApiError } from '@/lib/api'
import { assuntosSuporte, faqComprador, faqOrganizador, type Pergunta } from '@/lib/ajuda'

const semAcento = (s: string) =>
  s
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')

// Item 9.9: central de ajuda (perguntas frequentes e contato com o suporte).
export default function Ajuda() {
  const { usuario } = useAuth()
  const [busca, setBusca] = useState('')
  const [form, setForm] = useState({ nome: '', email: '', assunto: '', pedido: '', mensagem: '' })
  const [enviando, setEnviando] = useState(false)
  const [protocolo, setProtocolo] = useState<number | null>(null)
  const [erro, setErro] = useState<string | null>(null)

  const filtrar = (lista: Pergunta[]) => {
    const q = semAcento(busca.trim())
    return q ? lista.filter((x) => semAcento(`${x.p} ${x.r} ${x.k ?? ''}`).includes(q)) : lista
  }
  const comprador = filtrar(faqComprador)
  const organizador = filtrar(faqOrganizador)

  const enviar = async () => {
    setErro(null)
    setEnviando(true)
    try {
      const r = await api<{ protocolo: number }>('/suporte/contato', {
        method: 'POST',
        body: { ...form, nome: form.nome || usuario?.nome || '', email: form.email || usuario?.email || '' },
      })
      setProtocolo(r.protocolo)
    } catch (e) {
      setErro(e instanceof ApiError ? (e.status === 429 ? 'Muitas mensagens em pouco tempo. Tente de novo em alguns minutos.' : e.message) : 'Sem conexão. Tente de novo.')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <h1 className="text-3xl font-semibold text-foreground">Ajuda</h1>
      <p className="mt-2 text-muted-foreground">Respostas rápidas para quem compra e para quem organiza. Não achou? Fale com a gente no fim da página.</p>

      <div className="mt-6 flex flex-col gap-3 rounded-xl border border-border p-4 sm:flex-row sm:items-center">
        <ShieldCheck className="size-6 shrink-0 text-primary" aria-hidden="true" />
        <p className="flex-1 text-sm text-foreground">
          Vai comprar ingresso de alguém? Confira antes se ele é verdadeiro. A única troca segura é a transferência pelo Evve ou a revenda oficial.
        </p>
        <Link to="/verificar" className={buttonVariants({ variant: 'outline', size: 'sm' })}>
          Verificar ingresso
        </Link>
      </div>

      <div className="mt-8 flex flex-col gap-1.5">
        <Label htmlFor="busca-ajuda">Buscar nas perguntas</Label>
        <Input id="busca-ajuda" type="search" placeholder="reembolso, transferir, repasse…" value={busca} onChange={(e) => setBusca(e.target.value)} />
      </div>

      <Secao titulo="Para quem compra" id="comprador" perguntas={comprador} />
      <Secao titulo="Para quem organiza" id="organizador" perguntas={organizador} />
      {comprador.length + organizador.length === 0 && <p className="mt-6 text-muted-foreground">Nenhuma pergunta com esse termo. Escreva para a gente abaixo.</p>}

      <section aria-labelledby="titulo-contato" className="mt-12 rounded-xl border border-border p-4 sm:p-6">
        <h2 id="titulo-contato" className="text-xl font-semibold text-foreground">
          Fale com o suporte
        </h2>
        {protocolo ? (
          <div role="status" className="mt-3">
            <p className="text-foreground">
              Recebemos sua mensagem. Protocolo <strong className="tabular-nums">#{protocolo}</strong>.
            </p>
            <p className="mt-1 text-sm text-muted-foreground">Respondemos no e-mail que você informou, em geral no mesmo dia útil.</p>
          </div>
        ) : (
          <form
            className="mt-4 flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault()
              enviar()
            }}
          >
            {!usuario && (
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="contato-nome">Seu nome</Label>
                  <Input id="contato-nome" autoComplete="name" value={form.nome} onChange={(e) => setForm({ ...form, nome: e.target.value })} required />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="contato-email">E-mail para a resposta</Label>
                  <Input id="contato-email" type="email" autoComplete="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} required />
                </div>
              </div>
            )}
            {usuario && <p className="text-sm text-muted-foreground">Respondemos em {usuario.email}.</p>}
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="contato-assunto">Assunto</Label>
                <select
                  id="contato-assunto"
                  required
                  className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground"
                  value={form.assunto}
                  onChange={(e) => setForm({ ...form, assunto: e.target.value })}
                >
                  <option value="">Escolha</option>
                  {assuntosSuporte.map((a) => (
                    <option key={a.valor} value={a.valor}>
                      {a.rotulo}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="contato-pedido">Número do pedido (se tiver)</Label>
                <Input id="contato-pedido" inputMode="numeric" value={form.pedido} onChange={(e) => setForm({ ...form, pedido: e.target.value })} />
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="contato-mensagem">Mensagem</Label>
              <Textarea id="contato-mensagem" rows={5} maxLength={4000} value={form.mensagem} onChange={(e) => setForm({ ...form, mensagem: e.target.value })} required />
              <p className="text-xs text-muted-foreground">Não mande senha, número de cartão nem foto de documento.</p>
            </div>
            {erro && (
              <p role="alert" className="text-sm text-destructive">
                {erro}
              </p>
            )}
            <Button type="submit" className="self-start" disabled={enviando}>
              {enviando ? 'Enviando…' : 'Enviar mensagem'}
            </Button>
          </form>
        )}
      </section>

      <p className="mt-8 text-sm text-muted-foreground">
        Algo fora do ar?{' '}
        <Link to="/status" className="underline underline-offset-4">
          Veja o status da plataforma
        </Link>
        .
      </p>
    </main>
  )
}

function Secao({ titulo, id, perguntas }: { titulo: string; id: string; perguntas: Pergunta[] }) {
  if (perguntas.length === 0) return null
  return (
    <section aria-labelledby={`titulo-${id}`} className="mt-10">
      <h2 id={`titulo-${id}`} className="mb-3 text-xl font-semibold text-foreground">
        {titulo}
      </h2>
      <div className="divide-y divide-border rounded-xl border border-border">
        {perguntas.map((x) => (
          <details key={x.p} className="group px-4 py-3">
            <summary className="cursor-pointer list-none font-medium text-foreground marker:hidden focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
              <span className="flex items-center justify-between gap-3">
                {x.p}
                <span aria-hidden="true" className="text-muted-foreground transition-transform group-open:rotate-45">
                  +
                </span>
              </span>
            </summary>
            <p className="mt-2 text-sm text-muted-foreground">{x.r}</p>
          </details>
        ))}
      </div>
    </section>
  )
}
