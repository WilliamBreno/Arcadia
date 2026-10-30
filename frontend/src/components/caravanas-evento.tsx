import { useQuery } from '@tanstack/react-query'
import { Bus } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'
import { criarCaravana, minhasCaravanas, type DadosCaravana } from '@/lib/caravana'

// Caravanas na página do evento (item 8.7): só as do próprio usuário
// aparecem (o link é fechado) e quem quiser pode organizar uma.
export function CaravanasDoEvento({ slug, eventoComecou }: { slug: string; eventoComecou: boolean }) {
  const { usuario } = useAuth()
  const { data: minhas } = useQuery({
    queryKey: ['minhas-caravanas', slug, usuario?.id],
    enabled: !!usuario,
    queryFn: () => minhasCaravanas(slug).catch(() => []),
  })
  const [aberto, setAberto] = useState(false)
  if (eventoComecou && !minhas?.length) return null

  return (
    <section aria-labelledby="titulo-caravanas" className="mt-8 rounded-xl border border-border p-4 sm:p-5">
      <h2 id="titulo-caravanas" className="flex items-center gap-2 text-xl font-semibold text-foreground">
        <Bus className="size-5" aria-hidden="true" />
        Caravanas
      </h2>
      {minhas && minhas.length > 0 && (
        <ul className="mt-3 flex flex-col gap-2">
          {minhas.map((c) => (
            <li key={c.codigo}>
              <Link to={`/caravana/${c.codigo}`} className="block rounded-lg border border-border p-3 hover:bg-muted">
                <span className="block font-medium text-foreground">
                  {c.nome} {c.responsavel && <span className="text-xs font-normal text-muted-foreground">· você organiza</span>}
                </span>
                <span className="block text-xs text-muted-foreground">
                  {c.cidade} · {c.ponto_embarque}
                  {c.saida_em && ` · saída ${new Date(c.saida_em).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })}`}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
      {!eventoComecou &&
        (aberto ? (
          <NovaCaravana slug={slug} />
        ) : (
          <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground text-pretty">
              Vai com um grupo da sua cidade? Crie a caravana e mande o link para a turma. Cada um compra o próprio ingresso; você vê a lista de embarque.
            </p>
            {usuario ? (
              <Button size="sm" variant="outline" onClick={() => setAberto(true)}>
                Organizar caravana
              </Button>
            ) : (
              <Link to="/login" className="text-sm font-medium text-primary underline-offset-4 hover:underline">
                Entre para organizar
              </Link>
            )}
          </div>
        ))}
    </section>
  )
}

function NovaCaravana({ slug }: { slug: string }) {
  const navigate = useNavigate()
  const [d, setD] = useState({ nome: '', cidade: '', uf: '', ponto_embarque: '', saida: '', vagas: '', telefone: '' })
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)
  const campo = (k: keyof typeof d) => ({ value: d[k], onChange: (e: React.ChangeEvent<HTMLInputElement>) => setD({ ...d, [k]: e.target.value }) })

  const criar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setEnviando(true)
    const dados: DadosCaravana = {
      nome: d.nome,
      cidade: d.cidade,
      uf: d.uf,
      ponto_embarque: d.ponto_embarque,
      saida_em: d.saida ? new Date(d.saida).toISOString() : null,
      vagas: d.vagas ? Number(d.vagas) : null,
      telefone: d.telefone,
    }
    try {
      const r = await criarCaravana(slug, dados)
      navigate(`/caravana/${r.codigo}`)
    } catch (er) {
      setErro(er instanceof ApiError ? er.message : 'Não foi possível criar a caravana.')
      setEnviando(false)
    }
  }

  return (
    <form onSubmit={criar} className="mt-3 flex flex-col gap-3">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="cv-nome">Nome da caravana</Label>
        <Input id="cv-nome" required minLength={3} maxLength={80} placeholder="Caravana Arapiraca" {...campo('nome')} />
      </div>
      <div className="grid grid-cols-3 gap-3">
        <div className="col-span-2 flex flex-col gap-1.5">
          <Label htmlFor="cv-cidade">Cidade</Label>
          <Input id="cv-cidade" required minLength={2} {...campo('cidade')} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="cv-uf">UF</Label>
          <Input id="cv-uf" maxLength={2} {...campo('uf')} />
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="cv-ponto">Ponto de embarque</Label>
        <Input id="cv-ponto" required minLength={3} maxLength={200} {...campo('ponto_embarque')} />
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="cv-saida">Saída</Label>
          <Input id="cv-saida" type="datetime-local" {...campo('saida')} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="cv-vagas">Vagas (opcional)</Label>
          <Input id="cv-vagas" type="number" min="1" max="500" {...campo('vagas')} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="cv-tel">Seu telefone</Label>
          <Input id="cv-tel" type="tel" autoComplete="tel" {...campo('telefone')} />
        </div>
      </div>
      <p className="text-xs text-muted-foreground">O telefone só aparece para quem entrar na caravana. A caravana não aparece na página do evento: só quem tem o link entra.</p>
      {erro && (
        <p role="alert" className="text-sm text-destructive">
          {erro}
        </p>
      )}
      <Button type="submit" disabled={enviando}>
        Criar caravana
      </Button>
    </form>
  )
}
