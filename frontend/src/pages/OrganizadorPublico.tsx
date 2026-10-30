import { VoltarBotao } from '@/components/voltar-link'
import { SeloVerificado } from '@/components/selo-verificado'
import { SeloPro } from '@/components/selo-pro'
import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import type { OrganizadorDetalhe } from '@/lib/publico'
import { EventoCard } from '@/components/evento-card'
import { BotaoSeguir } from '@/components/botao-seguir'
import { Carregando } from '@/components/carregando'
import { Estrelas } from '@/components/estrelas'
import { formatarMedia } from '@/lib/avaliacoes'

export default function OrganizadorPublico() {
  const { slug } = useParams()
  const { data, isLoading, error } = useQuery({
    queryKey: ['organizador-publico', slug],
    queryFn: () => api<OrganizadorDetalhe>(`/organizadores/${slug}`),
  })

  if (isLoading) return <Carregando />
  if (error instanceof ApiError && error.status === 404) {
    return <p className="p-8 text-center text-muted-foreground">Organizador não encontrado.</p>
  }
  if (!data) return null

  return (
    <main className="mx-auto max-w-5xl px-4 py-12">
      <VoltarBotao fallback="/" />
      <div className="mb-8 flex flex-col items-center text-center">
        {data.organizador.logo_url && (
          <img src={data.organizador.logo_url} alt="" className="mb-4 size-20 rounded-full object-cover" />
        )}
        <h1 className="text-2xl font-semibold text-foreground">{data.organizador.nome_publico}</h1>
        {data.organizador.verificado && <SeloVerificado className="mt-2" />}
        {data.organizador.pro && <SeloPro className="mt-2 ml-1" />}
        <div className="mt-4 flex items-center gap-3">
          <BotaoSeguir tipo="organizador" alvo={String(data.organizador.id)} rotulo={data.organizador.nome_publico} />
          {data.seguidores > 0 && (
            <span className="text-sm text-muted-foreground">
              {data.seguidores} {data.seguidores === 1 ? 'seguidor' : 'seguidores'}
            </span>
          )}
        </div>
        {data.avaliacoes?.mostrar_media && (
          <p className="mt-3 flex items-center gap-2 text-sm text-muted-foreground">
            <span className="font-semibold tabular-nums text-foreground">{formatarMedia(data.avaliacoes.media)}</span>
            <Estrelas nota={data.avaliacoes.media} tamanho={14} />
            <span>
              {data.avaliacoes.total} avaliações de quem foi aos eventos
            </span>
          </p>
        )}
        {data.organizador.descricao && (
          <p className="mt-2 max-w-lg text-muted-foreground">{data.organizador.descricao}</p>
        )}
      </div>

      <h2 className="mb-4 text-xl font-semibold text-foreground">Eventos</h2>
      {data.eventos.length === 0 && <p className="text-muted-foreground">Nenhum evento publicado ainda.</p>}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
        {data.eventos.map((evento) => (
          <EventoCard key={evento.slug} evento={evento} />
        ))}
      </div>
    </main>
  )
}
