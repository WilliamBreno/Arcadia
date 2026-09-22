# Migrations

SQL puro, gerenciado por [golang-migrate](https://github.com/golang-migrate/migrate).

## Convenção

- Arquivos `NNNNNN_descricao.up.sql` e `NNNNNN_descricao.down.sql`, numeração sequencial com 6 dígitos.
- IDs de tabela: `bigserial` (autoincremento), nunca `uuid` (decisão registrada na seção 14 do `plano-eventos.md`).
- Seeds não inserem IDs explícitos. Quando for inevitável, rodar `setval()` na sequence ao final do arquivo.

## Criar uma nova migration

```bash
npx -y migrate create -ext sql -dir migrations -seq nome_da_migration
```

(ou instale o binário `migrate` localmente, se preferir não depender do `npx`)

## Rodar

```bash
cd backend
go run ./cmd/migrate up
go run ./cmd/migrate down
```

Lê `DATABASE_URL` do ambiente (ver `.env.example`).
