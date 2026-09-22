import { Route, Routes } from 'react-router-dom'

import { ProtectedRoute } from '@/components/protected-route'
import { SiteLayout } from '@/components/layout/site-layout'
import Cadastro from '@/pages/Cadastro'
import Home from '@/pages/Home'
import Login from '@/pages/Login'
import EventoEditar from '@/pages/organizador/EventoEditar'
import EventoNovo from '@/pages/organizador/EventoNovo'
import Eventos from '@/pages/organizador/Eventos'
import Locais from '@/pages/organizador/Locais'
import LocalForm from '@/pages/organizador/LocalForm'
import Perfil from '@/pages/organizador/Perfil'

function App() {
  return (
    <Routes>
      <Route element={<SiteLayout />}>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Login />} />
        <Route path="/cadastro" element={<Cadastro />} />

        <Route element={<ProtectedRoute />}>
          <Route path="/organizador/perfil" element={<Perfil />} />
          <Route path="/organizador/locais" element={<Locais />} />
          <Route path="/organizador/locais/novo" element={<LocalForm />} />
          <Route path="/organizador/locais/:id" element={<LocalForm />} />
          <Route path="/organizador/eventos" element={<Eventos />} />
          <Route path="/organizador/eventos/novo" element={<EventoNovo />} />
          <Route path="/organizador/eventos/:id" element={<EventoEditar />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
