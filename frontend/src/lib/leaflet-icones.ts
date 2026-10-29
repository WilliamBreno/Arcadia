import L from 'leaflet'
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png'
import markerIcon from 'leaflet/dist/images/marker-icon.png'
import markerShadow from 'leaflet/dist/images/marker-shadow.png'

// O Leaflet tenta descobrir sozinho a pasta das imagens e junta esse
// caminho na frente da URL já processada pelo Vite, gerando
// ".../images//node_modules/.../marker-icon.png" (marcador quebrado).
// Removendo a detecção, ele usa exatamente as URLs abaixo.
delete (L.Icon.Default.prototype as unknown as { _getIconUrl?: unknown })._getIconUrl
L.Icon.Default.mergeOptions({ iconRetinaUrl: markerIcon2x, iconUrl: markerIcon, shadowUrl: markerShadow })
