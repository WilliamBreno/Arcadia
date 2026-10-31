import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { VoltarLink } from '@/components/voltar-link'
import { api, ApiError } from '@/lib/api'
import { formatarCentavos } from '@/lib/evento'
import type { EventoDetalhe } from '@/lib/publico'
import { modalidades, obterVagasPublicas, restantes, type Modalidade } from '@/lib/vagas'
import { UploadAudio } from '@/components/upload-audio'
import { UploadImagem } from '@/components/upload-imagem'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

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

function calcularIdade(dataNascimento: string): number | null {
  if (!dataNascimento) return null
  const nascimento = new Date(dataNascimento)
  const hoje = new Date()
  let idade = hoje.getFullYear() - nascimento.getFullYear()
  if (
    hoje.getMonth() < nascimento.getMonth() ||
    (hoje.getMonth() === nascimento.getMonth() && hoje.getDate() < nascimento.getDate())
  ) {
    idade--
  }
  return idade
}

export default function Inscricao() {
  const { slug } = useParams()
  const navigate = useNavigate()

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

  // Item 8.4: inscrição paga (avisa o valor antes de enviar a ficha).
  const { data: detalhe } = useQuery({
    queryKey: ['evento-publico', slug],
    queryFn: () => api<EventoDetalhe>(`/eventos/${slug}`),
  })
  const valorInscricao = detalhe?.evento.inscricao_talento_centavos ?? 0

  // Item 11.6: vagas por ordem de chegada.
  const { data: vagas } = useQuery({ queryKey: ['vagas-publicas', slug], queryFn: () => obterVagasPublicas(slug!), retry: false })
  const tipoLotado = (t: string) => !!vagas && (vagas.participantes.esgotado || !!vagas.por_tipo[t as Modalidade]?.esgotado)
  const semVaga = tipoLotado(tipoApresentacao)
  const bloqueado = semVaga && !vagas?.lista_espera
  const restamTotal = vagas ? restantes(vagas.participantes) : null
  const restamTipo = vagas ? restantes(vagas.por_tipo[tipoApresentacao as Modalidade] ?? { limite: null, ocupadas: 0, esgotado: false }) : null
  const restam = restamTotal === null ? restamTipo : restamTipo === null ? restamTotal : Math.min(restamTotal, restamTipo)

  const idade = calcularIdade(dataNascimento)
  const menorDeIdade = idade !== null && idade < 18

  const enviar = async () => {
    setErro(null)
    setEnviando(true)
    try {
      await api(`/eventos/${slug}/inscricao`, {
        method: 'POST',
        body: {
          nome,
          nome_artistico: nomeArtistico,
          instagram,
          data_nascimento: dataNascimento ? new Date(dataNascimento).toISOString() : undefined,
          telefone,
          foto_url: fotoURL,
          tipo_apresentacao: tipoApresentacao,
          dados,
          responsavel_nome: responsavelNome,
          responsavel_contato: responsavelContato,
          autorizacao_responsavel_url: autorizacaoURL,
        },
      })
      navigate(`/e/${slug}#minha-inscricao`, { replace: true })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Erro ao enviar inscrição')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <VoltarLink to={`/e/${slug}`}>Voltar ao evento</VoltarLink>
      <Card>
        <CardHeader>
          <CardTitle>Inscrição de participante</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <p className="text-sm text-muted-foreground">
            Sua inscrição fica pendente até o organizador aprovar.
            {valorInscricao > 0 && (
              <>
                {' '}
                A inscrição custa <strong className="text-foreground">{formatarCentavos(valorInscricao)}</strong> + taxa de serviço. Você só paga se for
                aprovada, e tem 48 horas para pagar.
              </>
            )}
          </p>

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
              {modalidades.map((m) => (
                <option key={m.valor} value={m.valor}>
                  {m.nome}
                  {tipoLotado(m.valor) ? ' (vagas preenchidas)' : ''}
                </option>
              ))}
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

          {tipoApresentacao && (
            <UploadAudio
              id="audio"
              rotulo="Áudio/música da apresentação (opcional)"
              valor={dados.audio ?? ''}
              onEnviado={(url) => setDados({ ...dados, audio: url })}
            />
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

          {semVaga ? (
            <p role="status" className="rounded-lg bg-muted p-3 text-sm text-foreground">
              {vagas?.participantes.esgotado ? 'As vagas de participante deste evento estão preenchidas.' : 'As vagas desta modalidade estão preenchidas.'}
              {vagas?.lista_espera ? ' Você pode entrar na lista de espera: se alguém sair, o organizador chama quem estiver na lista.' : ''}
            </p>
          ) : (
            restam !== null && (
              <p className="text-sm text-muted-foreground">
                {restam === 1 ? 'Resta 1 vaga' : `Restam ${restam} vagas`} {vagas?.por_tipo[tipoApresentacao as Modalidade]?.limite != null ? 'nesta modalidade' : ''}, por
                ordem de chegada.
              </p>
            )
          )}
          {erro && <p className="text-sm text-destructive">{erro}</p>}
          <Button onClick={enviar} disabled={enviando || !nome || bloqueado}>
            {semVaga && vagas?.lista_espera ? 'Entrar na lista de espera' : 'Enviar inscrição'}
          </Button>
        </CardContent>
      </Card>
    </main>
  )
}
