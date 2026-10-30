import { Users } from 'lucide-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'

type TipoGrupo = { id: number; nome: string; ativo: boolean; esgotado?: boolean }

// Item 10.3: "Comprar em grupo" — segura os ingressos por até 24 h e cada
// amigo paga a sua parte pelo link.
export function CompraGrupo({
  slug,
  tipos,
  logado,
  pedirLogin,
}: {
  slug: string
  tipos: TipoGrupo[]
  logado: boolean
  pedirLogin: () => void
}) {
  const navigate = useNavigate()
  const opcoes = tipos.filter((t) => t.ativo && !t.esgotado)
  const [aberto, setAberto] = useState(false)
  const [tipo, setTipo] = useState('')
  const [qtd, setQtd] = useState('4')
  const [erro, setErro] = useState<string | null>(null)
  const [criando, setCriando] = useState(false)
  if (opcoes.length === 0) return null

  const criar = async () => {
    if (!logado) {
      pedirLogin()
      return
    }
    setErro(null)
    setCriando(true)
    try {
      const g = await api<{ token: string }>(`/eventos/${slug}/grupos`, {
        method: 'POST',
        body: { tipo_ingresso_id: Number(tipo || opcoes[0].id), quantidade: Number(qtd) },
      })
      navigate(`/grupo/${g.token}`)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível criar o grupo.')
    } finally {
      setCriando(false)
    }
  }

  return (
    <section aria-labelledby="titulo-grupo" className="mt-6 rounded-xl border border-dashed border-border p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-start gap-3">
          <Users className="mt-0.5 size-5 shrink-0 text-primary" aria-hidden="true" />
          <div>
            <h2 id="titulo-grupo" className="font-semibold text-foreground">
              Vai com a turma?
            </h2>
            <p className="text-sm text-muted-foreground">Segure os ingressos por até 24 h e mande o link: cada um paga a sua parte, no Pix ou no cartão.</p>
          </div>
        </div>
        {!aberto && (
          <Button variant="outline" size="sm" onClick={() => setAberto(true)}>
            Comprar em grupo
          </Button>
        )}
      </div>
      {aberto && (
        <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end">
          {opcoes.length > 1 && (
            <div className="flex flex-1 flex-col gap-1.5">
              <Label htmlFor="grupo-tipo">Ingresso</Label>
              <select id="grupo-tipo" className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground" value={tipo} onChange={(e) => setTipo(e.target.value)}>
                {opcoes.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.nome}
                  </option>
                ))}
              </select>
            </div>
          )}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="grupo-qtd">Quantas pessoas, com você</Label>
            <select id="grupo-qtd" className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground" value={qtd} onChange={(e) => setQtd(e.target.value)}>
              {Array.from({ length: 9 }, (_, i) => i + 2).map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </div>
          <Button onClick={criar} disabled={criando}>
            {criando ? 'Criando…' : 'Criar o grupo'}
          </Button>
        </div>
      )}
      {erro && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {erro}
        </p>
      )}
    </section>
  )
}
