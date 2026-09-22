# Arcadia — Plataforma de Eventos e Ingressos

Sistema de gerenciamento de eventos e venda de ingressos, com gestão de jurados e participantes (cosplay, dança, canto, atuação) como diferencial em relação a plataformas como Sympla e Even3.

Veja o plano completo em [`plano-eventos.md`](./plano-eventos.md).

## Estrutura do repositório

```
/
├── backend/            API em Go (Gin + GORM)
├── frontend/           SPA React + TypeScript + Vite
├── plano-eventos.md    plano de implementação (fonte da verdade)
├── CLAUDE.md           resumo do projeto para o Claude Code
└── docker-compose.yml  Postgres local
```

## Como rodar localmente

### Pré-requisitos

- Go 1.26+
- Node.js 20+
- Docker (para o Postgres local)

### Backend

```bash
cd backend
cp .env.example .env
go run ./cmd/api
```

A API sobe em `http://localhost:8080`. Teste com `GET /healthz`.

### Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

O frontend sobe em `http://localhost:5173` e faz proxy de `/api` para `http://localhost:8080`.

### Banco de dados (Postgres local via Docker)

```bash
docker compose up -d
```

## Deploy

- Backend: Render (root directory `backend/`)
- Frontend: Vercel (root directory `frontend/`)
  - Configurar a env `BACKEND_API_URL` no projeto Vercel (URL da API em produção) — usada só pelo `middleware.ts` (meta tags Open Graph em `/e/:slug`), não pelo bundle do cliente.
- Postgres: Neon
