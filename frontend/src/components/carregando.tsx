// Tela de carregamento: o ícone da marca pulsando. `role="status"` avisa
// leitores de tela; com "reduzir movimento" a animação é desligada.
export function Carregando({ texto = 'Carregando…' }: { texto?: string }) {
  return (
    <div role="status" aria-live="polite" className="flex flex-col items-center gap-3 p-12">
      <img
        src="/brand/icon-192.png"
        alt=""
        aria-hidden="true"
        className="size-14 animate-pulse motion-reduce:animate-none"
      />
      <span className="text-sm text-muted-foreground">{texto}</span>
    </div>
  )
}
