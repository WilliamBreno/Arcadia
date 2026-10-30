import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { VoltarLink } from '@/components/voltar-link'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import { candidatar, expositoresDoEvento, type DadosCandidatura } from '@/lib/expositor'

// Candidatura de expositor (item 8.6). O organizador analisa, escolhe a
// mesa e, se o espaço for pago, a pessoa tem 48 h para pagar.
export default function CandidaturaExpositor() {
  const { slug = '' } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['expositores', slug], queryFn: () => expositoresDoEvento(slug) })
  const [d, setD] = useState<DadosCandidatura>({
    tipo_espaco_id: 0,
    marca: '',
    categoria: '',
    descricao: '',
    portfolio_url: '',
    instagram: '',
    responsavel_nome: '',
    telefone: '',
  })
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

  if (isLoading) return <Carregando />
  const tipos = data?.tipos ?? []
  const tipoId = d.tipo_espaco_id || tipos[0]?.id || 0
  const campo = (k: keyof DadosCandidatura) => ({
    value: String(d[k]),
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setD({ ...d, [k]: e.target.value }),
  })

  const enviar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setEnviando(true)
    try {
      await candidatar(slug, { ...d, tipo_espaco_id: tipoId })
      await queryClient.invalidateQueries({ queryKey: ['meu-espaco', slug] })
      navigate(`/e/${slug}#meu-espaco`, { replace: true })
    } catch (er) {
      setErro(er instanceof ApiError ? er.message : 'Não foi possível enviar a candidatura.')
      setEnviando(false)
    }
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <VoltarLink to={`/e/${slug}`}>Voltar ao evento</VoltarLink>
      <Card>
        <CardHeader>
          <CardTitle>Candidatura de expositor</CardTitle>
        </CardHeader>
        <CardContent>
          {!data?.aberto ? (
            <p className="text-sm text-muted-foreground">Este evento não está recebendo expositores agora.</p>
          ) : (
            <form onSubmit={enviar} className="flex flex-col gap-4">
              <p className="text-sm text-muted-foreground text-pretty">
                O organizador analisa a candidatura e escolhe a sua mesa. Espaço pago: depois de aprovado, você tem 48 horas para pagar (valor + taxa de
                serviço), com garantia opcional para desistir com reembolso.
              </p>
              <fieldset className="flex flex-col gap-2">
                <legend className="mb-1 text-sm font-medium text-foreground">Tipo de espaço</legend>
                {tipos.map((t) => (
                  <label key={t.id} className="flex items-start gap-2 rounded-lg border border-border p-3 text-sm">
                    <input type="radio" name="tipo" className="mt-0.5" checked={tipoId === t.id} onChange={() => setD({ ...d, tipo_espaco_id: t.id })} />
                    <span>
                      <span className="font-medium text-foreground">{t.nome}</span>{' '}
                      <span className="text-muted-foreground">
                        · {t.preco_centavos ? `${formatarCentavos(t.preco_centavos)} + taxa` : 'gratuito'} · {t.credenciais}{' '}
                        {t.credenciais === 1 ? 'credencial' : 'credenciais'}
                      </span>
                      {t.descricao && <span className="mt-0.5 block text-muted-foreground">{t.descricao}</span>}
                    </span>
                  </label>
                ))}
              </fieldset>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="marca">Nome da marca ou loja</Label>
                <Input id="marca" required minLength={2} maxLength={80} {...campo('marca')} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="categoria">O que você vende</Label>
                <Input id="categoria" placeholder="Ilustração, artesanato, colecionáveis…" maxLength={40} {...campo('categoria')} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="descricao">Descrição (aparece para o público depois de confirmado)</Label>
                <textarea
                  id="descricao"
                  rows={3}
                  maxLength={1000}
                  className="rounded-lg border border-border bg-background px-2.5 py-2 text-sm"
                  {...campo('descricao')}
                />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="portfolio">Portfólio (link)</Label>
                  <Input id="portfolio" type="url" placeholder="https://" {...campo('portfolio_url')} />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="instagram">Instagram</Label>
                  <Input id="instagram" placeholder="@sualoja" {...campo('instagram')} />
                </div>
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="responsavel">Seu nome (responsável)</Label>
                  <Input id="responsavel" required minLength={3} autoComplete="name" {...campo('responsavel_nome')} />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="telefone">Telefone (só o organizador vê)</Label>
                  <Input id="telefone" type="tel" autoComplete="tel" {...campo('telefone')} />
                </div>
              </div>
              {erro && (
                <p role="alert" className="text-sm text-destructive">
                  {erro}
                </p>
              )}
              <Button type="submit" disabled={enviando || !tipoId}>
                {enviando ? 'Enviando…' : 'Enviar candidatura'}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </main>
  )
}
