import { calcularItem, type RegraTaxas } from '@/lib/taxas'

// Item 10.3: adicionais, pacotes e compra em grupo.
export type Adicional = {
  id: number
  nome: string
  descricao: string
  preco_centavos: number
  quantidade: number
  vendidos: number
  disponivel: number
  max_por_pedido: number
  ativo: boolean
}

export type PacoteItem = { tipo_ingresso_id: number; nome: string; quantidade: number; adicional: boolean }
export type Pacote = {
  id: number
  nome: string
  descricao: string
  preco_centavos: number
  tabela_centavos: number
  quantidade: number
  vendidos: number
  max_por_pedido: number
  ativo: boolean
  itens: PacoteItem[]
}

export type Extras = { adicionais: Adicional[]; pacotes: Pacote[] }

export type Grupo = {
  token: string
  evento_titulo: string
  evento_slug: string
  evento_inicio_em: string | null
  tipo_ingresso_id: number
  tipo_nome: string
  preco_centavos: number
  total_por_pessoa_centavos: number
  quantidade: number
  pagas: number
  em_pagamento: number
  restantes: number
  expira_em: string
  status: 'aberto' | 'cancelado' | 'encerrado'
  criador: string
  sou_criador: boolean
  minha_parte?: '' | 'reservado' | 'pago'
  meu_pedido_id?: number
}

export type MeuAdicional = {
  id: number
  nome: string
  evento_titulo: string
  evento_slug: string
  evento_inicio_em: string | null
  codigo: string
  qr: string
  status: 'pago' | 'utilizado'
  entregue_em: string | null
}

// Espelho de RatearPacote (backend): preço por unidade de cada item do
// pacote, proporcional à tabela; o centavo que sobra vai numa unidade.
export function precosDoPacote(p: Pacote, tabela: Record<number, number>): number[] {
  const soma = p.itens.reduce((a, it) => a + (tabela[it.tipo_ingresso_id] ?? 0) * it.quantidade, 0)
  const unidades = p.itens.reduce((a, it) => a + it.quantidade, 0)
  const precos: number[] = []
  for (const it of p.itens) {
    const unit = soma > 0 ? Math.floor((p.preco_centavos * (tabela[it.tipo_ingresso_id] ?? 0)) / soma) : Math.floor(p.preco_centavos / unidades)
    for (let i = 0; i < it.quantidade; i++) precos.push(unit)
  }
  const sobra = p.preco_centavos - precos.reduce((a, b) => a + b, 0)
  if (precos.length > 0) precos[0] += sobra
  return precos
}

export type ResumoValores = { preco: number; taxa: number; garantia: number; garantiaSeMarcar: number; total: number; temPago: boolean }

// Valores dos adicionais e pacotes escolhidos (prévia; o servidor recalcula).
export function valoresExtras(
  regra: RegraTaxas,
  garantia: boolean,
  desconto: (preco: number) => number,
  extras: Extras | undefined,
  qtdAdicionais: Record<number, number>,
  qtdPacotes: Record<number, number>,
  tabela: Record<number, number>,
): ResumoValores & { unidades: number } {
  const r = { preco: 0, taxa: 0, garantia: 0, garantiaSeMarcar: 0, total: 0, temPago: false, unidades: 0 }
  const somar = (preco: number, qtd: number) => {
    const v = calcularItem(regra, preco, garantia)
    r.preco += qtd * preco
    r.taxa += qtd * v.taxa
    r.garantia += qtd * v.garantia
    r.garantiaSeMarcar += qtd * calcularItem(regra, preco, true).garantia
    r.total += qtd * v.total
    r.temPago = r.temPago || preco > 0
  }
  for (const a of extras?.adicionais ?? []) {
    const q = qtdAdicionais[a.id] ?? 0
    if (q > 0) {
      somar(a.preco_centavos - desconto(a.preco_centavos), q)
      r.unidades += q
    }
  }
  for (const p of extras?.pacotes ?? []) {
    const q = qtdPacotes[p.id] ?? 0
    if (q <= 0) continue
    for (const preco of precosDoPacote(p, tabela)) somar(preco, q)
    r.unidades += q
  }
  return r
}

export function somarResumo(a: ResumoValores, b: ResumoValores): ResumoValores {
  return {
    preco: a.preco + b.preco,
    taxa: a.taxa + b.taxa,
    garantia: a.garantia + b.garantia,
    garantiaSeMarcar: a.garantiaSeMarcar + b.garantiaSeMarcar,
    total: a.total + b.total,
    temPago: a.temPago || b.temPago,
  }
}
