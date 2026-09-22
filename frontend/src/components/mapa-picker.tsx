import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png'
import markerIcon from 'leaflet/dist/images/marker-icon.png'
import markerShadow from 'leaflet/dist/images/marker-shadow.png'
import { useEffect, useState } from 'react'
import { MapContainer, Marker, TileLayer, useMap, useMapEvents } from 'react-leaflet'

// react-leaflet não resolve os ícones padrão com o empacotador do Vite —
// aponta pros assets já processados manualmente.
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
})

const CENTRO_PADRAO: [number, number] = [-9.6498, -35.7089] // Maceió/AL

type Props = {
  latitude: number | null
  longitude: number | null
  onSelecionar: (lat: number, lng: number) => void
}

function CliqueNoMapa({ onSelecionar }: { onSelecionar: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onSelecionar(e.latlng.lat, e.latlng.lng)
    },
  })
  return null
}

function RecentralizarMapa({ posicao }: { posicao: [number, number] }) {
  const mapa = useMap()
  useEffect(() => {
    mapa.setView(posicao)
  }, [mapa, posicao[0], posicao[1]])
  return null
}

export function MapaPicker({ latitude, longitude, onSelecionar }: Props) {
  const [busca, setBusca] = useState('')
  const [buscando, setBuscando] = useState(false)
  const posicao: [number, number] = latitude != null && longitude != null ? [latitude, longitude] : CENTRO_PADRAO

  const buscarEndereco = async () => {
    if (!busca.trim()) return
    setBuscando(true)
    try {
      const resposta = await fetch(
        `https://nominatim.openstreetmap.org/search?format=json&limit=1&q=${encodeURIComponent(busca)}`,
      )
      const resultados = await resposta.json()
      if (resultados[0]) {
        onSelecionar(Number(resultados[0].lat), Number(resultados[0].lon))
      }
    } finally {
      setBuscando(false)
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex gap-2">
        <input
          type="text"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), buscarEndereco())}
          placeholder="Buscar endereço…"
          className="h-8 flex-1 rounded-lg border border-border bg-background px-2.5 text-sm"
        />
        <button
          type="button"
          onClick={buscarEndereco}
          disabled={buscando}
          className="h-8 rounded-lg border border-border px-3 text-sm hover:bg-muted"
        >
          {buscando ? 'Buscando…' : 'Buscar'}
        </button>
      </div>
      <div className="h-64 overflow-hidden rounded-lg border border-border">
        <MapContainer center={posicao} zoom={13} className="h-full w-full">
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          {latitude != null && longitude != null && <Marker position={[latitude, longitude]} />}
          <CliqueNoMapa onSelecionar={onSelecionar} />
          <RecentralizarMapa posicao={posicao} />
        </MapContainer>
      </div>
      <p className="text-xs text-muted-foreground">Clique no mapa ou busque um endereço para marcar o local.</p>
    </div>
  )
}
