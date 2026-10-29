import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import '@/lib/leaflet-icones'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Circle, MapContainer, Marker, Popup, TileLayer, useMap } from 'react-leaflet'

import { distanciaKm, listarEventos } from '@/lib/descoberta'
import { EventoCard } from '@/components/evento-card'
import { ChipBotao } from '@/components/chip'
import { Button } from '@/components/ui/button'


const MACEIO = { lat: -9.6498, lng: -35.7089 }
const RAIOS = [10, 25, 50, 100]

function Centralizar({ centro, raio }: { centro: { lat: number; lng: number }; raio: number }) {
  const mapa = useMap()
  useEffect(() => {
    mapa.fitBounds(L.latLng(centro.lat, centro.lng).toBounds(raio * 2000))
  }, [mapa, centro.lat, centro.lng, raio])
  return null
}

function formatarDistancia(km: number) {
  return km < 1 ? 'a menos de 1 km' : `a ${km < 10 ? km.toFixed(1).replace('.', ',') : Math.round(km)} km`
}

// Mapa "perto de mim" (item 6.2). A localização só é pedida quando a pessoa
// clica no botão; até lá o mapa mostra Maceió.
export default function MapaEventos() {
  const [centro, setCentro] = useState(MACEIO)
  const [usandoLocalizacao, setUsandoLocalizacao] = useState(false)
  const [raio, setRaio] = useState(25)
  const [aviso, setAviso] = useState<string | null>(null)
  const [localizando, setLocalizando] = useState(false)

  useEffect(() => {
    document.title = 'Eventos perto de você | Evve'
  }, [])

  const { data, isLoading } = useQuery({
    queryKey: ['eventos-perto', centro.lat.toFixed(3), centro.lng.toFixed(3), raio],
    queryFn: () => listarEventos({ lat: centro.lat, lng: centro.lng, raioKm: raio, porPagina: 100 }),
  })
  const eventos = (data?.eventos ?? []).filter((e) => e.latitude != null && e.longitude != null)

  const usarLocalizacao = () => {
    setAviso(null)
    if (!('geolocation' in navigator)) {
      setAviso('Seu navegador não informa a localização.')
      return
    }
    setLocalizando(true)
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setCentro({ lat: pos.coords.latitude, lng: pos.coords.longitude })
        setUsandoLocalizacao(true)
        setLocalizando(false)
      },
      () => {
        setAviso('Não conseguimos sua localização. Mostrando eventos em Maceió.')
        setLocalizando(false)
      },
      { enableHighAccuracy: false, timeout: 10000, maximumAge: 600000 },
    )
  }

  return (
    <main className="mx-auto max-w-5xl px-4 py-10">
      <h1 className="text-3xl font-bold text-foreground text-balance sm:text-4xl">Eventos perto de você</h1>
      <p className="mt-1 text-muted-foreground">
        {usandoLocalizacao ? 'A partir da sua localização.' : 'Mostrando Maceió. Use sua localização para ver o que está mais perto.'}
      </p>

      <div className="mt-6 flex flex-wrap items-center gap-2">
        <Button onClick={usarLocalizacao} disabled={localizando}>
          {localizando ? 'Localizando…' : usandoLocalizacao ? 'Atualizar minha localização' : 'Usar minha localização'}
        </Button>
        <span className="ml-1 text-sm text-muted-foreground">Raio:</span>
        {RAIOS.map((r) => (
          <ChipBotao key={r} ativoAgora={raio === r} onClick={() => setRaio(r)}>
            {r} km
          </ChipBotao>
        ))}
      </div>
      {aviso && <p className="mt-2 text-sm text-destructive">{aviso}</p>}

      <div className="mt-6 h-96 overflow-hidden rounded-xl border border-border">
        <MapContainer center={[centro.lat, centro.lng]} zoom={12} scrollWheelZoom={false} className="h-full w-full">
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          <Centralizar centro={centro} raio={raio} />
          <Circle center={[centro.lat, centro.lng]} radius={raio * 1000} pathOptions={{ color: '#171717', weight: 1, fillOpacity: 0.04 }} />
          {eventos.map((e) => (
            <Marker key={e.slug} position={[e.latitude!, e.longitude!]}>
              <Popup>
                <Link to={`/e/${e.slug}`} className="font-medium">
                  {e.titulo}
                </Link>
                {e.inicio_em && <div>{new Date(e.inicio_em).toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short' })}</div>}
              </Popup>
            </Marker>
          ))}
        </MapContainer>
      </div>

      <h2 className="mb-3 mt-8 text-xl font-bold text-foreground">
        {isLoading ? 'Procurando eventos…' : `${eventos.length} evento(s) em até ${raio} km`}
      </h2>
      {!isLoading && eventos.length === 0 && (
        <p className="rounded-xl border border-border p-6 text-muted-foreground">
          Nenhum evento presencial nesse raio. Tente aumentar a distância.
        </p>
      )}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
        {eventos.map((e) => (
          <EventoCard key={e.slug} evento={e} detalhe={formatarDistancia(distanciaKm(centro, { lat: e.latitude!, lng: e.longitude! }))} />
        ))}
      </div>
    </main>
  )
}
