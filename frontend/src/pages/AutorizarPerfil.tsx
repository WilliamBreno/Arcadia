import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { api, ApiError } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Carregando } from '@/components/carregando'

type Pedido = {
  nome_artistico: string
  bio: string
  tem_foto: boolean
  autorizado: boolean
  autoriza_foto: boolean
  responsavel: string
}

// Página do responsável (item 8.1): autoriza (ou não) o perfil público de
// um menor de 12 a 17 anos, decide sobre a foto e pode desautorizar depois.
export default function AutorizarPerfil() {
  const [params] = useSearchParams()
  const token = params.get('t') ?? ''
  const queryClient = useQueryClient()
  const chave = ['autorizacao-perfil', token]
  const { data, isLoading, error } = useQuery({
    queryKey: chave,
    queryFn: () => api<Pedido>(`/talentos/autorizacao?t=${encodeURIComponent(token)}`, { semAuth: true }),
    retry: false,
  })
  const [foto, setFoto] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  useEffect(() => {
    document.title = 'Autorização de perfil | Evve'
    const meta = document.createElement('meta')
    meta.name = 'robots'
    meta.content = 'noindex, nofollow'
    document.head.appendChild(meta)
    return () => meta.remove()
  }, [])
  useEffect(() => {
    if (data) setFoto(data.autoriza_foto)
  }, [data])

  const decidir = async (autorizar: boolean) => {
    setErro(null)
    setEnviando(true)
    try {
      const r = await api<Pedido>('/talentos/autorizacao', { method: 'POST', body: { t: token, autorizar, foto }, semAuth: true })
      queryClient.setQueryData(chave, r)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
    } finally {
      setEnviando(false)
    }
  }

  if (isLoading) return <Carregando />
  if (error || !data) {
    return (
      <main className="mx-auto max-w-lg px-4 py-16">
        <h1 className="text-2xl font-bold text-foreground">Link inválido</h1>
        <p className="mt-2 text-muted-foreground">
          Este link não vale mais. Se o e-mail do responsável foi trocado, um link novo foi enviado.
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <h1 className="text-2xl font-bold text-foreground">Autorizar perfil público</h1>
      <p className="mt-2 text-foreground">
        Olá{data.responsavel ? `, ${data.responsavel.split(' ')[0]}` : ''}. O perfil <strong>{data.nome_artistico}</strong> pediu
        para ficar público no Evve.
      </p>

      <div className="mt-6 rounded-xl border border-border p-4 text-sm">
        <p className="font-medium text-foreground">O que aparece</p>
        <ul className="mt-1 list-disc pl-5 text-foreground">
          <li>nome artístico: {data.nome_artistico};</li>
          <li>o texto "sobre": {data.bio ? `"${data.bio}"` : '(vazio)'};</li>
          <li>personagens e colocações nos concursos;</li>
          <li>a foto, só se você marcar abaixo.</li>
        </ul>
        <p className="mt-3 font-medium text-foreground">O que nunca aparece</p>
        <p className="text-foreground">
          Nome real, idade, cidade, redes sociais ou contato. O perfil também não aparece no Google.
        </p>
      </div>

      {data.tem_foto && (
        <label className="mt-4 flex items-start gap-2 text-sm text-foreground">
          <input type="checkbox" className="mt-0.5" checked={foto} onChange={(e) => setFoto(e.target.checked)} />
          Autorizo mostrar a foto no perfil.
        </label>
      )}

      <p role="status" className="mt-4 text-sm font-medium text-foreground">
        {data.autorizado
          ? `Situação: autorizado${data.autoriza_foto ? ', com foto' : ', sem foto'}.`
          : 'Situação: ainda não autorizado.'}
      </p>
      {erro && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {erro}
        </p>
      )}
      <div className="mt-4 flex flex-wrap gap-2">
        <Button onClick={() => decidir(true)} disabled={enviando}>
          {data.autorizado ? 'Salvar' : 'Autorizar o perfil'}
        </Button>
        {data.autorizado ? (
          <Button variant="outline" onClick={() => decidir(false)} disabled={enviando}>
            Desautorizar e esconder o perfil
          </Button>
        ) : null}
      </div>
      <p className="mt-6 text-xs text-muted-foreground">
        Guarde este e-mail: pelo mesmo link você pode desautorizar quando quiser.
      </p>
    </main>
  )
}
