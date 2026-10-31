import { api } from '@/lib/api'

export type Municipio = { id: number; nome: string; uf: string }
export type OfertaArtista = {
  habilitada: boolean; preco_tipo: 'fixo' | 'a_partir' | 'consulta'; valor_centavos: number | null
  passagem_inclusa: boolean; hospedagem_inclusa: boolean; alimentacao_inclusa: boolean
  integrantes: number; duracao_minutos: number; origem_id: number | null; origem?: Municipio; observacoes: string
}
export type OrcamentoArtista = {
  id: number; grupo_id: number; grupo_nome: string; grupo_slug: string; cliente_nome: string; sou_cliente: boolean
  inicio: string; fim: string; destino_id: number; destino?: Municipio; tipo_evento: string; duracao_minutos: number
  observacoes: string; status: string; proposta_centavos: number | null; proposta_texto: string; versao: number
  motivo: string; criado_em: string; atualizado_em: string
}
export type MensagemArtista = { id: number; texto: string; autor: string; minha: boolean; criado_em: string }
export type ConversaArtista = { pedido: OrcamentoArtista; mensagens: MensagemArtista[] }
export type ViagemConfig = {
  habilitada: boolean; onibus_centavos_km: number | null; aviao_centavos_km: number | null
  distancia_min_km: number | null; aviao_desde_km: number | null; ida_volta: boolean
}
export type EstimativaViagem = {
  disponivel: boolean; passagem_inclusa: boolean; distancia_km: number; transporte: string
  viagem_centavos: number | null; total_centavos: number | null; preco_tipo: string; ida_volta: boolean; aviso: string
}
export const statusOrcamento: Record<string, string> = {
  novo: 'Novo', em_conversa: 'Em conversa', proposta_enviada: 'Proposta enviada', aceito: 'Confirmado',
  recusado: 'Recusado', realizado: 'Realizado', cancelado: 'Cancelado',
}
export const diaOrcamento = (s: string) => s.split('-').reverse().join('/')
// Converte a entrada decimal em centavos inteiros sem fazer cálculo de preço.
export function lerCentavos(s: string): number | null {
  const m = /^(\d{1,11})(?:[,.](\d{1,2}))?$/.exec(s.trim())
  if (!m) return null
  const valor = Number(m[1]) * 100 + Number((m[2] ?? '').padEnd(2, '0'))
  return Number.isSafeInteger(valor) && valor <= 1_000_000_000_000 ? valor : null
}
export const textoCentavos = (n: number | null) => n === null ? '' : `${Math.floor(n / 100)},${String(n % 100).padStart(2, '0')}`
export const obterOferta = (id: number) => api<OfertaArtista>(`/me/artistas/${id}/oferta`)
export const listarOrcamentos = (id?: number, pagina = 1) => api<OrcamentoArtista[]>((id ? `/me/artistas/${id}/orcamentos` : '/me/artistas-orcamentos') + `?pagina=${pagina}`)
