// Perguntas personalizadas do checkout (item 7.6).
export type Pergunta = {
  id: number
  texto: string
  tipo: 'texto' | 'opcao'
  opcoes: string[]
  obrigatoria: boolean
  finalidade: string
  tipo_ingresso_ids: number[]
  ordem: number
}

// Respostas de cada ingresso: chave "tipoId:índice" → { perguntaId: resposta }.
export type RespostasCheckout = Record<string, Record<number, string>>

export const valeParaTipo = (p: Pergunta, tipoId: number) =>
  p.tipo_ingresso_ids.length === 0 || p.tipo_ingresso_ids.includes(tipoId)

// Primeira obrigatória sem resposta (para avisar antes de enviar).
export function faltando(perguntas: Pergunta[], quantidades: Record<number, number>, respostas: RespostasCheckout) {
  for (const [tipo, qtd] of Object.entries(quantidades)) {
    for (let i = 0; i < qtd; i++) {
      for (const p of perguntas) {
        if (p.obrigatoria && valeParaTipo(p, Number(tipo)) && !respostas[`${tipo}:${i}`]?.[p.id]?.trim()) return p
      }
    }
  }
  return null
}

// Formato da API: um objeto por ingresso, com as chaves em texto.
export function respostasDoTipo(tipoId: number, qtd: number, respostas: RespostasCheckout) {
  return Array.from({ length: qtd }, (_, i) => {
    const r = respostas[`${tipoId}:${i}`] ?? {}
    return Object.fromEntries(Object.entries(r).map(([k, v]) => [String(k), v]))
  })
}
