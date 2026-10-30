import { AlertTriangle, CheckCircle2, XCircle } from 'lucide-react'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import type { Verificacao } from '@/lib/ajuda'

const resultado: Record<Verificacao['situacao'], { titulo: string; texto: string; tom: 'ok' | 'atencao' | 'ruim' }> = {
  valido: {
    titulo: 'Ingresso válido e ainda não usado',
    texto: 'Isso não prova que quem está vendendo vai entregar só para você: um print pode ser vendido várias vezes. Peça a transferência pelo Evve (o ingresso ganha um QR novo no seu nome) ou compre na revenda oficial.',
    tom: 'atencao',
  },
  na_revenda: {
    titulo: 'Este ingresso está à venda na revenda oficial',
    texto: 'Compre pela página do evento, na revenda oficial: o ingresso passa para o seu nome e o do vendedor deixa de valer. Não pague fora do Evve.',
    tom: 'atencao',
  },
  utilizado: { titulo: 'Este ingresso já foi usado', texto: 'Alguém já entrou no evento com ele. Não compre.', tom: 'ruim' },
  cancelado: { titulo: 'Ingresso cancelado', texto: 'Este ingresso não vale mais (cancelado, reembolsado ou o evento foi cancelado). Não compre.', tom: 'ruim' },
  transferido: {
    titulo: 'Este código não vale mais',
    texto: 'O ingresso foi transferido para outra pessoa e ganhou um QR novo. Quem mostra este código está usando uma versão antiga. Não compre.',
    tom: 'ruim',
  },
  nao_pago: { titulo: 'Pagamento não confirmado', texto: 'Este pedido não foi pago. Não é um ingresso válido.', tom: 'ruim' },
  nao_encontrado: {
    titulo: 'Não encontramos este ingresso',
    texto: 'Confira o código (8 letras e números, no ingresso). Se a pessoa não sabe dizer o código ou o ingresso não é do Evve, desconfie.',
    tom: 'ruim',
  },
  evento_encerrado: { titulo: 'O evento já terminou', texto: 'Este ingresso era de um evento que já aconteceu.', tom: 'ruim' },
}

// Item 9.9: "este ingresso é válido?" — contra golpe de ingresso falso.
export default function VerificarIngresso() {
  const [params] = useSearchParams()
  const [texto, setTexto] = useState(params.get('codigo') ?? '')
  const [dados, setDados] = useState<Verificacao | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [carregando, setCarregando] = useState(false)

  const verificar = async () => {
    setErro(null)
    setCarregando(true)
    try {
      setDados(await api<Verificacao>(`/verificar-ingresso?codigo=${encodeURIComponent(texto)}`, { semAuth: true }))
    } catch (e) {
      setDados(null)
      setErro(e instanceof ApiError && e.status === 429 ? 'Muitas consultas seguidas. Espere um minuto.' : 'Não foi possível consultar agora.')
    } finally {
      setCarregando(false)
    }
  }

  const r = dados ? resultado[dados.situacao] : null
  const Icone = r?.tom === 'ok' ? CheckCircle2 : r?.tom === 'atencao' ? AlertTriangle : XCircle
  const cor = r?.tom === 'atencao' ? 'border-yellow-500 text-yellow-700 dark:text-yellow-400' : r?.tom === 'ok' ? 'border-green-600 text-green-700' : 'border-destructive text-destructive'

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <h1 className="text-3xl font-semibold text-foreground">Este ingresso é válido?</h1>
      <p className="mt-2 text-muted-foreground">Vai comprar de alguém? Digite o código do ingresso (8 letras e números) ou cole o link que a pessoa mandou.</p>
      <form
        className="mt-6 flex flex-col gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (texto.trim()) verificar()
        }}
      >
        <Label htmlFor="codigo-verificar">Código ou link do ingresso</Label>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input id="codigo-verificar" autoComplete="off" spellCheck={false} value={texto} onChange={(e) => setTexto(e.target.value)} placeholder="Ex.: K7MZ4QPA" />
          <Button type="submit" disabled={carregando || !texto.trim()}>
            {carregando ? 'Consultando…' : 'Verificar'}
          </Button>
        </div>
      </form>

      <div aria-live="polite">
        {erro && (
          <p role="alert" className="mt-4 text-sm text-destructive">
            {erro}
          </p>
        )}
        {r && dados && (
          <section className={`mt-6 rounded-xl border-2 p-4 ${cor}`}>
            <h2 className="flex items-center gap-2 text-lg font-semibold">
              <Icone className="size-5 shrink-0" aria-hidden="true" />
              {r.titulo}
            </h2>
            {dados.evento_titulo && (
              <p className="mt-2 text-sm text-foreground">
                <Link to={`/e/${dados.evento_slug}`} className="font-medium underline underline-offset-4">
                  {dados.evento_titulo}
                </Link>
                {dados.evento_inicio_em && ` · ${new Date(dados.evento_inicio_em).toLocaleDateString('pt-BR')}`}
                {dados.tipo && ` · ${dados.tipo}`}
              </p>
            )}
            <p className="mt-2 text-sm text-foreground">{r.texto}</p>
          </section>
        )}
      </div>

      <section aria-labelledby="titulo-dicas" className="mt-10">
        <h2 id="titulo-dicas" className="text-lg font-semibold text-foreground">
          Como não cair em golpe
        </h2>
        <ul className="mt-3 flex list-disc flex-col gap-2 pl-5 text-sm text-muted-foreground">
          <li>Print de QR Code pode ser vendido para várias pessoas. Só entra quem chegar primeiro.</li>
          <li>Peça a transferência pelo Evve: o ingresso vai para o seu nome com um QR novo, e o antigo deixa de valer na hora.</li>
          <li>Ingresso esgotado? Procure a revenda oficial na página do evento, com preço de tabela.</li>
          <li>O Evve nunca pede pagamento por Pix para pessoa física fora do site.</li>
        </ul>
        <p className="mt-4 text-sm text-muted-foreground">
          Acha que caiu num golpe?{' '}
          <Link to="/ajuda#titulo-contato" className="underline underline-offset-4">
            Fale com o suporte
          </Link>
          .
        </p>
      </section>
    </main>
  )
}
