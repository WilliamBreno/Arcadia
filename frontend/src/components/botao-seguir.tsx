import { Bell, BellRing } from 'lucide-react'
import { useLocation, useNavigate } from 'react-router-dom'

import { useSeguindo, type TipoSeguir } from '@/lib/seguir'

type Props = { tipo: TipoSeguir; alvo: string; rotulo: string; texto?: string }

// Botão Seguir (item 6.4): quem segue recebe por e-mail cada evento novo
// do organizador ou da categoria. Sem login, leva ao login e volta.
export function BotaoSeguir({ tipo, alvo, rotulo, texto }: Props) {
  const { segue, seguir, deixar, logado } = useSeguindo()
  const navigate = useNavigate()
  const location = useLocation()
  const ativo = segue(tipo, alvo)
  const ocupado = seguir.isPending || deixar.isPending

  const alternar = () => {
    if (!logado) {
      navigate('/login', { state: { de: location.pathname + location.search } })
      return
    }
    if (ativo) deixar.mutate({ tipo, alvo })
    else seguir.mutate({ tipo, alvo, rotulo })
  }

  return (
    <button
      type="button"
      onClick={alternar}
      disabled={ocupado}
      aria-pressed={ativo}
      title={ativo ? 'Você recebe por e-mail os eventos novos. Clique para deixar de seguir.' : 'Receba por e-mail os eventos novos'}
      className={`inline-flex h-8 items-center gap-1.5 rounded-full px-3 text-sm font-medium transition-colors active:scale-95 disabled:opacity-60 ${
        ativo ? 'bg-muted text-foreground hover:bg-foreground/10' : 'bg-primary text-primary-foreground hover:opacity-90'
      }`}
    >
      {ativo ? <BellRing className="size-4" aria-hidden="true" /> : <Bell className="size-4" aria-hidden="true" />}
      {ativo ? 'Seguindo' : (texto ?? 'Seguir')}
    </button>
  )
}
