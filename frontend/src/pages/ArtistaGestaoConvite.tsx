import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Carregando } from '@/components/carregando'

export default function ArtistaGestaoConvite() {
  const { token = '' } = useParams()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { data, error, isLoading } = useQuery({ queryKey: ['convite-gestao', token], queryFn: () => api<{ grupo_nome: string; papel: string; antigo_cogestor: boolean; expira_em: string }>(`/me/artistas-gestao/${token}`), retry: false })
  const [erro, setErro] = useState('')
  const [ocupado, setOcupado] = useState(false)
  const [recusado, setRecusado] = useState(false)
  const decidir = async (aceitar: boolean) => {
    setErro(''); setOcupado(true)
    try {
      const r = await api<{ grupo_id: number }>(`/me/artistas-gestao/${token}`, { method: 'POST', body: { aceitar } })
      await qc.invalidateQueries({ queryKey: ['meus-artistas'] })
      if (aceitar) navigate(`/meus-artistas/${r.grupo_id}`)
      else setRecusado(true)
    } catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível responder.') } finally { setOcupado(false) }
  }
  if (isLoading) return <Carregando />
  return <main className="mx-auto flex max-w-lg flex-col gap-4 px-4 py-16">
    <h1 className="text-2xl font-bold">{recusado ? 'Convite recusado' : 'Convite de gestão'}</h1>
    {error && <p role="alert">{error instanceof ApiError ? error.message : 'Convite indisponível.'}</p>}
    {data && !recusado && <>
      <p>Você foi convidado para ser {data.papel === 'lider' ? 'líder' : 'cogestor'} de <strong>{data.grupo_nome}</strong>.</p>
      <p className="text-sm text-muted-foreground">{data.papel === 'lider' ? `Você assumirá a liderança. O líder atual ${data.antigo_cogestor ? 'continuará como cogestor' : 'sairá da gestão'}.` : 'Você poderá editar perfil, integrantes, portfólio e agenda, usando sua própria conta e senha.'}</p>
      <p className="text-sm text-muted-foreground">Válido até {new Date(data.expira_em).toLocaleDateString('pt-BR')}.</p>
      <div className="evve-actions flex flex-wrap gap-2"><Button disabled={ocupado} onClick={() => decidir(true)}>Aceitar convite</Button><Button variant="outline" disabled={ocupado} onClick={() => decidir(false)}>Recusar</Button></div>
    </>}
    {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
    <Link to="/meus-artistas" className="text-sm underline">Meus artistas</Link>
  </main>
}
