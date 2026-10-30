import { Route, Routes } from 'react-router-dom'

import { ProtectedRoute } from '@/components/protected-route'
import { SiteLayout } from '@/components/layout/site-layout'
import AreaJurado from '@/pages/AreaJurado'
import Cadastro from '@/pages/Cadastro'
import Checkin from '@/pages/Checkin'
import Convite from '@/pages/Convite'
import EventoDetalhe from '@/pages/EventoDetalhe'
import Home from '@/pages/Home'
import IngressoPublico from '@/pages/IngressoPublico'
import Certificado from '@/pages/Certificado'
import Pesquisa from '@/pages/Pesquisa'
import Ajuda from '@/pages/Ajuda'
import VerificarIngresso from '@/pages/VerificarIngresso'
import Status from '@/pages/Status'
import Grupo from '@/pages/Grupo'
import AdminSuporte from '@/pages/admin/AdminSuporte'
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
import AdminOrganizadores from '@/pages/admin/AdminOrganizadores'
import AdminColecoes from '@/pages/admin/AdminColecoes'
import AdminAvaliacoes from '@/pages/admin/AdminAvaliacoes'
import Cookies from '@/pages/Cookies'
import CircuitoPublico from '@/pages/CircuitoPublico'
import CandidaturaExpositor from '@/pages/CandidaturaExpositor'
import Caravana from '@/pages/Caravana'
import VendaPorta from '@/pages/VendaPorta'
import AoVivo from '@/pages/AoVivo'
import CredenciaisImprimir from '@/pages/organizador/CredenciaisImprimir'
import PainelDivulgador from '@/pages/PainelDivulgador'
import MeusCircuitos from '@/pages/organizador/Circuitos'
import Crm from '@/pages/organizador/Crm'
import PerfilTalento from '@/pages/PerfilTalento'
import MeuTalento from '@/pages/MeuTalento'
import AutorizarPerfil from '@/pages/AutorizarPerfil'
import PalcoCamarim from '@/pages/PalcoCamarim'
import Telao from '@/pages/Telao'
import WidgetEvento from '@/pages/WidgetEvento'
import CidadeEventos from '@/pages/CidadeEventos'
import ColecaoPublica from '@/pages/ColecaoPublica'
import MapaEventos from '@/pages/MapaEventos'
import QuantoCusta from '@/pages/QuantoCusta'
import Descadastrar from '@/pages/Descadastrar'
import ApiKeys from '@/pages/organizador/ApiKeys'
import Equipe from '@/pages/organizador/Equipe'
import Coproducoes from '@/pages/organizador/Coproducoes'
import Eventos from '@/pages/organizador/Eventos'
import Relatorios from '@/pages/organizador/Relatorios'
import Repasses from '@/pages/organizador/Repasses'
import Locais from '@/pages/organizador/Locais'
import LocalForm from '@/pages/organizador/LocalForm'
import Perfil from '@/pages/organizador/Perfil'

function App() {
  return (
    <Routes>
      {/* Item 7.5: widget para colar no site do organizador (sem cabeçalho). */}
      <Route path="/widget/e/:slug" element={<WidgetEvento />} />
      {/* Item 8.3: telão em tela cheia, aberto por link secreto. */}
      <Route path="/telao/:token" element={<Telao />} />
      <Route element={<SiteLayout />}>
        <Route path="/" element={<Home />} />
        <Route path="/e/:slug" element={<EventoDetalhe />} />
        <Route path="/eventos/:cidade" element={<CidadeEventos />} />
        <Route path="/eventos/:cidade/:categoria" element={<CidadeEventos />} />
        <Route path="/c/:slug" element={<ColecaoPublica />} />
        <Route path="/circuitos/:slug" element={<CircuitoPublico />} />
        <Route path="/divulgador/:token" element={<PainelDivulgador />} />
        <Route path="/mapa" element={<MapaEventos />} />
        <Route path="/quanto-custa" element={<QuantoCusta />} />
        <Route path="/cookies" element={<Cookies />} />
        <Route path="/t/:slug" element={<PerfilTalento />} />
        <Route path="/autorizar-perfil" element={<AutorizarPerfil />} />
        <Route path="/descadastrar" element={<Descadastrar />} />
        <Route path="/o/:slug" element={<OrganizadorPublico />} />
        <Route path="/convite/:token" element={<Convite />} />
        <Route path="/ingresso/:codigo/:token" element={<IngressoPublico />} />
        <Route path="/certificado/:codigo" element={<Certificado />} />
        <Route path="/pesquisa/:token" element={<Pesquisa />} />
        <Route path="/ajuda" element={<Ajuda />} />
        <Route path="/verificar" element={<VerificarIngresso />} />
        <Route path="/status" element={<Status />} />
        <Route path="/grupo/:token" element={<Grupo />} />
        <Route path="/login" element={<Login />} />
        <Route path="/cadastro" element={<Cadastro />} />

        <Route element={<ProtectedRoute />}>
          <Route path="/meu-talento" element={<MeuTalento />} />
          <Route path="/palco/:eventoId" element={<PalcoCamarim />} />
          <Route path="/porta/:eventoId" element={<VendaPorta />} />
          <Route path="/ao-vivo/:eventoId" element={<AoVivo />} />
          <Route path="/e/:slug/inscricao" element={<Inscricao />} />
          <Route path="/e/:slug/expositor" element={<CandidaturaExpositor />} />
          <Route path="/caravana/:codigo" element={<Caravana />} />
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
          <Route path="/organizador/equipe" element={<Equipe />} />
          <Route path="/organizador/coproducoes" element={<Coproducoes />} />
          <Route path="/organizador/api" element={<ApiKeys />} />
          <Route path="/organizador/relatorios" element={<Relatorios />} />
          <Route path="/organizador/circuitos" element={<MeusCircuitos />} />
          <Route path="/organizador/crm" element={<Crm />} />
          <Route path="/admin/repasses" element={<AdminRepasses />} />
          <Route path="/admin/relatorios" element={<AdminRelatorios />} />
          <Route path="/admin/organizadores" element={<AdminOrganizadores />} />
          <Route path="/admin/colecoes" element={<AdminColecoes />} />
          <Route path="/admin/avaliacoes" element={<AdminAvaliacoes />} />
          <Route path="/admin/suporte" element={<AdminSuporte />} />
          <Route path="/organizador/eventos/novo" element={<EventoNovo />} />
          <Route path="/organizador/eventos/:id" element={<EventoEditar />} />
          <Route path="/organizador/eventos/:id/credenciais/imprimir" element={<CredenciaisImprimir />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
