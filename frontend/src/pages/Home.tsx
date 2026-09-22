import { Button } from '@/components/ui/button'

export default function Home() {
  return (
    <main className="mx-auto flex max-w-5xl flex-col items-center justify-center gap-4 px-4 py-24 text-center">
      <h1 className="text-2xl font-semibold text-foreground">Arcadia</h1>
      <p className="text-muted-foreground">Plataforma de eventos e ingressos — em construção.</p>
      <Button>Explorar eventos</Button>
    </main>
  )
}
