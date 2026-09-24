// Service worker do leitor de check-in: deixa a tela abrir sem internet.
// Só guarda o "casco" do app (HTML + JS/CSS/fontes com hash). Nunca guarda
// /api (dados do evento ficam no IndexedDB, só no aparelho da portaria).
const CACHE = 'evve-shell-v1'

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE).then((c) => c.add('/')))
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((chaves) => Promise.all(chaves.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (event) => {
  const req = event.request
  const url = new URL(req.url)
  if (req.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api')) return

  // Navegação (qualquer rota da SPA): rede primeiro, cai no index em cache.
  if (req.mode === 'navigate') {
    event.respondWith(
      fetch(req)
        .then((resp) => {
          const copia = resp.clone()
          caches.open(CACHE).then((c) => c.put('/', copia))
          return resp
        })
        .catch(() => caches.match('/')),
    )
    return
  }

  // Assets com hash (imutáveis): cache primeiro, atualiza no fundo.
  if (url.pathname.startsWith('/assets/')) {
    event.respondWith(
      caches.match(req).then(
        (hit) =>
          hit ||
          fetch(req).then((resp) => {
            const copia = resp.clone()
            caches.open(CACHE).then((c) => c.put(req, copia))
            return resp
          }),
      ),
    )
  }
})
