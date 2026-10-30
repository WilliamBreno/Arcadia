import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

import { VoltarLink } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { idade, rotuloTipo, type DadosPerfil, type MeuPerfil } from '@/lib/talento'
import { UploadImagem } from '@/components/upload-imagem'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Carregando } from '@/components/carregando'

const vazio: DadosPerfil = {
  nome_artistico: '',
  bio: '',
  foto_url: '',
  cidade: '',
  uf: '',
  instagram: '',
  tiktok: '',
  youtube: '',
  data_nascimento: '',
  publico: true,
  responsavel_nome: '',
  responsavel_email: '',
}

// Meu perfil de talento (item 8.1): criar, editar, ver a situação (público,
// aguardando o responsável...) e ocultar participações.
export default function MeuTalento() {
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['meu-talento'], queryFn: () => api<MeuPerfil>('/me/talento') })
  const [d, setD] = useState<DadosPerfil>(vazio)
  const [salvando, setSalvando] = useState(false)
  const [msg, setMsg] = useState<{ ok: boolean; texto: string } | null>(null)
  const [copiado, setCopiado] = useState(false)

  useEffect(() => {
    document.title = 'Meu perfil de talento | Evve'
  }, [])
  useEffect(() => {
    if (data?.perfil) setD({ ...vazio, ...data.perfil })
  }, [data])

  if (isLoading) return <Carregando />

  const anos = idade(d.data_nascimento)
  const menor = anos !== null && anos < 18
  const muitoNovo = anos !== null && anos < 12
  const link = data?.slug ? `${window.location.origin}/t/${data.slug}` : ''

  const salvar = async (e: React.FormEvent) => {
    e.preventDefault()
    setMsg(null)
    setSalvando(true)
    try {
      const r = await api<MeuPerfil>('/me/talento', { method: 'PUT', body: d })
      queryClient.setQueryData(['meu-talento'], r)
      setMsg({ ok: true, texto: r.situacao.visivel ? 'Perfil salvo e público.' : 'Perfil salvo.' })
    } catch (err) {
      setMsg({ ok: false, texto: err instanceof ApiError ? err.message : 'Não foi possível salvar.' })
    } finally {
      setSalvando(false)
    }
  }

  const ocultar = async (fichaId: number, ocultarAgora: boolean) => {
    const r = await api<MeuPerfil>(`/me/talento/participacoes/${fichaId}`, { method: 'POST', body: { ocultar: ocultarAgora } })
    queryClient.setQueryData(['meu-talento'], r)
  }

  return (
    <main className="mx-auto max-w-2xl px-4 py-12">
      <VoltarLink to="/meus-eventos">Voltar aos meus eventos</VoltarLink>
      <h1 className="text-2xl font-semibold text-foreground">Meu perfil de talento</h1>
      <p className="mt-1 text-muted-foreground">
        Uma página sua, para compartilhar, com seus personagens e colocações nos concursos que usam o Evve.
      </p>

      {data?.perfil && (
        <div
          role="status"
          className={`mt-6 rounded-xl p-4 text-sm ${data.situacao.visivel ? 'border border-foreground' : 'bg-muted'}`}
        >
          {data.situacao.visivel ? (
            <>
              <p className="font-medium text-foreground">Seu perfil está público.</p>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <Link to={`/t/${data.slug}`} className="break-all text-foreground underline underline-offset-4">
                  {link}
                </Link>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={async () => {
                    await navigator.clipboard.writeText(link).catch(() => undefined)
                    setCopiado(true)
                    window.setTimeout(() => setCopiado(false), 2000)
                  }}
                >
                  {copiado ? 'Copiado' : 'Copiar link'}
                </Button>
              </div>
              {data.situacao.menor && (
                <p className="mt-2 text-muted-foreground">
                  Como você é menor de idade, o perfil mostra só o nome artístico, a bio, {data.autoriza_foto ? 'a foto, ' : ''}
                  personagens e colocações, e não aparece no Google.
                </p>
              )}
            </>
          ) : (
            <p className="text-foreground">{data.situacao.motivo}</p>
          )}
        </div>
      )}

      <form onSubmit={salvar} className="mt-6 flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="t-nome">Nome artístico</Label>
          <Input
            id="t-nome"
            value={d.nome_artistico}
            maxLength={60}
            onChange={(e) => setD({ ...d, nome_artistico: e.target.value })}
            required
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="t-nasc">Data de nascimento</Label>
          <Input
            id="t-nasc"
            type="date"
            value={d.data_nascimento}
            onChange={(e) => setD({ ...d, data_nascimento: e.target.value })}
            required
            className="max-w-xs"
          />
          <p className="text-xs text-muted-foreground">
            Não aparece no perfil. Serve para aplicar as regras de proteção a menores.
          </p>
        </div>

        {muitoNovo ? (
          <p role="alert" className="rounded-xl bg-muted p-4 text-sm text-foreground">
            Perfis públicos são para quem tem 12 anos ou mais. Seu nome artístico continua aparecendo nos resultados dos
            concursos.
          </p>
        ) : (
          <>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="t-bio">Sobre você</Label>
              <Textarea id="t-bio" rows={4} maxLength={500} value={d.bio} onChange={(e) => setD({ ...d, bio: e.target.value })} />
              <p className="text-right text-xs tabular-nums text-muted-foreground">{d.bio.length}/500</p>
              {menor && <p className="text-xs text-muted-foreground">Não escreva nome real, escola, cidade ou contato aqui.</p>}
            </div>
            <UploadImagem id="t-foto" rotulo="Foto" valor={d.foto_url} onEnviado={(url) => setD({ ...d, foto_url: url })} />

            {menor ? (
              <fieldset className="flex flex-col gap-3 rounded-xl border border-border p-4">
                <legend className="px-1 text-sm font-medium text-foreground">Responsável</legend>
                <p className="text-sm text-muted-foreground">
                  Como você tem menos de 18 anos, o perfil só fica público depois que seu responsável autorizar pelo link que
                  mandamos para o e-mail dele. Ele também decide se a foto aparece.
                </p>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="t-resp-nome">Nome do responsável</Label>
                  <Input
                    id="t-resp-nome"
                    value={d.responsavel_nome}
                    onChange={(e) => setD({ ...d, responsavel_nome: e.target.value })}
                    required
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="t-resp-email">E-mail do responsável</Label>
                  <Input
                    id="t-resp-email"
                    type="email"
                    value={d.responsavel_email}
                    onChange={(e) => setD({ ...d, responsavel_email: e.target.value })}
                    required
                  />
                </div>
              </fieldset>
            ) : (
              <>
                <div className="grid grid-cols-3 gap-3">
                  <div className="col-span-2 flex flex-col gap-1.5">
                    <Label htmlFor="t-cidade">Cidade</Label>
                    <Input id="t-cidade" value={d.cidade} onChange={(e) => setD({ ...d, cidade: e.target.value })} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="t-uf">UF</Label>
                    <Input
                      id="t-uf"
                      maxLength={2}
                      value={d.uf}
                      onChange={(e) => setD({ ...d, uf: e.target.value.toUpperCase() })}
                    />
                  </div>
                </div>
                <div className="grid gap-3 sm:grid-cols-3">
                  {(
                    [
                      ['instagram', 'Instagram'],
                      ['tiktok', 'TikTok'],
                      ['youtube', 'YouTube'],
                    ] as const
                  ).map(([k, rotulo]) => (
                    <div key={k} className="flex flex-col gap-1.5">
                      <Label htmlFor={`t-${k}`}>{rotulo}</Label>
                      <Input
                        id={`t-${k}`}
                        value={d[k]}
                        placeholder="@seuperfil"
                        onChange={(e) => setD({ ...d, [k]: e.target.value })}
                      />
                    </div>
                  ))}
                </div>
              </>
            )}

            <label className="flex items-center gap-2 text-sm text-foreground">
              <input type="checkbox" checked={d.publico} onChange={(e) => setD({ ...d, publico: e.target.checked })} />
              Deixar meu perfil público
            </label>
          </>
        )}

        {msg && (
          <p role="status" className={`text-sm ${msg.ok ? 'text-foreground' : 'text-destructive'}`}>
            {msg.texto}
          </p>
        )}
        {!muitoNovo && (
          <div>
            <Button type="submit" disabled={salvando}>
              {salvando ? 'Salvando…' : data?.perfil ? 'Salvar perfil' : 'Criar meu perfil'}
            </Button>
          </div>
        )}
      </form>

      <section className="mt-10" aria-labelledby="titulo-minhas-part">
        <h2 id="titulo-minhas-part" className="text-lg font-semibold text-foreground">
          Seus concursos
        </h2>
        <p className="text-sm text-muted-foreground">
          Os concursos em que você foi aprovado entram sozinhos. Oculte o que não quiser mostrar.
        </p>
        {(data?.participacoes.length ?? 0) === 0 && <p className="mt-3 text-sm text-muted-foreground">Nenhum concurso ainda.</p>}
        <div className="mt-3 flex flex-col gap-2">
          {data?.participacoes.map((x) => (
            <Card key={x.ficha_id} className={x.oculta ? 'opacity-60' : ''}>
              <CardContent className="flex flex-wrap items-center justify-between gap-2 py-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-foreground">
                    {x.personagem || rotuloTipo[x.tipo] || 'Apresentação'} · {x.evento_titulo}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {x.posicao ? `${x.posicao}º lugar` : 'Resultado ainda não liberado'}
                    {x.oculta ? ' · oculto no perfil' : ''}
                  </p>
                </div>
                <Button type="button" variant="outline" size="sm" onClick={() => ocultar(x.ficha_id, !x.oculta)}>
                  {x.oculta ? 'Mostrar' : 'Ocultar'}
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>
    </main>
  )
}
