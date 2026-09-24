import { Route, Routes } from 'react-router-dom'

import { ProtectedRoute } from '@/components/protected-route'
import { SiteLayout } from '@/components/layout/site-layout'
import AreaJurado from '@/pages/AreaJurado'
import Cadastro from '@/pages/Cadastro'
import Checkin from '@/pages/Checkin'
import Convite from '@/pages/Convite'
import EventoDetalhe from '@/pages/EventoDetalhe'
import Home from '@/pages/Home'
import Inscricao from '@/pages/Inscricao'
import Login from '@/pages/Login'
import MeusEventos from '@/pages/MeusEventos'
import MeusIngressos from '@/pages/MeusIngressos'
import OrganizadorPublico from '@/pages/OrganizadorPublico'
import Pedido from '@/pages/Pedido'
import EventoEditar from '@/pages/organizador/EventoEditar'
import EventoNovo from '@/pages/organizador/EventoNovo'
import AdminRelatorios from '@/pages/admin/AdminRelatorios'
import AdminRepasses from '@/pages/admin/AdminRepasses'
import Eventos from '@/pages/organizador/Eventos'
import Repasses from '@/pages/organizador/Repasses'
import Locais from '@/pages/organizador/Locais'
import LocalForm from '@/pages/organizador/LocalForm'
import Perfil from '@/pages/organizador/Perfil'

function App() {
  return (
    <Routes>
      <Route element={<SiteLayout />}>
        <Route path="/" element={<Home />} />
        <Route path="/e/:slug" element={<EventoDetalhe />} />
        <Route path="/o/:slug" element={<OrganizadorPublico />} />
        <Route path="/convite/:token" element={<Convite />} />
        <Route path="/login" element={<Login />} />
        <Route path="/cadastro" element={<Cadastro />} />

        <Route element={<ProtectedRoute />}>
          <Route path="/e/:slug/inscricao" element={<Inscricao />} />
          <Route path="/e/:slug/jurado" element={<AreaJurado />} />
          <Route path="/pedidos/:id" element={<Pedido />} />
          <Route path="/meus-ingressos" element={<MeusIngressos />} />
          <Route path="/meus-eventos" element={<MeusEventos />} />
          <Route path="/checkin/:eventoId" element={<Checkin />} />
          <Route path="/organizador/perfil" element={<Perfil />} />
          <Route path="/organizador/locais" element={<Locais />} />
          <Route path="/organizador/locais/novo" element={<LocalForm />} />
          <Route path="/organizador/locais/:id" element={<LocalForm />} />
          <Route path="/organizador/eventos" element={<Eventos />} />
          <Route path="/organizador/repasses" element={<Repasses />} />
          <Route path="/admin/repasses" element={<AdminRepasses />} />
          <Route path="/admin/relatorios" element={<AdminRelatorios />} />
          <Route path="/organizador/eventos/novo" element={<EventoNovo />} />
          <Route path="/organizador/eventos/:id" element={<EventoEditar />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
