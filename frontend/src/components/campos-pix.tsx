import { Link } from 'react-router-dom'
import type { UseFormRegister, FieldValues, Path } from 'react-hook-form'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useOrganizador } from '@/hooks/use-organizador'

// Tipos de chave Pix aceitos (o servidor confere o formato de cada um).
export const TIPOS_PIX = [
  { valor: 'cpf', rotulo: 'CPF', exemplo: '000.000.000-00' },
  { valor: 'cnpj', rotulo: 'CNPJ', exemplo: '00.000.000/0000-00' },
  { valor: 'email', rotulo: 'E-mail', exemplo: 'voce@exemplo.com' },
  { valor: 'telefone', rotulo: 'Celular', exemplo: '(82) 99999-9999' },
  { valor: 'aleatoria', rotulo: 'Chave aleatória', exemplo: 'copie do app do seu banco' },
] as const

type Props<T extends FieldValues> = {
  register: UseFormRegister<T>
  tipoSelecionado: string
}

// Campos da chave Pix (dados de recebimento dos repasses).
export function CamposPix<T extends FieldValues>({ register, tipoSelecionado }: Props<T>) {
  const tipo = TIPOS_PIX.find((t) => t.valor === tipoSelecionado)
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="tipo_chave_pix">Tipo de chave</Label>
        <select
          id="tipo_chave_pix"
          className="h-8 rounded-lg border border-border bg-background px-2.5 text-sm"
          {...register('tipo_chave_pix' as Path<T>)}
        >
          <option value="">Escolha…</option>
          {TIPOS_PIX.map((t) => (
            <option key={t.valor} value={t.valor}>
              {t.rotulo}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-col gap-1.5 sm:col-span-2">
        <Label htmlFor="chave_pix">Chave Pix</Label>
        <Input
          id="chave_pix"
          autoComplete="off"
          inputMode={tipoSelecionado === 'cpf' || tipoSelecionado === 'cnpj' || tipoSelecionado === 'telefone' ? 'numeric' : undefined}
          placeholder={tipo?.exemplo ?? 'Escolha o tipo primeiro'}
          disabled={!tipo}
          {...register('chave_pix' as Path<T>)}
        />
      </div>
      <p className="text-xs text-muted-foreground sm:col-span-3">
        É para esta chave que enviamos o dinheiro dos seus ingressos depois do evento. Use uma chave no mesmo CPF/CNPJ do seu
        perfil: a verificação confere a titularidade.
      </p>
    </div>
  )
}

// Aviso no painel quando o organizador ainda não cadastrou o Pix.
export function AvisoPix() {
  const { organizador } = useOrganizador()
  if (!organizador || organizador.chave_pix) return null
  return (
    <div className="mb-6 flex flex-col gap-3 rounded-xl border border-foreground/20 bg-muted/40 p-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="font-medium text-foreground">Cadastre sua chave Pix para receber</p>
        <p className="text-sm text-muted-foreground">
          Sem ela você não consegue publicar eventos pagos, e os repasses não têm para onde ir.
        </p>
      </div>
      <Link
        to="/organizador/perfil#pix"
        className="inline-flex h-8 shrink-0 items-center justify-center rounded-lg bg-primary px-3 text-sm font-medium text-primary-foreground"
      >
        Cadastrar Pix
      </Link>
    </div>
  )
}
