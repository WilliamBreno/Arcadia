import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ArtistaGestao, ArtistaMidiaKit } from '@/components/artista-gestao'
import { ArtistaAgenda } from '@/components/artista-agenda'
import { ArtistaOferta } from '@/components/artista-oferta'
import { ArtistaOrcamentos } from '@/components/artista-orcamentos'
import { BadgeCheck, Copy, ExternalLink, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { UploadImagem } from '@/components/upload-imagem'
import { VoltarLink } from '@/components/voltar-link'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api, ApiError } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { obterPainelArtista, textoAnos, tiposArtista, type DadosArtista, type PainelArtista, type TipoArtista } from '@/lib/artistas'

const msg = (e: unknown, padrao: string) => (e instanceof ApiError ? e.message : padrao)
const dataCurta = (iso: string | null) => (iso ? new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' }) : '')

function Aviso({ erro, ok }: { erro: string | null; ok?: string | null }) {
  if (erro)
    return (
      <p role="alert" className="text-sm text-destructive">
        {erro}
      </p>
    )
  if (ok)
    return (
      <p role="status" className="text-sm text-muted-foreground">
        {ok}
      </p>
    )
  return null
}

function PerfilCard({ painel, aoSalvar }: { painel: PainelArtista; aoSalvar: (p: PainelArtista) => void }) {
  const [d, setD] = useState<DadosArtista>(painel.grupo)
  const [erro, setErro] = useState<string | null>(null)
  const [ok, setOk] = useState<string | null>(null)
  const [salvando, setSalvando] = useState(false)
  const mudar = (p: Partial<DadosArtista>) => {
    setOk(null)
    setD({ ...d, ...p })
  }
  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setSalvando(true)
    try {
      const r = await api<PainelArtista>(`/me/artistas/${painel.grupo.id}`, { method: 'PUT', body: d })
      aoSalvar(r)
      setD(r.grupo)
      setOk('Perfil salvo.')
    } catch (err) {
      setErro(msg(err, 'Não foi possível salvar.'))
    } finally {
      setSalvando(false)
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Perfil</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={salvar} className="flex flex-col gap-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="g-nome">{painel.grupo.solo ? 'Nome artístico' : 'Nome do grupo'}</Label>
              <Input id="g-nome" required maxLength={80} value={d.nome} onChange={(e) => mudar({ nome: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="g-tipo">O que faz</Label>
              <select id="g-tipo" className="h-9 rounded-lg border border-border bg-background px-2.5 text-sm" value={d.tipo} onChange={(e) => mudar({ tipo: e.target.value as TipoArtista })}>
                {tiposArtista.map((t) => (
                  <option key={t.valor} value={t.valor}>
                    {t.nome}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="g-insta">Instagram (aparece em destaque)</Label>
            <Input id="g-insta" placeholder="@seugrupo" value={d.instagram} onChange={(e) => mudar({ instagram: e.target.value })} />
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="g-tiktok">TikTok (opcional)</Label>
              <Input id="g-tiktok" value={d.tiktok} onChange={(e) => mudar({ tiktok: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="g-yt">YouTube (opcional)</Label>
              <Input id="g-yt" value={d.youtube} onChange={(e) => mudar({ youtube: e.target.value })} />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="g-bio">Apresentação</Label>
            <Textarea id="g-bio" rows={5} maxLength={1500} value={d.bio} placeholder="Quem são, o estilo, desde quando, prêmios…" onChange={(e) => mudar({ bio: e.target.value })} />
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2 flex flex-col gap-1.5">
              <Label htmlFor="g-cidade">Cidade</Label>
              <Input id="g-cidade" value={d.cidade} onChange={(e) => mudar({ cidade: e.target.value })} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="g-uf">UF</Label>
              <Input id="g-uf" maxLength={2} className="uppercase" value={d.uf} onChange={(e) => mudar({ uf: e.target.value.toUpperCase() })} />
            </div>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <UploadImagem id="g-logo" rotulo="Logo ou foto de perfil (quadrada)" valor={d.logo_url} onEnviado={(url) => mudar({ logo_url: url })} />
            <UploadImagem id="g-capa" rotulo="Capa (horizontal)" valor={d.capa_url} onEnviado={(url) => mudar({ capa_url: url })} />
          </div>
          <label className="flex items-start gap-2 text-sm text-foreground">
            <input type="checkbox" className="mt-0.5" checked={d.publico ?? true} onChange={(e) => mudar({ publico: e.target.checked })} />
            <span>
              Perfil público
              <span className="block text-xs text-muted-foreground">Desmarcado, o perfil some da vitrine e o link para de funcionar.</span>
            </span>
          </label>
          <Aviso erro={erro} ok={ok} />
          <Button type="submit" className="self-start" disabled={salvando}>
            {salvando ? 'Salvando…' : 'Salvar perfil'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

function IntegrantesCard({ painel, aoMudar }: { painel: PainelArtista; aoMudar: (p: PainelArtista) => void }) {
  const navigate = useNavigate()
  const id = painel.grupo.id
  const [email, setEmail] = useState('')
  const [nome, setNome] = useState('')
  const [funcao, setFuncao] = useState('')
  const [desde, setDesde] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const [ok, setOk] = useState<string | null>(null)
  const solo = painel.grupo.solo
  const podeConvidar = !solo || painel.integrantes.length === 0

  const convidar = async (e: React.FormEvent) => {
    e.preventDefault()
    setErro(null)
    setOk(null)
    try {
      const r = await api<{ token_proprio: string }>(`/me/artistas/${id}/integrantes`, {
        method: 'POST',
        body: { email, nome_artistico: nome, funcao, desde_ano: desde ? Number(desde) : null },
      })
      if (r.token_proprio) {
        navigate(`/grupo-convite/${r.token_proprio}`)
        return
      }
      setOk(`Convite enviado para ${email}. A pessoa aparece no perfil depois de aceitar.`)
      setEmail('')
      setNome('')
      setFuncao('')
      setDesde('')
      aoMudar(await obterPainelArtista(id))
    } catch (err) {
      setErro(msg(err, 'Não foi possível convidar.'))
    }
  }

  const remover = async (iid: number, quem: string) => {
    if (!window.confirm(`Tirar ${quem} do grupo? Some do perfil na hora.`)) return
    try {
      aoMudar(await api<PainelArtista>(`/me/artistas/${id}/integrantes/${iid}`, { method: 'DELETE' }))
    } catch (err) {
      setErro(msg(err, 'Não foi possível remover.'))
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{solo ? 'Você no perfil' : 'Integrantes'}</CardTitle>
        <p className="text-sm text-muted-foreground">
          {solo
            ? 'Confirme os seus dados para aparecer no perfil.'
            : 'Cada pessoa recebe um convite por e-mail e só aparece depois de aceitar com a própria conta Evve. Menores de idade só aparecem com a autorização do responsável.'}
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {painel.integrantes.length > 0 && (
          <ul className="flex flex-col divide-y divide-border rounded-xl border border-border">
            {painel.integrantes.map((i) => (
              <li key={i.id} className="flex items-center justify-between gap-3 px-3 py-2.5">
                <div className="flex min-w-0 items-center gap-3">
                  <span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted text-sm font-semibold text-foreground">
                    {i.foto_url ? <img src={i.foto_url} alt="" className="size-full object-cover" /> : i.nome_artistico.slice(0, 1).toUpperCase()}
                  </span>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-foreground">
                      {i.nome_artistico}
                      {i.sou_eu && <span className="font-normal text-muted-foreground"> (você)</span>}
                    </p>
                    <p className="truncate text-xs text-muted-foreground">
                      {[i.funcao, textoAnos(i.anos, painel.grupo.tipo)].filter(Boolean).join(' · ')}
                      {i.status === 'convidado' ? ` · convite enviado para ${i.email}` : ''}
                    </p>
                    {i.pendente && i.status !== 'convidado' && <p className="text-xs text-foreground">{i.pendente}</p>}
                  </div>
                </div>
                {!(solo && i.sou_eu) && (
                  <Button size="sm" variant="ghost" aria-label={`Tirar ${i.nome_artistico}`} onClick={() => remover(i.id, i.nome_artistico)}>
                    <Trash2 className="size-4" aria-hidden="true" />
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
        {podeConvidar && (
          <form onSubmit={convidar} className="flex flex-col gap-3 rounded-xl bg-muted p-3">
            <p className="text-sm font-medium text-foreground">{solo ? 'Entrar no perfil' : 'Convidar integrante'}</p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="i-email">E-mail {solo ? '(o da sua conta)' : 'da pessoa'}</Label>
                <Input id="i-email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="i-nome">Nome artístico</Label>
                <Input id="i-nome" required maxLength={60} value={nome} onChange={(e) => setNome(e.target.value)} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="i-funcao">Função</Label>
                <Input id="i-funcao" maxLength={60} placeholder={painel.grupo.tipo === 'cosplay' ? 'Cosplayer' : 'Dançarina, coreógrafo…'} value={funcao} onChange={(e) => setFuncao(e.target.value)} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="i-desde">Começou em (ano)</Label>
                <Input id="i-desde" inputMode="numeric" maxLength={4} placeholder="2018" value={desde} onChange={(e) => setDesde(e.target.value.replace(/\D/g, ''))} />
              </div>
            </div>
            <Aviso erro={erro} ok={ok} />
            <Button type="submit" size="sm" className="self-start">
              {solo ? 'Continuar' : 'Enviar convite'}
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  )
}

function PortfolioCard({ painel, aoMudar }: { painel: PainelArtista; aoMudar: (p: PainelArtista) => void }) {
  const id = painel.grupo.id
  const [video, setVideo] = useState('')
  const [legenda, setLegenda] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const adicionar = async (tipo: 'foto' | 'video', url: string) => {
    setErro(null)
    try {
      aoMudar(await api<PainelArtista>(`/me/artistas/${id}/midias`, { method: 'POST', body: { tipo, url, legenda } }))
      setVideo('')
      setLegenda('')
    } catch (err) {
      setErro(msg(err, 'Não foi possível adicionar.'))
    }
  }
  const remover = async (mid: number) => {
    try {
      aoMudar(await api<PainelArtista>(`/me/artistas/${id}/midias/${mid}`, { method: 'DELETE' }))
    } catch (err) {
      setErro(msg(err, 'Não foi possível remover.'))
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Portfólio</CardTitle>
        <p className="text-sm text-muted-foreground">Fotos e vídeos de apresentações. Vídeos por link do YouTube, Instagram ou TikTok. Até 40 itens.</p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {painel.midias.length > 0 && (
          <ul className="grid grid-cols-3 gap-2 sm:grid-cols-5">
            {painel.midias.map((m) => (
              <li key={m.id} className="relative overflow-hidden rounded-lg bg-muted">
                {m.tipo === 'foto' ? (
                  <img src={m.url} alt={m.legenda} className="aspect-square w-full object-cover" />
                ) : (
                  <a href={m.url} target="_blank" rel="noopener noreferrer" className="flex aspect-square items-center justify-center p-2 text-center text-xs text-foreground">
                    Vídeo · {m.plataforma ?? 'link'}
                  </a>
                )}
                <button
                  type="button"
                  aria-label="Remover"
                  onClick={() => remover(m.id)}
                  className="absolute right-1 top-1 rounded-full bg-background p-1 text-foreground shadow-sm"
                >
                  <Trash2 className="size-3.5" aria-hidden="true" />
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="m-legenda">Legenda (opcional, vale para o próximo item)</Label>
          <Input id="m-legenda" maxLength={140} placeholder="Cover no Anime Fest 2026" value={legenda} onChange={(e) => setLegenda(e.target.value)} />
        </div>
        <UploadImagem id="m-foto" rotulo="Adicionar foto" valor="" onEnviado={(url) => url && adicionar('foto', url)} />
        <form
          className="flex flex-col gap-2 sm:flex-row"
          onSubmit={(e) => {
            e.preventDefault()
            adicionar('video', video)
          }}
        >
          <Input type="url" placeholder="https://youtu.be/…" aria-label="Link do vídeo" value={video} onChange={(e) => setVideo(e.target.value)} />
          <Button type="submit" variant="outline" disabled={!video.trim()}>
            Adicionar vídeo
          </Button>
        </form>
        <Aviso erro={erro} />
      </CardContent>
    </Card>
  )
}

function SeloCard({ painel, aoMudar }: { painel: PainelArtista; aoMudar: (p: PainelArtista) => void }) {
  const id = painel.grupo.id
  const [erro, setErro] = useState<string | null>(null)
  const [copiado, setCopiado] = useState(false)
  const chamar = async (pedir: boolean) => {
    setErro(null)
    try {
      aoMudar(await api<PainelArtista>(`/me/artistas/${id}/instagram`, { method: 'POST', body: { pedir } }))
    } catch (err) {
      setErro(msg(err, 'Não foi possível agora.'))
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BadgeCheck className="size-5" aria-hidden="true" />
          Selo de Instagram verificado
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 text-sm">
        {!painel.grupo.instagram ? (
          <p className="text-muted-foreground">Informe o Instagram no perfil para pedir o selo.</p>
        ) : painel.instagram_verificado ? (
          <p className="text-foreground">@{painel.grupo.instagram} verificado. O selo aparece no perfil e na vitrine.</p>
        ) : !painel.instagram_codigo ? (
          <>
            <p className="text-muted-foreground">Prove que @{painel.grupo.instagram} é de vocês: coloque um código na bio do Instagram e a equipe da Evve confere.</p>
            <Button size="sm" className="self-start" onClick={() => chamar(false)}>
              Gerar código
            </Button>
          </>
        ) : (
          <>
            <ol className="list-decimal pl-5 text-muted-foreground">
              <li>Copie o código e cole na bio do Instagram @{painel.grupo.instagram}.</li>
              <li>Clique em "Pedir verificação". Depois do selo, pode tirar o código da bio.</li>
            </ol>
            <div className="flex items-center gap-2">
              <code className="rounded-lg bg-muted px-3 py-2 font-mono text-base font-semibold text-foreground">{painel.instagram_codigo}</code>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  navigator.clipboard?.writeText(painel.instagram_codigo).then(() => setCopiado(true))
                }}
              >
                <Copy className="size-4" aria-hidden="true" />
                {copiado ? 'Copiado' : 'Copiar'}
              </Button>
            </div>
            {painel.instagram_pedido ? (
              <p role="status" className="text-foreground">
                Pedido enviado. A equipe confere a bio em breve.
              </p>
            ) : (
              <Button size="sm" className="self-start" onClick={() => chamar(true)}>
                Pedir verificação
              </Button>
            )}
          </>
        )}
        <Aviso erro={erro} />
      </CardContent>
    </Card>
  )
}

function AtracoesCard({ painel, aoMudar }: { painel: PainelArtista; aoMudar: (p: PainelArtista) => void }) {
  const id = painel.grupo.id
  const [erro, setErro] = useState<string | null>(null)
  const decidir = async (aid: number, aceitar: boolean) => {
    setErro(null)
    try {
      aoMudar(await api<PainelArtista>(`/me/artistas/${id}/atracoes/${aid}`, { method: 'POST', body: { aceitar } }))
    } catch (err) {
      setErro(msg(err, 'Não foi possível agora.'))
    }
  }
  if (painel.atracoes.length === 0) return null
  const rotulo = { convidado: 'Aguardando a sua resposta', confirmado: 'Confirmado', recusado: 'Recusado' }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Convites de eventos</CardTitle>
        <p className="text-sm text-muted-foreground">Confirmados aparecem nas atrações do evento e no seu perfil.</p>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <ul className="flex flex-col divide-y divide-border rounded-xl border border-border">
          {painel.atracoes.map((a) => (
            <li key={a.id} className="flex flex-col gap-2 px-3 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <Link to={`/e/${a.evento_slug}`} className="inline-flex items-center gap-1 font-medium text-foreground underline-offset-4 hover:underline">
                  {a.evento_titulo}
                  <ExternalLink className="size-3.5" aria-hidden="true" />
                </Link>
                <p className="text-xs text-muted-foreground">
                  {dataCurta(a.evento_inicio)}
                  {a.cidade && ` · ${a.cidade}/${a.uf}`} · {rotulo[a.status]}
                </p>
              </div>
              <div className="flex gap-2">
                {a.status !== 'confirmado' && (
                  <Button size="sm" onClick={() => decidir(a.id, true)}>
                    Confirmar
                  </Button>
                )}
                {a.status !== 'recusado' && (
                  <Button size="sm" variant="outline" onClick={() => decidir(a.id, false)}>
                    {a.status === 'confirmado' ? 'Cancelar' : 'Recusar'}
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
        <Aviso erro={erro} />
      </CardContent>
    </Card>
  )
}

// Item 11.1: painel do líder (/meus-artistas/:id).
export default function ArtistaPainel() {
  const { id } = useParams()
  const queryClient = useQueryClient()
  const { usuario } = useAuth()
  const chave = ['painel-artista', Number(id), usuario?.id]
  const { data, isLoading, error } = useQuery({ queryKey: chave, queryFn: () => obterPainelArtista(Number(id)), retry: false })
  const atualizar = (p: PainelArtista) => queryClient.setQueryData(chave, p)

  useEffect(() => {
    if (data) document.title = `${data.grupo.nome} | Meus artistas`
  }, [data])

  if (isLoading) return <Carregando />
  if (error instanceof ApiError || !data) {
    return <p className="p-8 text-center text-muted-foreground">{error instanceof ApiError && error.status === 403 ? 'Só o líder e os cogestores gerenciam este perfil.' : 'Perfil não encontrado.'}</p>
  }

  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-6 px-4 py-10">
      <VoltarLink to="/meus-artistas">Meus artistas</VoltarLink>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold text-foreground">{data.grupo.nome}</h1>
        <Link to={`/artista/${data.grupo.slug}`} className={buttonVariants({ variant: 'outline', size: 'sm' })}>
          Ver perfil público
        </Link>
      </div>
      <ArtistaGestao id={data.grupo.id} solo={data.grupo.solo} />
      <ArtistaOrcamentos id={data.grupo.id} />
      <ArtistaOferta id={data.grupo.id} />
      <ArtistaAgenda id={data.grupo.id} />
      <ArtistaMidiaKit id={data.grupo.id} />
      {data.lider && <AtracoesCard painel={data} aoMudar={atualizar} />}
      <IntegrantesCard painel={data} aoMudar={atualizar} />
      <PerfilCard key={data.grupo.id} painel={data} aoSalvar={atualizar} />
      <PortfolioCard painel={data} aoMudar={atualizar} />
      {data.lider && <SeloCard painel={data} aoMudar={atualizar} />}
    </main>
  )
}
