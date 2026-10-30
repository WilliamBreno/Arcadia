// Item 9.5: mapa de assentos e mesas.
export type LugarMapa = { id: number; numero: number; ocupado: boolean }
export type GrupoMapa = { grupo: string; mesa_inteira: boolean; lugares: LugarMapa[] }
export type SetorMapa = {
  id: number
  nome: string
  formato: 'fileiras' | 'mesas'
  tipo_ingresso_ids: number[]
  grupos: GrupoMapa[]
  total: number
  livres: number
}

export function rotuloLugar(setor: SetorMapa, grupo: string, numero: number) {
  return setor.formato === 'mesas' ? `${setor.nome} · Mesa ${grupo} · Cadeira ${numero}` : `${setor.nome} · Fileira ${grupo} · Lugar ${numero}`
}

// Cadeiras escolhidas: id do lugar → tipo de ingresso daquela cadeira.
export type Escolhas = Record<number, number>
