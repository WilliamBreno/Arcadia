import * as React from 'react'
import { Input as InputPrimitive } from '@base-ui/react/input'
import { Popover } from '@base-ui/react/popover'
import { CalendarDays, ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from 'cn'

const iso = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const meses = ['Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho', 'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro']

// Mantém o input real: validação min/max/required, teclado e React Hook Form.
export function DateInput({ className, ref, ...props }: React.ComponentProps<'input'>) {
  const input = React.useRef<HTMLInputElement | null>(null)
  const [aberto, setAberto] = React.useState(false)
  const [mes, setMes] = React.useState(() => new Date())
  const [selecionado, setSelecionado] = React.useState('')
  const mudar = (valor: string) => {
    const el = input.current
    if (!el) return
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(el, valor)
    el.dispatchEvent(new Event('input', { bubbles: true }))
    el.dispatchEvent(new Event('change', { bubbles: true }))
    setSelecionado(valor.slice(0, 10))
  }
  const limite = (valor: string | number | undefined) => valor == null ? '' : String(valor).slice(0, 10)
  const min = limite(props.min), max = limite(props.max)
  const ano = mes.getFullYear(), m = mes.getMonth()
  const primeiro = new Date(ano, m, 1).getDay()
  const dias = new Date(ano, m + 1, 0).getDate()
  const indisponivel = (d: string) => !!((min && d < min) || (max && d > max))
  const escolher = (dia: string) => {
    const atual = input.current?.value ?? ''
    let valor = dia
    if (props.type === 'datetime-local') {
      valor += `T${atual.slice(11) || '09:00'}`
      if (props.min && valor < String(props.min)) valor = String(props.min)
      if (props.max && valor > String(props.max)) valor = String(props.max)
    }
    mudar(valor); setAberto(false); input.current?.focus()
  }
  const abrir = (open: boolean) => {
    if (open) {
      const atual = input.current?.value.slice(0, 10) || ''
      setSelecionado(atual)
      const base = atual || (max && max < iso(new Date()) ? max : min && min > iso(new Date()) ? min : iso(new Date()))
      setMes(new Date(`${base}T12:00:00`))
    }
    setAberto(open)
  }
  const inicioAno = Math.min(1900, Number(min.slice(0, 4)) || 1900, ano)
  const fimAno = Math.max(new Date().getFullYear() + 30, Number(max.slice(0, 4)) || 0, ano)
  return (
    <Popover.Root open={aberto} onOpenChange={abrir}>
      <div className={cn('flex h-10 min-w-0 items-center rounded-lg border border-input bg-background focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/30', className)}>
        <InputPrimitive {...props} ref={node => { const el = node as HTMLInputElement | null; input.current = el; if (typeof ref === 'function') ref(el); else if (ref) ref.current = el }} data-slot="input" lang="pt-BR" className="evve-date h-full w-full min-w-0 rounded-l-lg bg-transparent px-2 text-base outline-none disabled:opacity-50 md:text-sm" />
        <Popover.Trigger type="button" disabled={props.disabled || props.readOnly} aria-label={`Abrir calendário${props.id ? `: ${props.id}` : ''}`} className="flex h-full shrink-0 items-center rounded-r-lg border-l border-input px-2 text-muted-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring disabled:opacity-50">
          <CalendarDays className="size-4" />
        </Popover.Trigger>
      </div>
      <Popover.Portal><Popover.Positioner sideOffset={8} align="end" className="z-50"><Popover.Popup className="w-72 max-w-full rounded-xl border border-border bg-background p-3 text-foreground shadow-xl">
        <Popover.Title className="sr-only">Escolher data</Popover.Title>
        <div className="mb-3 flex items-center gap-1">
          <button type="button" aria-label="Mês anterior" onClick={() => setMes(new Date(ano, m - 1, 1))} className="rounded-md p-1 hover:bg-muted"><ChevronLeft className="size-4" /></button>
          <select aria-label="Mês" className="min-w-0 flex-1 rounded-md bg-muted p-1 text-sm" value={m} onChange={e => setMes(new Date(ano, Number(e.target.value), 1))}>{meses.map((nome, i) => <option key={nome} value={i}>{nome}</option>)}</select>
          <select aria-label="Ano" className="rounded-md bg-muted p-1 text-sm" value={ano} onChange={e => setMes(new Date(Number(e.target.value), m, 1))}>{Array.from({ length: fimAno - inicioAno + 1 }, (_, i) => inicioAno + i).map(a => <option key={a}>{a}</option>)}</select>
          <button type="button" aria-label="Próximo mês" onClick={() => setMes(new Date(ano, m + 1, 1))} className="rounded-md p-1 hover:bg-muted"><ChevronRight className="size-4" /></button>
        </div>
        <div className="grid grid-cols-7 gap-1 text-center text-sm">
          {['D', 'S', 'T', 'Q', 'Q', 'S', 'S'].map((d, i) => <span key={i} aria-hidden="true" className="py-1 text-xs text-muted-foreground">{d}</span>)}
          {Array.from({ length: primeiro }, (_, i) => <span key={`v${i}`} />)}
          {Array.from({ length: dias }, (_, i) => {
            const dia = iso(new Date(ano, m, i + 1))
            return <button key={dia} type="button" aria-label={new Date(`${dia}T12:00:00`).toLocaleDateString('pt-BR', { day: 'numeric', month: 'long', year: 'numeric' })} aria-pressed={dia === selecionado} aria-current={dia === iso(new Date()) ? 'date' : undefined} disabled={indisponivel(dia)} onClick={() => escolher(dia)} className={cn('h-8 rounded-md hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring disabled:opacity-25 disabled:hover:bg-transparent', dia === selecionado && 'bg-primary text-primary-foreground hover:bg-primary', dia === iso(new Date()) && 'border border-primary')}>{i + 1}</button>
          })}
        </div>
        <div className="mt-3 flex justify-between border-t border-border pt-2 text-sm">
          <button type="button" disabled={indisponivel(iso(new Date()))} onClick={() => escolher(iso(new Date()))} className="rounded-md px-2 py-1 hover:bg-muted disabled:opacity-30">Hoje</button>
          {!props.required && <button type="button" onClick={() => { mudar(''); setAberto(false) }} className="rounded-md px-2 py-1 hover:bg-muted">Limpar</button>}
          <Popover.Close className="rounded-md px-2 py-1 hover:bg-muted">Fechar</Popover.Close>
        </div>
        {props.type === 'datetime-local' && <p className="mt-2 text-xs text-muted-foreground">Ajuste o horário diretamente no campo.</p>}
      </Popover.Popup></Popover.Positioner></Popover.Portal>
    </Popover.Root>
  )
}
