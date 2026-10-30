import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useParams } from 'react-router-dom'

import { obterTelao, rotuloTipo } from '@/lib/palco'

// Telão (item 8.3): tela cheia para TV ou projetor, aberta por link secreto.
// Atualiza sozinha a cada 3 s. Fotos: referência sempre; participante só 18+.
export default function Telao() {
  const { token = '' } = useParams()
  const { data, error } = useQuery({
    queryKey: ['telao', token],
    queryFn: () => obterTelao(token),
    refetchInterval: 3000,
    retry: 1,
  })

  useEffect(() => {
    document.title = 'Telão | Evve'
    const meta = document.createElement('meta')
    meta.name = 'robots'
    meta.content = 'noindex, nofollow'
    document.head.appendChild(meta)
    return () => meta.remove()
  }, [])

  const fundo = { background: '#0b0b0c', color: '#f5f5f4', minHeight: '100svh' }

  if (error) {
    return (
      <main style={fundo} className="flex items-center justify-center p-8 text-center">
        <p className="text-2xl">Link do telão inválido ou trocado. Peça o link novo ao organizador.</p>
      </main>
    )
  }
  if (!data) return <main style={fundo} />

  const atual = data.modo === 'apresentacao' ? data.atual : undefined

  return (
    <main style={fundo} className="flex flex-col p-6 sm:p-10">
      <p className="text-lg opacity-70 sm:text-2xl">{data.evento}</p>

      {data.modo === 'resultado' ? (
        <section className="flex flex-1 flex-col justify-center gap-10" aria-label="Resultado">
          {!data.resultado || Object.keys(data.resultado).length === 0 ? (
            <p className="text-center text-4xl font-bold sm:text-6xl">Resultado em instantes</p>
          ) : (
            <div className="grid gap-10 md:grid-cols-2">
              {Object.entries(data.resultado).map(([tipo, podio]) => (
                <div key={tipo}>
                  <h2 className="mb-4 text-2xl font-semibold opacity-80 sm:text-3xl">{rotuloTipo[tipo] ?? 'Resultado'}</h2>
                  <ol className="flex flex-col gap-3">
                    {podio.map((p) => (
                      <li key={`${tipo}-${p.posicao}-${p.nome}`} className="flex items-baseline gap-4">
                        <span className="w-16 text-4xl font-bold tabular-nums sm:text-6xl">{p.posicao}º</span>
                        <span className="text-3xl font-semibold sm:text-5xl">{p.nome}</span>
                      </li>
                    ))}
                  </ol>
                  {data.publico?.[tipo]?.[0] && (
                    <p className="mt-4 text-xl opacity-80 sm:text-2xl">
                      Prêmio do público: <strong>{data.publico[tipo][0].nome}</strong>
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}
        </section>
      ) : atual ? (
        <section className="flex flex-1 flex-col justify-center gap-8 md:flex-row md:items-center" aria-label="No palco">
          {(atual.foto_referencia || atual.foto_participante) && (
            <div className="flex shrink-0 gap-4">
              {atual.foto_referencia && (
                <figure className="flex flex-col items-center gap-2">
                  <img src={atual.foto_referencia} alt="" className="h-64 w-48 rounded-2xl object-cover sm:h-96 sm:w-72" />
                  <figcaption className="text-sm opacity-60">Referência</figcaption>
                </figure>
              )}
              {atual.foto_participante && (
                <figure className="flex flex-col items-center gap-2">
                  <img src={atual.foto_participante} alt="" className="h-64 w-48 rounded-2xl object-cover sm:h-96 sm:w-72" />
                  <figcaption className="text-sm opacity-60">No palco</figcaption>
                </figure>
              )}
            </div>
          )}
          <div className="min-w-0">
            <p className="text-xl uppercase tracking-widest opacity-60 sm:text-2xl">{rotuloTipo[atual.tipo] ?? 'No palco'}</p>
            <h1 className="mt-2 text-5xl font-bold text-balance sm:text-8xl">{atual.nome}</h1>
            {(atual.personagem || atual.obra) && (
              <p className="mt-4 text-3xl opacity-90 sm:text-5xl">
                {atual.personagem}
                {atual.obra ? <span className="opacity-60"> · {atual.obra}</span> : null}
              </p>
            )}
          </div>
        </section>
      ) : (
        <section className="flex flex-1 flex-col items-center justify-center text-center" aria-label="Intervalo">
          <p className="text-5xl font-bold sm:text-7xl">Já já tem mais!</p>
        </section>
      )}

      {data.modo !== 'resultado' && data.proximos.length > 0 && (
        <footer className="mt-6 border-t pt-4" style={{ borderColor: 'rgba(255,255,255,0.15)' }}>
          <p className="text-lg opacity-70 sm:text-2xl">
            A seguir: {data.proximos.map((p) => (p.personagem ? `${p.nome} (${p.personagem})` : p.nome)).join(' · ')}
          </p>
        </footer>
      )}
    </main>
  )
}
