import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { ApiError } from '@/lib/api'
import { decidirAutorizacaoIntegrante, verAutorizacaoIntegrante } from '@/lib/artistas'
import { Button } from '@/components/ui/button'
import { Carregando } from '@/components/carregando'

export default function AutorizarIntegrante() {
  const [params] = useSearchParams()
  const token = params.get('t') ?? ''
  const queryClient = useQueryClient()
  const chave = ['autorizacao-integrante', token]
  const { data, isLoading, error } = useQuery({
    queryKey: chave,
    queryFn: () => verAutorizacaoIntegrante(token),
    enabled: !!token,
    retry: false,
  })
  const [foto, setFoto] = useState(false)
  const [enviando, setEnviando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)

  useEffect(() => {
    document.title = 'Autorização de integrante | Evve'
    const meta = document.createElement('meta')
    meta.name = 'robots'
    meta.content = 'noindex, nofollow'
    document.head.appendChild(meta)
    return () => meta.remove()
  }, [])
  useEffect(() => {
    if (data) setFoto(data.tem_foto && data.autoriza_foto)
  }, [data])

  const decidir = async (autorizar: boolean) => {
    setErro(null)
    setAviso(null)
    setEnviando(true)
    try {
      const resposta = await decidirAutorizacaoIntegrante(token, autorizar, autorizar && !!data?.tem_foto && foto)
      queryClient.setQueryData(chave, resposta)
      setAviso(autorizar ? 'Autorização salva.' : 'A pessoa não será mostrada no perfil do grupo.')
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
    } finally {
      setEnviando(false)
    }
  }

  if (isLoading) return <Carregando />
  if (!token || error || !data) {
    return (
      <main className="mx-auto max-w-lg px-4 py-16">
        <h1 className="text-2xl font-bold text-foreground">Link inválido</h1>
        <p role="alert" className="mt-2 text-destructive">
          Este link não é válido ou não está mais disponível. Se o e-mail do responsável foi trocado, use o novo link enviado.
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <h1 className="text-2xl font-bold text-foreground">Autorizar integrante</h1>
      <p className="mt-2 text-foreground">
        Olá{data.responsavel ? `, ${data.responsavel.split(' ')[0]}` : ''}. Um jovem pelo qual você é responsável quer aparecer
        como integrante do grupo <strong>{data.grupo_nome}</strong> com o nome artístico <strong>{data.nome_artistico}</strong> ({data.funcao}).
      </p>
      <div className="mt-6 rounded-xl border border-border p-4 text-sm text-foreground">
        <p className="font-medium">O que aparece</p>
        <ul className="mt-1 list-disc pl-5">
          <li>nome artístico: {data.nome_artistico};</li>
          <li>função: {data.funcao};</li>
          <li>anos de experiência;</li>
          <li>a foto, só se você autorizar abaixo.</li>
        </ul>
        <p className="mt-3 font-medium">O que nunca aparece</p>
        <p>Nome real, idade, cidade, redes sociais ou contato.</p>
      </div>
      {data.tem_foto && (
        <label className="mt-4 flex items-start gap-2 text-sm text-foreground">
          <input type="checkbox" className="mt-0.5" checked={foto} disabled={enviando} onChange={(e) => setFoto(e.target.checked)} />
          Mostrar a foto
        </label>
      )}
      <p role="status" className="mt-4 text-sm font-medium text-foreground">
        {data.autorizado
          ? `Situação: autorizado${data.tem_foto && data.autoriza_foto ? ', com foto' : ', sem foto'}.`
          : 'Situação: não autorizado.'}
      </p>
      {aviso && <p role="status" className="mt-2 text-sm text-muted-foreground">{aviso}</p>}
      {erro && <p role="alert" className="mt-2 text-sm text-destructive">{erro}</p>}
      <div className="evve-actions mt-4 flex flex-wrap gap-2">
        <Button onClick={() => decidir(true)} disabled={enviando}>Autorizar</Button>
        <Button variant="outline" onClick={() => decidir(false)} disabled={enviando}>
          {data.autorizado ? 'Desautorizar' : 'Não autorizar'}
        </Button>
      </div>
      <p className="mt-6 text-xs text-muted-foreground">
        Guarde este e-mail: pelo mesmo link você pode desautorizar quando quiser.
      </p>
    </main>
  )
}
