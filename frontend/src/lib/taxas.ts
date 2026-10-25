// Espelho de backend/internal/service/taxas.go (item 5.1), só para
// exibir a taxa e a garantia antes da compra. O valor cobrado é sempre
// recalculado no servidor.

export type RegraTaxas = {
  faixas: { ate_centavos: number; taxa_centavos: number }[] // ate_centavos < 0 = sem limite
  garantia_percentual_bp: number
  garantia_min_centavos: number
  garantia_max_centavos: number
  limite_bp: number | null
  taxa_paga_por: 'comprador' | 'organizador'
}

export type ValoresItem = {
  taxa: number // taxa que o comprador paga (0 se o organizador absorve)
  garantia: number
  total: number
}

export function taxaPorFaixa(regra: RegraTaxas, preco: number): number {
  if (preco <= 0) return 0
  const faixa = regra.faixas.find((f) => f.ate_centavos < 0 || preco <= f.ate_centavos)
  return faixa?.taxa_centavos ?? 0
}

export function garantiaDoPreco(regra: RegraTaxas, preco: number): number {
  if (preco <= 0) return 0
  let g = Math.floor((preco * regra.garantia_percentual_bp + 5000) / 10000)
  if (g < regra.garantia_min_centavos) g = regra.garantia_min_centavos
  if (regra.garantia_max_centavos > 0 && g > regra.garantia_max_centavos) g = regra.garantia_max_centavos
  return g
}

export function calcularItem(regra: RegraTaxas, preco: number, garantia: boolean): ValoresItem {
  const absorvida = regra.taxa_paga_por === 'organizador'
  let taxa = taxaPorFaixa(regra, preco)
  if (!absorvida && regra.limite_bp !== null) {
    taxa = Math.min(taxa, Math.floor((preco * regra.limite_bp) / 10000))
  }
  if (absorvida) taxa = 0
  const g = garantia ? garantiaDoPreco(regra, preco) : 0
  return { taxa, garantia: g, total: preco + taxa + g }
}
