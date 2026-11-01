import { Bell, BellRing } from 'lucide-react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'

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
    <Button
      variant={ativo ? 'secondary' : 'default'}
      size="sm"
      type="button"
      onClick={alternar}
      disabled={ocupado}
      aria-pressed={ativo}
      title={ativo ? 'Você recebe por e-mail os eventos novos. Clique para deixar de seguir.' : 'Receba por e-mail os eventos novos'}
      className="rounded-full"
    >
      {ativo ? <BellRing className="size-4" aria-hidden="true" /> : <Bell className="size-4" aria-hidden="true" />}
      {ativo ? 'Seguindo' : (texto ?? 'Seguir')}
    </Button>
  )
}
