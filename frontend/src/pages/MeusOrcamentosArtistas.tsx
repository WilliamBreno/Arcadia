import { ArtistaOrcamentos } from '@/components/artista-orcamentos'
import { VoltarLink } from '@/components/voltar-link'
export default function MeusOrcamentosArtistas() {
  return <main className="mx-auto flex max-w-3xl flex-col gap-5 px-4 py-10"><VoltarLink to="/artistas">Artistas</VoltarLink><h1 className="text-2xl font-bold">Contratações</h1><ArtistaOrcamentos /></main>
}
