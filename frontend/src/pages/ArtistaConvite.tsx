import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'

import { ApiError } from '@/lib/api'
import { aceitarConviteIntegrante, recusarConviteIntegrante, verConviteIntegrante } from '@/lib/artistas'
import type { ConviteIntegrante } from '@/lib/artistas'
import { useAuth } from '@/hooks/use-auth'
import { Carregando } from '@/components/carregando'
import { UploadImagem } from '@/components/upload-imagem'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

function calcularIdade(data: string): number | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(data)) return null
  const [ano, mes, dia] = data.split('-').map(Number)
  const nascimento = new Date(ano, mes - 1, dia)
  const hoje = new Date()
  if (nascimento.getFullYear() !== ano || nascimento.getMonth() !== mes - 1 || nascimento.getDate() !== dia || nascimento > hoje) return null
  let idade = hoje.getFullYear() - ano
  if (hoje.getMonth() < mes - 1 || (hoje.getMonth() === mes - 1 && hoje.getDate() < dia)) idade--
  return idade
}

export default function ArtistaConvite() {
  const { token = '' } = useParams()
  const { usuario, carregando } = useAuth()
  const location = useLocation()
  const { data: convite, isLoading, error } = useQuery({
    queryKey: ['convite-integrante', token, usuario?.id ?? null],
    queryFn: () => verConviteIntegrante(token, !!usuario),
    enabled: !!token && !carregando,
    retry: false,
  })

  if (isLoading || carregando) return <Carregando />
  if (!token || error || !convite) {
    return (
      <main className="mx-auto max-w-lg px-4 py-16">
        <h1 className="text-2xl font-bold text-foreground">Convite indisponível</h1>
        <p role="alert" className="mt-2 text-destructive">
          {error instanceof ApiError ? error.message : 'Este convite é inválido, expirou ou não pôde ser carregado.'}
        </p>
      </main>
    )
  }

  if (!usuario) {
    return (
      <main className="mx-auto max-w-sm px-4 py-24 text-center">
        <h1 className="text-xl font-semibold text-foreground">Convite para {convite.grupo_nome}</h1>
        {convite.funcao && <p className="mt-2 text-foreground">Função: {convite.funcao}</p>}
        <p className="mt-2 break-words text-muted-foreground">Convite enviado para {convite.email_convite}.</p>
        <p role="status" className="mt-2 text-muted-foreground">Entre ou crie uma conta com esse e-mail para responder ao convite.</p>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <Link to="/login" state={{ de: location.pathname }} className="text-primary underline-offset-4 hover:underline">Entrar</Link>
          <Link to="/cadastro" className="text-primary underline-offset-4 hover:underline">Criar conta</Link>
        </div>
      </main>
    )
  }

  return <FormularioConvite key={`${token}-${usuario.id}`} token={token} convite={convite} />
}

function FormularioConvite({ token, convite }: { token: string; convite: ConviteIntegrante }) {
  const [nomeArtistico, setNomeArtistico] = useState(convite.nome_artistico)
  const [dataNascimento, setDataNascimento] = useState(convite.data_nascimento?.slice(0, 10) ?? '')
  const [foto, setFoto] = useState('')
  const [instagram, setInstagram] = useState('')
  const [responsavelNome, setResponsavelNome] = useState('')
  const [responsavelEmail, setResponsavelEmail] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)
  const [resultado, setResultado] = useState<'aceito' | 'aguardando' | 'recusado' | null>(null)
  const idade = calcularIdade(dataNascimento)
  const menorDe12 = idade !== null && idade < 12
  const precisaResponsavel = idade !== null && idade >= 12 && idade < 18
  const adulto = idade !== null && idade >= 18
  const hoje = new Date()
  const dataMaxima = `${hoje.getFullYear()}-${String(hoje.getMonth() + 1).padStart(2, '0')}-${String(hoje.getDate()).padStart(2, '0')}`

  const aceitar = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setErro(null)
    if (idade === null) {
      setErro('Informe uma data de nascimento válida, que não esteja no futuro.')
      return
    }
    if (nomeArtistico.trim().length < 2) {
      setErro('Informe seu nome artístico.')
      return
    }
    if (precisaResponsavel && (!responsavelNome.trim() || !responsavelEmail.trim())) {
      setErro('Informe o nome e o e-mail do responsável.')
      return
    }
    setEnviando(true)
    try {
      const resposta = await aceitarConviteIntegrante(token, {
        nome_artistico: nomeArtistico.trim(),
        data_nascimento: dataNascimento,
        foto_url: menorDe12 ? '' : foto,
        instagram: adulto ? instagram.trim().replace(/^@/, '') : '',
        responsavel_nome: precisaResponsavel ? responsavelNome.trim() : '',
        responsavel_email: precisaResponsavel ? responsavelEmail.trim() : '',
      })
      setResultado(resposta.aguardando_responsavel ? 'aguardando' : 'aceito')
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível aceitar o convite.')
    } finally {
      setEnviando(false)
    }
  }

  const recusar = async () => {
    setErro(null)
    setEnviando(true)
    try {
      await recusarConviteIntegrante(token)
      setResultado('recusado')
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível recusar o convite.')
    } finally {
      setEnviando(false)
    }
  }

  if (resultado) {
    return (
      <main className="mx-auto max-w-lg px-4 py-12">
        <h1 className="text-2xl font-bold text-foreground">{resultado === 'recusado' ? 'Convite recusado' : 'Convite aceito'}</h1>
        <p role="status" className="mt-2 text-muted-foreground">
          {resultado === 'aguardando'
            ? 'Falta a autorização do responsável. Enviamos um e-mail para ele; você só aparecerá no perfil depois que ele autorizar.'
            : resultado === 'recusado' ? 'Você recusou o convite para integrar o grupo.' : 'Sua participação no grupo foi confirmada.'}
        </p>
        <div className="mt-6 flex flex-wrap gap-3">
          <Link to={`/artista/${convite.grupo_slug}`} className="text-primary underline-offset-4 hover:underline">Ver perfil do grupo</Link>
          <Link to="/meus-artistas" className="text-primary underline-offset-4 hover:underline">Meus artistas</Link>
        </div>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <Card>
        <CardHeader>
          <CardTitle>Convite para {convite.grupo_nome}</CardTitle>
          {convite.funcao && <p className="text-sm text-muted-foreground">Função: {convite.funcao}</p>}
        </CardHeader>
        <CardContent>
          <form onSubmit={aceitar} className="flex flex-col gap-4">
            <fieldset disabled={enviando} className="flex min-w-0 flex-col gap-4">
              <legend className="sr-only">Seus dados de integrante</legend>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="nome_artistico">Nome artístico</Label>
                <Input id="nome_artistico" required minLength={2} maxLength={60} value={nomeArtistico} onChange={(e) => setNomeArtistico(e.target.value)} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="data_nascimento">Data de nascimento</Label>
                <Input id="data_nascimento" type="date" required max={dataMaxima} value={dataNascimento} onChange={(e) => setDataNascimento(e.target.value)} />
              </div>
              {idade !== null && !menorDe12 && <UploadImagem id="foto_integrante" rotulo="Foto (opcional)" valor={foto} onEnviado={setFoto} />}
              {adulto && (
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="instagram">Instagram (opcional)</Label>
                  <Input id="instagram" value={instagram} onChange={(e) => setInstagram(e.target.value)} />
                </div>
              )}
              {precisaResponsavel && (
                <div className="flex flex-col gap-4 rounded-lg border border-border p-3">
                  <p role="status" className="text-sm text-muted-foreground">O perfil só mostra você depois que seu responsável autorizar por e-mail. A foto também precisa de autorização.</p>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="responsavel_nome">Nome do responsável</Label>
                    <Input id="responsavel_nome" required value={responsavelNome} onChange={(e) => setResponsavelNome(e.target.value)} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="responsavel_email">E-mail do responsável</Label>
                    <Input id="responsavel_email" type="email" required value={responsavelEmail} onChange={(e) => setResponsavelEmail(e.target.value)} />
                  </div>
                </div>
              )}
            </fieldset>
            <div className="rounded-xl border border-border bg-muted p-4 text-sm text-foreground">
              <h2 className="font-medium">O que vai aparecer no perfil</h2>
              <p role="status" className="mt-1">
                {idade === null ? 'Informe sua data de nascimento para conferir o que será público.'
                  : menorDe12 ? 'Você aparece só como “Integrante”, sem nome nem foto. Seu Instagram também não aparece.'
                  : precisaResponsavel ? 'Depois da autorização: nome artístico, função e anos de experiência. A foto só aparece se o responsável autorizar. Nome real, idade, cidade, redes sociais e contato nunca aparecem.'
                  : 'Nome artístico, função e anos de experiência, além da foto e do Instagram, se você informar. Sua data de nascimento não aparece.'}
              </p>
            </div>
            {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
            <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
              <Button type="submit" disabled={enviando || idade === null}>Aceitar e aparecer no perfil</Button>
              <Button type="button" variant="outline" disabled={enviando} onClick={recusar}>Recusar</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
