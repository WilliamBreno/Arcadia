import { useEffect, useState } from 'react'

import { VoltarBotao } from '@/components/voltar-link'
import { aoMudarEscolha, lerEscolha, salvarEscolha } from '@/lib/consentimento'
import { Button } from '@/components/ui/button'

// Página "Cookies e pixels" (item 7.5): o que usamos, de quem, e como mudar.
export default function Cookies() {
  const [escolha, setEscolha] = useState(lerEscolha)
  useEffect(() => {
    document.title = 'Cookies e pixels | Evve'
    return aoMudarEscolha(setEscolha)
  }, [])

  const situacao =
    escolha === null
      ? 'Você ainda não escolheu: nada de marketing está ligado.'
      : escolha.marketing
        ? 'Você aceitou os pixels de anúncio.'
        : 'Você recusou os pixels de anúncio.'

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarBotao fallback="/" />
      <h1 className="text-3xl font-bold text-foreground">Cookies e pixels</h1>

      <section className="mt-8">
        <h2 className="text-xl font-semibold text-foreground">O que é sempre usado</h2>
        <p className="mt-2 text-foreground">
          Um cookie de sessão para manter você conectado com segurança, e o armazenamento do navegador para lembrar escolhas como
          esta. Sem eles o site não funciona, por isso não pedem consentimento. Não servem para anúncio.
        </p>
      </section>

      <section className="mt-8">
        <h2 className="text-xl font-semibold text-foreground">Pixels de anúncio dos organizadores (opcional)</h2>
        <p className="mt-2 text-foreground">
          O organizador de um evento pode ligar pixels da Meta (Facebook e Instagram), do Google Analytics 4 e do TikTok para
          medir os anúncios dele. Se você aceitar:
        </p>
        <ul className="mt-2 list-disc pl-5 text-foreground">
          <li>eles funcionam só na página daquele evento e no seu pedido de compra;</li>
          <li>
            medem que você viu o evento, começou a compra e concluiu a compra (valor e quantidade). Nome, e-mail, CPF e dados do
            pagamento nunca são enviados;
          </li>
          <li>cada plataforma usa os próprios cookies, conforme a política de privacidade dela.</li>
        </ul>
        <p className="mt-2 text-foreground">Se você recusar, nada disso carrega e o site funciona igual.</p>
      </section>

      <section className="mt-8 rounded-xl border border-border p-4">
        <p className="text-foreground" role="status">
          {situacao}
        </p>
        <div className="mt-3 flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => salvarEscolha(false)}>
            Recusar pixels
          </Button>
          <Button variant="outline" onClick={() => salvarEscolha(true)}>
            Aceitar pixels
          </Button>
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          A escolha fica salva só neste navegador. Você pode mudar quando quiser.
        </p>
      </section>
    </main>
  )
}
