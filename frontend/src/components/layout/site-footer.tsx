import { Link } from 'react-router-dom'

const NOME_PLATAFORMA = import.meta.env.VITE_NOME_PLATAFORMA ?? 'Evve'

export function SiteFooter() {
  return (
    <footer className="border-border border-t">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-center gap-x-6 gap-y-2 px-4 py-6 text-sm text-muted-foreground">
        <Link to="/quanto-custa" className="hover:text-foreground">
          Quanto custa
        </Link>
        <Link to="/mapa" className="hover:text-foreground">
          Eventos perto de você
        </Link>
        <Link to="/cookies" className="hover:text-foreground">
          Cookies
        </Link>
        <span>
          © {new Date().getFullYear()} {NOME_PLATAFORMA}
        </span>
      </div>
    </footer>
  )
}
