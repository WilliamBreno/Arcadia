import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import type { Local } from '@/lib/organizador'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

// Pode ser usado dentro do formulário do evento, sem formulário aninhado.
export function NovoLocalEvento({ aoCriar }: { aoCriar: (id: string) => void }) {
  const [aberto, setAberto] = useState(false)
  const [dados, setDados] = useState({ nome: '', cidade: '', uf: '', logradouro: '', numero: '', bairro: '' })
  const [erro, setErro] = useState(''), [salvando, setSalvando] = useState(false)
  const qc = useQueryClient()
  const criar = async () => {
    setErro('')
    if (dados.nome.trim().length < 2 || !dados.cidade.trim() || dados.uf.length !== 2) { setErro('Informe o nome do local, a cidade e a UF.'); return }
    setSalvando(true)
    try {
      const local = await api<Local>('/org/locais', { method: 'POST', body: { ...dados, nome: dados.nome.trim(), cidade: dados.cidade.trim(), uf: dados.uf.toUpperCase() } })
      qc.setQueryData<Local[]>(['org-locais'], antigos => [...(antigos ?? []), local])
      aoCriar(String(local.id)); setAberto(false)
    } catch (e) { setErro(e instanceof ApiError ? e.message : 'Não foi possível cadastrar o local.') }
    finally { setSalvando(false) }
  }
  return <div className="flex flex-col gap-2">
    <Button type="button" variant="outline" className="self-start" onClick={() => setAberto(!aberto)}>{aberto ? 'Fechar cadastro de local' : 'Cadastrar local presencial'}</Button>
    {aberto && <fieldset className="flex flex-col gap-3 rounded-lg border border-border p-3"><legend className="px-1 text-sm font-medium">Novo local presencial</legend>
      {(['nome', 'cidade', 'uf', 'logradouro', 'numero', 'bairro'] as const).map(campo => <div key={campo} className="flex flex-col gap-1"><Label htmlFor={`local-${campo}`}>{{ nome: 'Nome do local', cidade: 'Cidade', uf: 'UF', logradouro: 'Rua / avenida', numero: 'Número', bairro: 'Bairro' }[campo]}</Label><Input id={`local-${campo}`} maxLength={campo === 'uf' ? 2 : 180} value={dados[campo]} onChange={e => setDados({ ...dados, [campo]: e.target.value })} /></div>)}
      <p className="text-xs text-muted-foreground">O local será cadastrado e selecionado neste evento. Salve o evento para aplicar a escolha.</p>
      {erro && <p role="alert" className="text-sm text-destructive">{erro}</p>}
      <Button type="button" disabled={salvando} onClick={criar}>{salvando ? 'Cadastrando…' : 'Cadastrar e selecionar local'}</Button>
    </fieldset>}
  </div>
}
