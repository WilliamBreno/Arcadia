import { NavLink } from 'react-router-dom'

const links = [
  { to: '/admin/repasses', rotulo: 'Repasses' },
  { to: '/admin/organizadores', rotulo: 'Organizadores' },
  { to: '/admin/colecoes', rotulo: 'Coleções' },
  { to: '/admin/artistas', rotulo: 'Artistas' },
  { to: '/admin/avaliacoes', rotulo: 'Avaliações' },
  { to: '/admin/suporte', rotulo: 'Suporte e status' },
  { to: '/admin/relatorios', rotulo: 'Relatórios' },
]

// Navegação entre as telas do admin da plataforma.
export function AdminNav() {
  return (
    <nav className="mb-6 flex flex-wrap gap-1 rounded-2xl border border-border bg-card p-2 text-sm shadow-xs" aria-label="Admin">
      {links.map((l) => (
        <NavLink
          key={l.to}
          to={l.to}
          className={({ isActive }) =>
            `evve-nav-link ${isActive ? 'bg-accent font-semibold text-accent-foreground' : ''}`
          }
        >
          {l.rotulo}
        </NavLink>
      ))}
    </nav>
  )
}
