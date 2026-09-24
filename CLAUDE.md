# Evve — Plataforma de Eventos e Ingressos

Sistema de gerenciamento de eventos e venda de ingressos, com gestão de jurados e participantes (concursos de cosplay, dança, canto e atuação) como diferencial.

**Leia `plano-eventos.md` antes de qualquer tarefa.** Ele é a fonte da verdade: decisões de negócio, modelo de dados, regras financeiras, telas e o checklist de fases.

**Comando "Pode atualizar"** = implementar o próximo item pendente do checklist (seção 11 do plano), com commit e push.

## Estrutura

- `backend/` — API em Go (Gin + GORM)
- `frontend/` — SPA em React + TypeScript + Vite
- `plano-eventos.md` — plano completo (fonte da verdade)

## Regras que valem sempre

- Dinheiro sempre em **centavos (`int64`)**, nunca `float`. Cálculo sempre no backend.
- Operações financeiras são **idempotentes** e rodam em **transação**.
- Dúvida sobre dinheiro, cancelamento ou dados pessoais: parar e perguntar.
- Nenhum segredo no repositório — apenas `.env.example`.
- Commit e push no GitHub após cada item concluído do checklist.
- Não avançar de fase sem o dono pedir.
