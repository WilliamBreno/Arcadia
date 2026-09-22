const NOME_PLATAFORMA = import.meta.env.VITE_NOME_PLATAFORMA ?? 'Arcadia'

export function SiteFooter() {
  return (
    <footer className="border-border border-t">
      <div className="mx-auto flex max-w-5xl items-center justify-center px-4 py-6 text-sm text-muted-foreground">
        © {new Date().getFullYear()} {NOME_PLATAFORMA}
      </div>
    </footer>
  )
}
