import { useQuery } from '@tanstack/react-query'
import { Link2, Sparkles } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api, ApiError } from '@/lib/api'

export type LoteSugerido = { nome: string; preco_centavos: number; quantidade: number; observacao: string }
export type Rascunho = {
  titulo?: string
  categoria?: string
  descricao?: string
  inicio_em?: string | null
  fim_em?: string | null
  onde?: string
  origem?: string
  lotes?: LoteSugerido[]
  dicas?: string[]
}

type Importado = {
  titulo: string
  descricao: string
  inicio_em: string | null
  fim_em: string | null
  local_nome: string
  endereco: string
  cidade: string
  uf: string
  online: boolean
  origem: string
}

// Item 10.5: criar evento em 2 minutos — importar de um link ou escrever com IA.
// Nada é salvo aqui: o rascunho preenche o formulário, que o organizador revisa.
export function CriacaoRapida({ titulo, categoria, aoPreencher }: { titulo: string; categoria: string; aoPreencher: (r: Rascunho) => void }) {
  const situacao = useQuery({
    queryKey: ['criacao-rapida'],
    queryFn: () => api<{ ia_ligada: boolean; restantes_hoje: number; limite_dia: number }>('/org/criacao-rapida'),
  })
  const [aba, setAba] = useState<'link' | 'ia'>('link')
  const [link, setLink] = useState('')
  const [ideia, setIdeia] = useState('')
  const [publico, setPublico] = useState('')
  const [preco, setPreco] = useState('')
  const [carregando, setCarregando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)
  const iaLigada = !!situacao.data?.ia_ligada

  const importar = async () => {
    setErro(null)
    setAviso(null)
    setCarregando(true)
    try {
      const r = await api<Importado>('/org/criacao-rapida/importar', { method: 'POST', body: { url: link } })
      const onde = r.online ? 'Online' : [r.local_nome, r.endereco, [r.cidade, r.uf].filter(Boolean).join('/')].filter(Boolean).join(' · ')
      aoPreencher({ titulo: r.titulo, descricao: r.descricao, inicio_em: r.inicio_em, fim_em: r.fim_em, onde, origem: r.origem })
      setAviso(`Dados trazidos de ${r.origem}. Confira tudo abaixo antes de continuar. A capa não é copiada: envie a sua.`)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível importar.')
    } finally {
      setCarregando(false)
    }
  }

  const gerar = async () => {
    setErro(null)
    setAviso(null)
    setCarregando(true)
    try {
      const reais = Number(preco.replace(',', '.'))
      const r = await api<{ descricao: string; lotes: LoteSugerido[]; dicas: string[]; restantes_hoje: number }>('/org/criacao-rapida/ia', {
        method: 'POST',
        body: { titulo, categoria, ideia, publico: Number(publico) || 0, preco_centavos: Number.isFinite(reais) && reais > 0 ? Math.round(reais * 100) : 0 },
      })
      aoPreencher({ descricao: r.descricao, lotes: r.lotes, dicas: r.dicas })
      setAviso(`Rascunho pronto: revise o texto e os lotes abaixo. Restam ${r.restantes_hoje} rascunho(s) hoje.`)
      situacao.refetch()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível gerar agora.')
    } finally {
      setCarregando(false)
    }
  }

  return (
    <section aria-labelledby="titulo-rapida" className="mb-6 rounded-xl border border-dashed border-border p-4">
      <h2 id="titulo-rapida" className="font-semibold text-foreground">
        Criar em 2 minutos
      </h2>
      <p className="text-sm text-muted-foreground">Traga os dados de uma página que você já tem ou peça um rascunho de texto e lotes.</p>
      <div role="tablist" aria-label="Como começar" className="mt-3 flex gap-2">
        <Button type="button" size="sm" role="tab" aria-selected={aba === 'link'} variant={aba === 'link' ? 'default' : 'outline'} onClick={() => setAba('link')}>
          <Link2 className="size-4" aria-hidden="true" />
          Importar de um link
        </Button>
        {iaLigada && (
          <Button type="button" size="sm" role="tab" aria-selected={aba === 'ia'} variant={aba === 'ia' ? 'default' : 'outline'} onClick={() => setAba('ia')}>
            <Sparkles className="size-4" aria-hidden="true" />
            Escrever com IA
          </Button>
        )}
      </div>

      {aba === 'link' ? (
        <div className="mt-3 flex flex-col gap-2">
          <Label htmlFor="link-importar">Link da página do evento</Label>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Input id="link-importar" type="url" inputMode="url" placeholder="https://www.sympla.com.br/evento/…" value={link} onChange={(e) => setLink(e.target.value)} />
            <Button type="button" onClick={importar} disabled={carregando || !link.trim()}>
              {carregando ? 'Lendo a página…' : 'Importar'}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">Funciona com Sympla, Even3, Eventbrite e sites com dados de evento. Instagram não (exige login).</p>
        </div>
      ) : (
        <div className="mt-3 flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ideia-ia">Conte em poucas palavras: o que é, para quem e o que tem de especial</Label>
            <Textarea id="ideia-ia" rows={3} maxLength={1500} value={ideia} onChange={(e) => setIdeia(e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="publico-ia">Público esperado</Label>
              <Input id="publico-ia" inputMode="numeric" placeholder="300" value={publico} onChange={(e) => setPublico(e.target.value)} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="preco-ia">Preço pensado (R$)</Label>
              <Input id="preco-ia" inputMode="decimal" placeholder="40" value={preco} onChange={(e) => setPreco(e.target.value)} />
            </div>
          </div>
          <Button type="button" className="self-start" onClick={gerar} disabled={carregando || titulo.trim().length < 3}>
            {carregando ? 'Escrevendo…' : 'Gerar rascunho'}
          </Button>
          <p className="text-xs text-muted-foreground">
            {titulo.trim().length < 3 ? 'Preencha o título abaixo primeiro. ' : ''}Texto gerado por IA: revise antes de publicar. {situacao.data && `${situacao.data.restantes_hoje} de ${situacao.data.limite_dia} hoje.`}
          </p>
        </div>
      )}
      {aviso && (
        <p role="status" className="mt-3 text-sm text-foreground">
          {aviso}
        </p>
      )}
      {erro && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {erro}
        </p>
      )}
    </section>
  )
}
