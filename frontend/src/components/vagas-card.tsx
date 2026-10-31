import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'
import { modalidades, obterVagasOrg, type ResumoVagas, type SituacaoVagas } from '@/lib/vagas'

type Campos = Record<'jurados' | 'participantes' | 'cosplay' | 'danca' | 'canto' | 'atuacao', string>

const paraCampos = (r: ResumoVagas): Campos => {
  const v = (s: SituacaoVagas) => (s.limite === null ? '' : String(s.limite))
  return {
    jurados: v(r.jurados),
    participantes: v(r.participantes),
    cosplay: v(r.por_tipo.cosplay),
    danca: v(r.por_tipo.danca),
    canto: v(r.por_tipo.canto),
    atuacao: v(r.por_tipo.atuacao),
  }
}

function Ocupacao({ s }: { s: SituacaoVagas }) {
  if (s.limite === null) return <span className="text-xs text-muted-foreground">{s.ocupadas} até agora · sem limite</span>
  return (
    <span className={`text-xs ${s.esgotado ? 'font-medium text-foreground' : 'text-muted-foreground'}`}>
      {s.ocupadas} de {s.limite} {s.esgotado ? '· preenchidas' : `· restam ${s.limite - s.ocupadas}`}
    </span>
  )
}

// Item 11.6: o organizador limita quantos jurados e participantes entram,
// por ordem de chegada. Em branco = sem limite.
export function VagasCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['org-vagas', eventoId], queryFn: () => obterVagasOrg(eventoId) })
  const [campos, setCampos] = useState<Campos | null>(null)
  const [lista, setLista] = useState<boolean | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [salvo, setSalvo] = useState(false)
  const [salvando, setSalvando] = useState(false)

  if (!data) return null
  const atual = campos ?? paraCampos(data)
  const listaAtual = lista ?? data.lista_espera
  const mudar = (k: keyof Campos, valor: string) => {
    setSalvo(false)
    setCampos({ ...atual, [k]: valor.replace(/\D/g, '').slice(0, 5) })
  }

  const salvar = async () => {
    setErro(null)
    setSalvando(true)
    const n = (s: string) => (s.trim() === '' ? null : Number(s))
    try {
      const r = await api<ResumoVagas>(`/org/eventos/${eventoId}/vagas`, {
        method: 'PUT',
        body: {
          jurados: n(atual.jurados),
          participantes: n(atual.participantes),
          cosplay: n(atual.cosplay),
          danca: n(atual.danca),
          canto: n(atual.canto),
          atuacao: n(atual.atuacao),
          lista_espera: listaAtual,
        },
      })
      queryClient.setQueryData(['org-vagas', eventoId], r)
      setCampos(null)
      setLista(null)
      setSalvo(true)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
    } finally {
      setSalvando(false)
    }
  }

  const campo = (k: keyof Campos, rotulo: string, s: SituacaoVagas) => (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={`vagas-${k}`}>{rotulo}</Label>
      <Input id={`vagas-${k}`} inputMode="numeric" placeholder="Sem limite" value={atual[k]} onChange={(e) => mudar(k, e.target.value)} />
      <Ocupacao s={s} />
    </div>
  )

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Vagas de jurados e participantes</CardTitle>
        <p className="text-sm text-muted-foreground">
          Opcional. Quem chega primeiro entra; quando as vagas acabam, o link de convite e a inscrição avisam que estão preenchidas. Deixe em branco
          para não limitar.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {campo('jurados', 'Jurados', data.jurados)}
          {campo('participantes', 'Participantes (total)', data.participantes)}
        </div>
        <fieldset className="flex flex-col gap-3">
          <legend className="mb-1 text-sm font-medium text-foreground">Por modalidade</legend>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">{modalidades.map((m) => <div key={m.valor}>{campo(m.valor, m.nome, data.por_tipo[m.valor])}</div>)}</div>
        </fieldset>
        <label className="flex items-start gap-2 text-sm text-foreground">
          <input
            type="checkbox"
            className="mt-0.5"
            checked={listaAtual}
            onChange={(e) => {
              setSalvo(false)
              setLista(e.target.checked)
            }}
          />
          <span>
            Lista de espera na inscrição aberta
            <span className="block text-xs text-muted-foreground">
              Com as vagas preenchidas, quem se inscreve fica na lista. Se alguém sair, você aprova quem estiver na lista.
            </span>
          </span>
        </label>
        <p className="text-xs text-muted-foreground">Baixar um limite não tira ninguém que já entrou: só fecha novas entradas.</p>
        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
        <div className="flex items-center gap-3">
          <Button size="sm" onClick={salvar} disabled={salvando}>
            {salvando ? 'Salvando…' : 'Salvar vagas'}
          </Button>
          {salvo && (
            <span role="status" className="text-sm text-muted-foreground">
              Salvo.
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
