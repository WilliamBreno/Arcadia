import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Download } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { api, ApiError, urlApi } from '@/lib/api'
import { rotuloFuncaoCert, type CertificadoPublico } from '@/lib/pos-evento'

// Item 9.7: certificados da conta (pela conta ou pelo e-mail do ingresso).
export function MeusCertificados() {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['meus-certificados'], queryFn: () => api<CertificadoPublico[]>('/me/certificados') })
  const [editando, setEditando] = useState<{ codigo: string; nome: string } | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  if (!data || data.length === 0) return null

  const salvar = async () => {
    if (!editando) return
    setErro(null)
    try {
      await api(`/me/certificados/${encodeURIComponent(editando.codigo)}/nome`, { method: 'POST', body: { nome: editando.nome } })
      setEditando(null)
      queryClient.invalidateQueries({ queryKey: ['meus-certificados'] })
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível corrigir.')
    }
  }

  return (
    <section aria-labelledby="titulo-certificados" className="mt-10">
      <h2 id="titulo-certificados" className="mb-3 text-lg font-semibold text-foreground">
        Meus certificados
      </h2>
      <ul className="flex flex-col gap-3">
        {data.map((c) => (
          <li key={c.codigo} className="flex flex-col gap-2 rounded-xl border border-border p-4">
            <div>
              <p className="font-medium text-foreground">{c.evento_titulo}</p>
              <p className="text-sm text-muted-foreground">
                {c.nome} · {rotuloFuncaoCert[c.funcao]}
                {c.detalhe ? ` · ${c.detalhe}` : ''}
                {c.evento_data ? ` · ${c.evento_data}` : ''}
              </p>
            </div>
            {editando?.codigo === c.codigo ? (
              <div className="flex flex-col gap-2 sm:flex-row">
                <Input aria-label="Nome no certificado" value={editando.nome} onChange={(e) => setEditando({ ...editando, nome: e.target.value })} maxLength={160} />
                <Button size="sm" onClick={salvar} disabled={editando.nome.trim().length < 3}>
                  Salvar
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setEditando(null)}>
                  Cancelar
                </Button>
              </div>
            ) : (
              <div className="flex flex-wrap gap-2">
                <a href={urlApi(`/certificados/${c.codigo}/pdf`)} download className={buttonVariants({ size: 'sm' })}>
                  <Download className="size-4" aria-hidden="true" />
                  Baixar PDF
                </a>
                <Link to={`/certificado/${c.codigo}`} className={buttonVariants({ size: 'sm', variant: 'outline' })}>
                  Validação
                </Link>
                {c.pode_corrigir && (
                  <Button size="sm" variant="ghost" onClick={() => setEditando({ codigo: c.codigo, nome: c.nome })}>
                    Corrigir o nome (uma vez)
                  </Button>
                )}
              </div>
            )}
          </li>
        ))}
      </ul>
      {erro && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {erro}
        </p>
      )}
    </section>
  )
}
