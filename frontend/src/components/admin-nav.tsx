import { NavLink } from 'react-router-dom'

const links = [
  { to: '/admin/repasses', rotulo: 'Repasses' },
  { to: '/admin/organizadores', rotulo: 'Organizadores' },
  { to: '/admin/colecoes', rotulo: 'Coleções' },
  { to: '/admin/avaliacoes', rotulo: 'Avaliações' },
  { to: '/admin/relatorios', rotulo: 'Relatórios' },
]

// Navegação entre as telas do admin da plataforma.
export function AdminNav() {
  return (
    <nav className="mb-6 flex gap-4 border-b border-border text-sm" aria-label="Admin">
      {links.map((l) => (
        <NavLink
          key={l.to}
          to={l.to}
          className={({ isActive }) =>
            `-mb-px border-b-2 pb-2 ${isActive ? 'border-primary font-medium text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}`
          }
        >
          {l.rotulo}
        </NavLink>
      ))}
    </nav>
  )
}
