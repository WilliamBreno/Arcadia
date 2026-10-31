import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { IconeColecao } from '@/components/icone-colecao'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { api, ApiError } from '@/lib/api'

type Tema = {
  id: number
  slug: string
  titulo: string
  icone: string
  exibir_de: string | null
  exibir_ate: string | null
  marcado: boolean
  pela_equipe: boolean
  bloqueado: boolean
  aberto: boolean
}

const MAX = 3
const dataBR = (d: string | null) => (d ? d.split('-').reverse().slice(0, 2).join('/') : '')

// Item 11.7: o organizador coloca o evento nos temas da época (Halloween,
// Réveillon...), que aparecem em /colecoes e na página inicial.
export function TemasCard({ eventoId }: { eventoId: number }) {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['org-temas', eventoId], queryFn: () => api<Tema[]>(`/org/eventos/${eventoId}/temas`) })
  const [escolha, setEscolha] = useState<number[] | null>(null)
  const [erro, setErro] = useState<string | null>(null)
  const [salvo, setSalvo] = useState(false)
  const [salvando, setSalvando] = useState(false)

  if (!data || data.length === 0) return null
  // Os incluídos pela equipe da Evve ficam marcados e não contam no limite.
  const marcados = escolha ?? data.filter((t) => t.marcado && !t.pela_equipe).map((t) => t.id)
  const alternar = (id: number) => {
    setSalvo(false)
    setErro(null)
    if (marcados.includes(id)) setEscolha(marcados.filter((x) => x !== id))
    else if (marcados.length < MAX) setEscolha([...marcados, id])
    else setErro(`Escolha até ${MAX} temas.`)
  }

  const salvar = async () => {
    setErro(null)
    setSalvando(true)
    try {
      const r = await api<Tema[]>(`/org/eventos/${eventoId}/temas`, { method: 'PUT', body: { colecao_ids: marcados } })
      queryClient.setQueryData(['org-temas', eventoId], r)
      setEscolha(null)
      setSalvo(true)
    } catch (e) {
      setErro(e instanceof ApiError ? e.message : 'Não foi possível salvar.')
    } finally {
      setSalvando(false)
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Temas da vitrine</CardTitle>
        <p className="text-sm text-muted-foreground">
          Coloque o evento em até {MAX} temas. Ele aparece na coleção do tema durante a época, na página de coleções e na página inicial.
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {data.map((t) => {
            const ativo = t.pela_equipe || marcados.includes(t.id)
            const desabilitado = t.bloqueado || t.pela_equipe || (!t.aberto && !ativo)
            return (
              <li key={t.id}>
                <button
                  type="button"
                  aria-pressed={ativo}
                  disabled={desabilitado}
                  onClick={() => alternar(t.id)}
                  className={`flex w-full items-center gap-3 rounded-xl border px-3 py-2 text-left transition-colors disabled:cursor-not-allowed ${
                    t.pela_equipe ? '' : 'disabled:opacity-50'
                  } ${
                    ativo ? 'border-foreground bg-muted' : 'border-border hover:bg-muted'
                  }`}
                >
                  <IconeColecao nome={t.icone} className="size-8 shrink-0" />
                  <span className="min-w-0">
                    <span className="block text-sm font-medium text-foreground">{t.titulo}</span>
                    <span className="block text-xs text-muted-foreground">
                      {t.bloqueado
                        ? 'Tirado pela equipe da Evve'
                        : t.pela_equipe
                          ? 'Incluído pela equipe da Evve'
                        : t.exibir_de || t.exibir_ate
                          ? `No ar de ${dataBR(t.exibir_de) || '…'} a ${dataBR(t.exibir_ate) || '…'}`
                          : 'O ano todo'}
                    </span>
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
        {erro && (
          <p role="alert" className="text-sm text-destructive">
            {erro}
          </p>
        )}
        <div className="flex items-center gap-3">
          <Button size="sm" onClick={salvar} disabled={salvando || escolha === null}>
            {salvando ? 'Salvando…' : 'Salvar temas'}
          </Button>
          {salvo && (
            <span role="status" className="text-sm text-muted-foreground">
              Salvo.
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
