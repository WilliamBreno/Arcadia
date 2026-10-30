import { useQuery, useQueryClient } from '@tanstack/react-query'
import { BadgeCheck, Download } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'

import { Carregando } from '@/components/carregando'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError, urlApi } from '@/lib/api'
import { cargaHoraria, rotuloFuncaoCert, type CertificadoPublico } from '@/lib/pos-evento'

// Item 9.7: validação pública do certificado (/certificado/CODIGO). Com o
// link do e-mail (?t=...), a pessoa corrige o nome uma vez.
export default function Certificado() {
  const { codigo = '' } = useParams()
  const [params] = useSearchParams()
  const token = params.get('t') ?? ''
  const queryClient = useQueryClient()
  const chave = ['certificado', codigo, token]
  const { data, isLoading, error } = useQuery({
    queryKey: chave,
    retry: false,
    queryFn: () => api<CertificadoPublico>(`/certificados/${encodeURIComponent(codigo)}${token ? `?t=${encodeURIComponent(token)}` : ''}`, { semAuth: true }),
  })
  const [nome, setNome] = useState('')
  const [editando, setEditando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  if (isLoading) return <Carregando />
  if (!data) {
    return (
      <main className="mx-auto max-w-md px-4 py-16 text-center">
        <h1 className="text-xl font-semibold text-foreground">Certificado não encontrado</h1>
        <p className="mt-2 text-muted-foreground">
          {error instanceof ApiError && error.status !== 404 ? error.message : 'Confira o código impresso no certificado. Ele tem 12 letras e números.'}
        </p>
      </main>
    )
  }

  const corrigir = async () => {
    setErro(null)
    try {
      await api(`/certificados/${encodeURIComponent(codigo)}/nome`, { method: 'POST', body: { t: token, nome }, semAuth: true })
      setEditando(false)
      queryClient.invalidateQueries({ queryKey: chave })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível corrigir o nome.')
    }
  }

  return (
    <main className="mx-auto max-w-lg px-4 py-12">
      <div className="flex items-center gap-2 text-sm font-medium text-green-700 dark:text-green-400">
        <BadgeCheck className="size-5" aria-hidden="true" />
        Certificado válido, emitido pelo Evve
      </div>
      <h1 className="mt-3 text-3xl font-semibold text-foreground">{data.nome}</h1>
      <dl className="mt-6 grid grid-cols-3 gap-x-4 gap-y-3 text-sm">
        <dt className="text-muted-foreground">Evento</dt>
        <dd className="col-span-2 text-foreground">
          <Link to={`/e/${data.evento_slug}`} className="underline underline-offset-4">
            {data.evento_titulo}
          </Link>
        </dd>
        {data.evento_data && (
          <>
            <dt className="text-muted-foreground">Data</dt>
            <dd className="col-span-2 text-foreground">{data.evento_data}</dd>
          </>
        )}
        <dt className="text-muted-foreground">Como</dt>
        <dd className="col-span-2 text-foreground">
          {rotuloFuncaoCert[data.funcao]}
          {data.detalhe ? ` · ${data.detalhe}` : ''}
        </dd>
        {!!data.carga_horaria_min && (
          <>
            <dt className="text-muted-foreground">Carga horária</dt>
            <dd className="col-span-2 text-foreground">{cargaHoraria(data.carga_horaria_min)}</dd>
          </>
        )}
        {data.organizador && (
          <>
            <dt className="text-muted-foreground">Organização</dt>
            <dd className="col-span-2 text-foreground">{data.organizador}</dd>
          </>
        )}
        <dt className="text-muted-foreground">Código</dt>
        <dd className="col-span-2 font-mono text-foreground">{data.codigo}</dd>
      </dl>

      <a href={urlApi(`/certificados/${encodeURIComponent(data.codigo)}/pdf`)} className={`${buttonVariants()} mt-8 w-full sm:w-auto`} download>
        <Download className="size-4" aria-hidden="true" />
        Baixar PDF
      </a>

      {data.pode_corrigir && (
        <section aria-labelledby="titulo-nome" className="mt-8 rounded-xl border border-border p-4">
          <h2 id="titulo-nome" className="text-base font-semibold text-foreground">
            O nome está certo?
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Dá para corrigir uma vez (acento, sobrenome, nome social). Depois disso o nome fica fixo e é ele que o código valida.
          </p>
          {editando ? (
            <div className="mt-3 flex flex-col gap-2">
              <Label htmlFor="nome-certificado">Nome no certificado</Label>
              <Input id="nome-certificado" value={nome} onChange={(e) => setNome(e.target.value)} maxLength={160} />
              <div className="flex gap-2">
                <Button onClick={corrigir} disabled={nome.trim().length < 3}>
                  Salvar nome
                </Button>
                <Button variant="ghost" onClick={() => setEditando(false)}>
                  Cancelar
                </Button>
              </div>
            </div>
          ) : (
            <Button
              variant="outline"
              className="mt-3"
              onClick={() => {
                setNome(data.nome)
                setEditando(true)
              }}
            >
              Corrigir o nome
            </Button>
          )}
          {erro && (
            <p role="alert" className="mt-2 text-sm text-destructive">
              {erro}
            </p>
          )}
        </section>
      )}
    </main>
  )
}
