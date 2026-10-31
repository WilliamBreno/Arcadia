// Descoberta (item 6.2): cidades, categorias, destaques e coleções.
import { api } from '@/lib/api'
import type { EventoPublicoItem, ListaEventosResposta } from '@/lib/publico'

export type Cidade = { cidade: string; uf: string; slug: string; eventos: number }
export type Categoria = { nome: string; slug: string; eventos: number }
export type Descoberta = { cidades: Cidade[]; categorias: Categoria[]; cidade: Cidade | null }
export type Destaques = { em_alta: EventoPublicoItem[]; quase_esgotando: EventoPublicoItem[]; novos: EventoPublicoItem[] }
export type Colecao = { slug: string; titulo: string; descricao: string; icone: string; eventos: EventoPublicoItem[] }
// Item 11.7: grade de coleções (só título e ícone).
export type ColecaoGrade = { slug: string; titulo: string; icone: string }

export const QUANDO = [
  { valor: '', rotulo: 'Qualquer data' },
  { valor: 'hoje', rotulo: 'Hoje' },
  { valor: 'amanha', rotulo: 'Amanhã' },
  { valor: 'fim-de-semana', rotulo: 'Fim de semana' },
] as const

export const obterDescoberta = (cidade?: string) =>
  api<Descoberta>(`/descoberta${cidade ? `?cidade=${encodeURIComponent(cidade)}` : ''}`)
export const obterDestaques = (cidade?: string) =>
  api<Destaques>(`/destaques${cidade ? `?cidade=${encodeURIComponent(cidade)}` : ''}`)
export const listarColecoes = () => api<Colecao[]>('/colecoes')
export const listarGradeColecoes = () => api<ColecaoGrade[]>('/colecoes?grade=1', { semAuth: true })
export const obterColecao = (slug: string) => api<Colecao>(`/colecoes/${encodeURIComponent(slug)}`)

export type FiltrosLista = {
  cidade?: string
  uf?: string
  categoria?: string
  quando?: string
  q?: string
  gratuito?: boolean
  lat?: number
  lng?: number
  raioKm?: number
  porPagina?: number
}

export function listarEventos(f: FiltrosLista) {
  const p = new URLSearchParams()
  if (f.cidade) p.set('cidade', f.cidade)
  if (f.uf) p.set('uf', f.uf)
  if (f.categoria) p.set('categoria', f.categoria)
  if (f.quando) p.set('atalho', f.quando)
  if (f.q) p.set('q', f.q)
  if (f.gratuito) p.set('gratuito', 'true')
  if (f.lat != null && f.lng != null) {
    p.set('lat', String(f.lat))
    p.set('lng', String(f.lng))
    p.set('raio_km', String(f.raioKm ?? 50))
    p.set('ordenar', 'distancia')
  }
  p.set('por_pagina', String(f.porPagina ?? 48))
  return api<ListaEventosResposta>(`/eventos?${p.toString()}`)
}

// Distância em km (Haversine), para mostrar "a 3 km" no mapa.
export function distanciaKm(a: { lat: number; lng: number }, b: { lat: number; lng: number }): number {
  const rad = (g: number) => (g * Math.PI) / 180
  const dLat = rad(b.lat - a.lat)
  const dLng = rad(b.lng - a.lng)
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(rad(a.lat)) * Math.cos(rad(b.lat)) * Math.sin(dLng / 2) ** 2
  return 6371 * 2 * Math.asin(Math.min(1, Math.sqrt(h)))
}
