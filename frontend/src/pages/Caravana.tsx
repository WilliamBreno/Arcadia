import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { VoltarLink } from '@/components/voltar-link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'
import { encerrarCaravana, entrarCaravana, listaEmbarque, marcarEmbarque, sairCaravana, verCaravana } from '@/lib/caravana'

// Caravana (item 8.7): quem tem o link vê, entra e sai; o responsável vê
// a lista de embarque e marca quem embarcou.
export default function Caravana() {
  const { codigo = '' } = useParams()
  const { usuario } = useAuth()
  const queryClient = useQueryClient()
  const { data: c, isLoading, error } = useQuery({ queryKey: ['caravana', codigo], queryFn: () => verCaravana(codigo), retry: false })
  const recarregar = () => queryClient.invalidateQueries({ queryKey: ['caravana', codigo] })

  if (isLoading) return <Carregando />
  if (error || !c) {
    return (
      <main className="mx-auto max-w-lg px-4 py-16 text-center">
        <h1 className="text-2xl font-bold text-foreground">Caravana não encontrada</h1>
        <p className="mt-2 text-muted-foreground">O link pode ter sido encerrado pelo responsável.</p>
        <Link to="/" className="mt-6 inline-block text-sm font-medium underline underline-offset-4">
          Voltar para o início
        </Link>
      </main>
    )
  }
  const saida = c.saida_em ? new Date(c.saida_em).toLocaleString('pt-BR', { dateStyle: 'full', timeStyle: 'short' }) : null

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <VoltarLink to={`/e/${c.evento_slug}`}>Voltar ao evento</VoltarLink>
      <p className="text-sm font-medium uppercase tracking-wide text-muted-foreground">Caravana para {c.evento_titulo}</p>
      <h1 className="mt-1 text-3xl font-bold text-foreground text-balance">{c.nome}</h1>
      <dl className="mt-4 grid grid-cols-1 gap-2 text-sm">
        <div>
          <dt className="text-muted-foreground">Embarque</dt>
          <dd className="text-foreground">
            {c.ponto_embarque} · {c.cidade}
            {c.uf && `/${c.uf}`}
          </dd>
        </div>
        {saida && (
          <div>
            <dt className="text-muted-foreground">Saída</dt>
            <dd className="text-foreground">{saida}</dd>
          </div>
        )}
        <div>
          <dt className="text-muted-foreground">Responsável</dt>
          <dd className="text-foreground">
            {c.responsavel_nome}
            {c.telefone && ` · ${c.telefone}`}
          </dd>
        </div>
        <div>
          <dt className="text-muted-foreground">Pessoas</dt>
          <dd className="text-foreground">
            {c.membros}
            {c.vagas ? ` de ${c.vagas} vagas` : ''}
          </dd>
        </div>
      </dl>
      <p className="mt-3 text-sm text-muted-foreground">
        Cada pessoa compra o próprio ingresso.{' '}
        <Link to={`/e/${c.evento_slug}`} className="underline underline-offset-4">
          Ver ingressos do evento
        </Link>
      </p>

      {c.responsavel ? (
        <PainelResponsavel codigo={codigo} />
      ) : c.membro ? (
        <div className="mt-6 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border p-4">
          <p className="text-sm text-foreground">Você está nesta caravana.</p>
          <Button variant="outline" size="sm" onClick={async () => (await sairCaravana(codigo), recarregar())}>
            Sair da caravana
          </Button>
        </div>
      ) : (
        <Entrar codigo={codigo} nomePadrao={usuario?.nome ?? ''} aoEntrar={recarregar} lotada={!!c.vagas && c.membros >= c.vagas} />
      )}
    </main>
  )
}

function Entrar({ codigo, nomePadrao, aoEntrar, lotada }: { codigo: string; nomePadrao: string; aoEntrar: () => void; lotada: boolean }) {
  const [nome, setNome] = useState(nomePadrao)
  const [telefone, setTelefone] = useState('')
  const [aceite, setAceite] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  if (lotada) return <p className="mt-6 rounded-xl border border-border p-4 text-sm text-muted-foreground">Todas as vagas desta caravana já foram preenchidas.</p>
  const entrar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    try {
      await entrarCaravana(codigo, nome, telefone, aceite)
      aoEntrar()
    } catch (er) {
      setErro(er instanceof ApiError ? er.message : 'Não foi possível entrar agora.')
    }
  }
  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Entrar na caravana</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={entrar} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="m-nome">Seu nome</Label>
            <Input id="m-nome" required minLength={2} autoComplete="name" value={nome} onChange={(e) => setNome(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="m-tel">Telefone (para o responsável falar com você)</Label>
            <Input id="m-tel" type="tel" autoComplete="tel" value={telefone} onChange={(e) => setTelefone(e.target.value)} />
          </div>
          <label className="flex items-start gap-2 text-sm text-foreground">
            <input type="checkbox" className="mt-0.5" checked={aceite} onChange={(e) => setAceite(e.target.checked)} />
            <span>Aceito que o responsável pela caravana veja meu nome e telefone para organizar o embarque.</span>
          </label>
          {erro && (
            <p role="alert" className="text-sm text-destructive">
              {erro}
            </p>
          )}
          <Button type="submit" disabled={!aceite}>
            Entrar
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

function PainelResponsavel({ codigo }: { codigo: string }) {
  const queryClient = useQueryClient()
  const { data: lista } = useQuery({ queryKey: ['caravana-lista', codigo], queryFn: () => listaEmbarque(codigo) })
  const [copiado, setCopiado] = useState(false)
  const link = `${window.location.origin}/caravana/${codigo}`
  const recarregar = () => queryClient.invalidateQueries({ queryKey: ['caravana-lista', codigo] })
  const embarcaram = lista?.filter((l) => l.embarcou_em).length ?? 0

  return (
    <div className="mt-6 flex flex-col gap-4">
      <div className="rounded-xl border border-border p-4">
        <p className="text-sm font-medium text-foreground">Link para a turma</p>
        <div className="mt-2 flex gap-2">
          <Input readOnly value={link} aria-label="Link da caravana" onFocus={(e) => e.currentTarget.select()} />
          <Button
            variant="outline"
            onClick={async () => {
              await navigator.clipboard?.writeText(link)
              setCopiado(true)
            }}
          >
            {copiado ? 'Copiado' : 'Copiar'}
          </Button>
        </div>
      </div>

      <section aria-labelledby="t-embarque" className="rounded-xl border border-border p-4">
        <h2 id="t-embarque" className="flex items-baseline justify-between text-lg font-semibold text-foreground">
          Lista de embarque
          {lista && lista.length > 0 && (
            <span className="text-sm font-normal text-muted-foreground">
              {embarcaram} de {lista.length} embarcaram
            </span>
          )}
        </h2>
        {lista?.length === 0 && <p className="mt-2 text-sm text-muted-foreground">Ninguém entrou ainda. Mande o link para a turma.</p>}
        <ul className="mt-2 flex flex-col divide-y divide-border">
          {lista?.map((l) => (
            <li key={l.id} className="flex items-center gap-3 py-2">
              <input
                type="checkbox"
                id={`emb-${l.id}`}
                className="size-5"
                checked={!!l.embarcou_em}
                onChange={async (e) => (await marcarEmbarque(codigo, l.id, e.target.checked), recarregar())}
              />
              <label htmlFor={`emb-${l.id}`} className="min-w-0 flex-1">
                <span className="block text-sm font-medium text-foreground">{l.nome}</span>
                <span className="block text-xs text-muted-foreground">
                  {l.telefone || 'sem telefone'} · {l.tem_ingresso ? 'com ingresso' : 'ainda sem ingresso'}
                </span>
              </label>
              {l.telefone && (
                <a href={`tel:${l.telefone.replace(/[^\d+]/g, '')}`} className="text-xs underline underline-offset-4">
                  Ligar
                </a>
              )}
            </li>
          ))}
        </ul>
      </section>

      <Button
        variant="ghost"
        className="self-start"
        onClick={async () => {
          if (!confirm('Encerrar a caravana? O link para de funcionar para todos.')) return
          await encerrarCaravana(codigo)
          queryClient.invalidateQueries({ queryKey: ['caravana', codigo] })
        }}
      >
        Encerrar caravana
      </Button>
    </div>
  )
}
