import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { AvaliarFicha } from '@/components/avaliar-ficha'
import { Card, CardContent } from '@/components/ui/card'
import { Carregando } from '@/components/carregando'

type FichaJurado = {
  id: number
  nome: string
  nome_artistico: string
  instagram: string
  idade: number | null
  foto_url: string
  tipo_apresentacao: string | null
  ordem_apresentacao?: number | null
  dados: Record<string, string>
}

const rotuloTipo: Record<string, string> = {
  cosplay: 'Cosplay',
  danca: 'Dança',
  canto: 'Canto',
  atuacao: 'Atuação',
}

export default function AreaJurado() {
  const { slug } = useParams()
  const { data: fichas, isLoading, error } = useQuery({
    queryKey: ['jurado-participantes', slug],
    queryFn: () => api<FichaJurado[]>(`/eventos/${slug}/participantes`),
  })

  if (isLoading) return <Carregando />
  if (error instanceof ApiError && error.status === 403) {
    return <p className="p-8 text-center text-muted-foreground">Você não é jurado confirmado deste evento.</p>
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-12">
      <h1 className="mb-6 text-2xl font-semibold text-foreground">Participantes</h1>
      {fichas?.length === 0 && <p className="text-muted-foreground">Nenhum participante aprovado ainda.</p>}
      <div className="flex flex-col gap-3">
        {fichas?.map((f) => (
          <Card key={f.id}>
            <CardContent className="flex gap-4 py-4">
              {f.foto_url && <img src={f.foto_url} alt="" className="size-16 shrink-0 rounded-lg object-cover" />}
              <div className="flex-1">
                <p className="font-medium text-foreground">
                  {f.ordem_apresentacao ? `${f.ordem_apresentacao}. ` : ''}
                  {f.nome} {f.nome_artistico && `(${f.nome_artistico})`}
                </p>
                <p className="text-sm text-muted-foreground">
                  {f.tipo_apresentacao && rotuloTipo[f.tipo_apresentacao]}
                  {f.idade != null && ` · ${f.idade} anos`}
                  {f.instagram && ` · ${f.instagram}`}
                </p>
                {f.dados && Object.keys(f.dados).length > 0 && (
                  <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                    {Object.entries(f.dados)
                      .filter(([chave]) => chave !== 'foto_referencia' && chave !== 'audio')
                      .map(([chave, valor]) => (
                        <div key={chave}>
                          <dt className="text-muted-foreground capitalize">{chave.replace(/_/g, ' ')}</dt>
                          <dd className="text-foreground">{valor}</dd>
                        </div>
                      ))}
                  </dl>
                )}
                {f.dados?.foto_referencia && (
                  <img src={f.dados.foto_referencia} alt="Referência" className="mt-2 h-24 rounded-lg object-cover" />
                )}
                {f.dados?.audio && <audio controls src={f.dados.audio} className="mt-2 w-full" />}
                <AvaliarFicha slug={slug!} fichaId={f.id} />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </main>
  )
}
