import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { UploadAudio } from '@/components/upload-audio'
import { UploadImagem } from '@/components/upload-imagem'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/hooks/use-auth'
import { Carregando } from '@/components/carregando'

type ConviteInfo = { tipo: 'jurado' | 'participante_especial'; evento_titulo: string; evento_slug: string }

const rotuloTipo: Record<ConviteInfo['tipo'], string> = {
  jurado: 'Jurado',
  participante_especial: 'Participante especial',
}

function calcularIdade(dataNascimento: string): number | null {
  if (!dataNascimento) return null
  const nascimento = new Date(dataNascimento)
  const hoje = new Date()
  let idade = hoje.getFullYear() - nascimento.getFullYear()
  const aindaNaoFezAniversario =
    hoje.getMonth() < nascimento.getMonth() ||
    (hoje.getMonth() === nascimento.getMonth() && hoje.getDate() < nascimento.getDate())
  if (aindaNaoFezAniversario) idade--
  return idade
}

export default function Convite() {
  const { token } = useParams()
  const { usuario, carregando } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()

  const { data: convite, isLoading, error } = useQuery({
    queryKey: ['convite', token],
    queryFn: () => api<ConviteInfo>(`/convites/${token}`),
  })

  if (isLoading || carregando) return <Carregando />
  if (error instanceof ApiError) {
    return <p className="p-8 text-center text-muted-foreground">Convite inválido, expirado ou revogado.</p>
  }
  if (!convite) return null

  if (!usuario) {
    return (
      <main className="mx-auto max-w-sm px-4 py-24 text-center">
        <h1 className="text-xl font-semibold text-foreground">
          Convite de {rotuloTipo[convite.tipo]} — {convite.evento_titulo}
        </h1>
        <p className="mt-2 text-muted-foreground">Entre ou crie uma conta para aceitar o convite.</p>
        <div className="mt-6 flex justify-center gap-3">
          <Link to="/login" state={{ de: location.pathname }} className="text-primary underline-offset-4 hover:underline">
            Entrar
          </Link>
          <Link to="/cadastro" className="text-primary underline-offset-4 hover:underline">
            Criar conta
          </Link>
        </div>
      </main>
    )
  }

  return <FormularioFicha token={token!} convite={convite} onConfirmado={() => navigate(`/e/${convite.evento_slug}`)} />
}

const camposPorTipo: Record<string, { chave: string; rotulo: string }[]> = {
  cosplay: [
    { chave: 'personagem', rotulo: 'Personagem' },
    { chave: 'obra_origem', rotulo: 'Obra de origem' },
  ],
  danca: [
    { chave: 'estilo', rotulo: 'Estilo' },
    { chave: 'instagram_grupo', rotulo: 'Instagram do grupo/dançarino(a)' },
  ],
  canto: [
    { chave: 'musica', rotulo: 'Música' },
    { chave: 'artista', rotulo: 'Artista original' },
  ],
  atuacao: [
    { chave: 'titulo_cena', rotulo: 'Título da cena' },
    { chave: 'necessidades_palco', rotulo: 'Necessidades de palco' },
  ],
}

function FormularioFicha({
  token,
  convite,
  onConfirmado,
}: {
  token: string
  convite: ConviteInfo
  onConfirmado: () => void
}) {
  const ehParticipante = convite.tipo === 'participante_especial'

  const [nome, setNome] = useState('')
  const [nomeArtistico, setNomeArtistico] = useState('')
  const [instagram, setInstagram] = useState('')
  const [dataNascimento, setDataNascimento] = useState('')
  const [telefone, setTelefone] = useState('')
  const [fotoURL, setFotoURL] = useState('')
  const [tipoApresentacao, setTipoApresentacao] = useState('cosplay')
  const [dados, setDados] = useState<Record<string, string>>({})
  const [responsavelNome, setResponsavelNome] = useState('')
  const [responsavelContato, setResponsavelContato] = useState('')
  const [autorizacaoURL, setAutorizacaoURL] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

  const idade = calcularIdade(dataNascimento)
  const menorDeIdade = idade !== null && idade < 18

  const confirmar = async () => {
    setErro(null)
    setEnviando(true)
    try {
      await api(`/convites/${token}/aceitar`, {
        method: 'POST',
        body: {
          nome,
          nome_artistico: nomeArtistico,
          instagram,
          data_nascimento: dataNascimento ? new Date(dataNascimento).toISOString() : undefined,
          telefone,
          foto_url: fotoURL,
          tipo_apresentacao: ehParticipante ? tipoApresentacao : undefined,
          dados: ehParticipante ? dados : undefined,
          responsavel_nome: responsavelNome,
          responsavel_contato: responsavelContato,
          autorizacao_responsavel_url: autorizacaoURL,
        },
      })
      onConfirmado()
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao confirmar')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <Card>
        <CardHeader>
          <CardTitle>
            Convite de {rotuloTipo[convite.tipo]} — {convite.evento_titulo}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="nome">Nome</Label>
            <Input id="nome" value={nome} onChange={(e) => setNome(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="nome_artistico">Nome artístico</Label>
            <Input id="nome_artistico" value={nomeArtistico} onChange={(e) => setNomeArtistico(e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="instagram">Instagram</Label>
              <Input id="instagram" value={instagram} onChange={(e) => setInstagram(e.target.value)} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="data_nascimento">Data de nascimento</Label>
              <Input id="data_nascimento" type="date" value={dataNascimento} onChange={(e) => setDataNascimento(e.target.value)} />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="telefone">Telefone</Label>
            <Input id="telefone" value={telefone} onChange={(e) => setTelefone(e.target.value)} />
          </div>

          <UploadImagem id="foto" rotulo="Foto" valor={fotoURL} onEnviado={setFotoURL} />

          {ehParticipante && (
            <>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="tipo_apresentacao">Tipo de apresentação</Label>
                <select
                  id="tipo_apresentacao"
                  className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
                  value={tipoApresentacao}
                  onChange={(e) => {
                    setTipoApresentacao(e.target.value)
                    setDados({})
                  }}
                >
                  <option value="cosplay">Cosplay</option>
                  <option value="danca">Dança</option>
                  <option value="canto">Canto</option>
                  <option value="atuacao">Atuação</option>
                </select>
              </div>
              {camposPorTipo[tipoApresentacao].map((campo) => (
                <div key={campo.chave} className="flex flex-col gap-1.5">
                  <Label htmlFor={campo.chave}>{campo.rotulo}</Label>
                  <Input
                    id={campo.chave}
                    value={dados[campo.chave] ?? ''}
                    onChange={(e) => setDados({ ...dados, [campo.chave]: e.target.value })}
                  />
                </div>
              ))}
              {tipoApresentacao === 'cosplay' && (
                <UploadImagem
                  id="foto_referencia"
                  rotulo="Foto de referência (material oficial, obrigatória)"
                  valor={dados.foto_referencia ?? ''}
                  onEnviado={(url) => setDados({ ...dados, foto_referencia: url })}
                />
              )}
              <UploadAudio
                id="audio"
                rotulo="Áudio/música da apresentação (opcional)"
                valor={dados.audio ?? ''}
                onEnviado={(url) => setDados({ ...dados, audio: url })}
              />
            </>
          )}

          {menorDeIdade && (
            <div className="flex flex-col gap-4 rounded-lg border border-border p-3">
              <p className="text-sm text-muted-foreground">
                Menor de 18 anos — precisa de nome, contato e autorização do responsável.
              </p>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="responsavel_nome">Nome do responsável</Label>
                <Input id="responsavel_nome" value={responsavelNome} onChange={(e) => setResponsavelNome(e.target.value)} />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="responsavel_contato">Contato do responsável</Label>
                <Input id="responsavel_contato" value={responsavelContato} onChange={(e) => setResponsavelContato(e.target.value)} />
              </div>
              <UploadImagem
                id="autorizacao"
                rotulo="Autorização assinada (foto ou scan)"
                valor={autorizacaoURL}
                onEnviado={setAutorizacaoURL}
              />
            </div>
          )}

          {erro && <p className="text-sm text-destructive">{erro}</p>}
          <Button onClick={confirmar} disabled={enviando || !nome}>
            Confirmar
          </Button>
        </CardContent>
      </Card>
    </main>
  )
}
