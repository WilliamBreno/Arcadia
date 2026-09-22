import { Button } from '@/components/ui/button'

export default function Home() {
  return (
    <main className="flex min-h-svh flex-col items-center justify-center gap-4">
      <h1 className="text-2xl font-semibold">Arcadia</h1>
      <p className="text-muted-foreground">Plataforma de eventos e ingressos — em construção.</p>
      <Button>Explorar eventos</Button>
    </main>
  )
}
