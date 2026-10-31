import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'

import { Carregando } from '@/components/carregando'
import { GradeColecoes } from '@/components/grade-colecoes'
import { VoltarBotao } from '@/components/voltar-link'
import { listarGradeColecoes } from '@/lib/descoberta'

// Item 11.7: /colecoes — todos os temas da época e coleções com eventos.
export default function Colecoes() {
  const { data, isLoading } = useQuery({ queryKey: ['colecoes-grade'], queryFn: listarGradeColecoes })

  useEffect(() => {
    document.title = 'Coleções | Evve'
  }, [])

  return (
    <main className="mx-auto max-w-5xl px-4 py-10">
      <VoltarBotao fallback="/" />
      <h1 className="mt-2 text-3xl font-bold text-foreground sm:text-4xl">Coleções</h1>
      <p className="mt-2 text-muted-foreground">Eventos por tema e por época, do Halloween ao Réveillon.</p>
      <div className="mt-8">
        {isLoading ? (
          <Carregando />
        ) : data && data.length > 0 ? (
          <GradeColecoes colecoes={data} />
        ) : (
          <p className="rounded-xl border border-border p-6 text-muted-foreground">Nenhuma coleção com eventos agora. Volte em breve.</p>
        )}
      </div>
    </main>
  )
}
