# Plataforma de Eventos e Ingressos — Plano de implementação para o Claude Code

> **Nome do produto:** ainda não definido. Use a constante `NOME_PLATAFORMA` (config/env) no código e na UI, sem fixar um nome.
> **Idioma:** UI e mensagens em pt-BR. Nomes de domínio no código em português (`Usuario`, `Evento`, `Ingresso`, `Pedido`...).
> **Este arquivo é a fonte da verdade.** Coloque-o na raiz do repositório como `plano-eventos.md` e referencie-o no `CLAUDE.md`.

---

## 0. Como trabalhar neste projeto (leia antes de escrever código)

1. Leia o arquivo inteiro. As seções **2 (decisões fechadas)** e **7 (regras financeiras)** valem mais que qualquer suposição sua.
2. Implemente **por fase e por item** (seção 11), na ordem. Ao concluir um item, marque `[x]`. Não avance para a fase seguinte sem o dono pedir.
3. **Comando "Pode atualizar":** implemente o próximo item pendente (não marcado), marque `[x]`, e ao final resuma (a) o que foi feito, (b) o que o dono precisa testar manualmente, (c) qualquer decisão que você tomou por conta própria (registre também na seção 14).
4. **Dinheiro:** nunca use `float`. Todo valor em **centavos (`int64`)**. Todo cálculo de preço, taxa, reembolso e repasse é feito **no backend** (nunca confie no frontend). Toda mutação financeira é **idempotente** e roda **dentro de transação**. Escreva testes unitários para cálculo de preço, reembolso e repasse.
5. Se a especificação for ambígua em algo que envolva **dinheiro, cancelamento ou dados pessoais**, pare e pergunte. Para o resto, escolha a opção mais simples e registre na seção 14.
6. Antes de dar um item por concluído: `go build ./... && go vet ./... && go test ./...` e `tsc -b && npm run build` sem erros.
7. **Frontend:** use classes da escala nomeada do Tailwind (`w-80`, `p-4`). **Evite valores arbitrários com colchetes** (`w-[347px]`), pois quebram o parser JSX quando o dono edita pelo editor web do GitHub no celular. Se precisar de valor exato, use `style` inline.
8. **Migrations/seeds:** não insira IDs explícitos em seeds. Se for inevitável, rode `setval()` nas sequences ao final (o dono já teve *duplicate key* por sequence defasada em outros projetos).
9. Nunca commite segredos. Toda credencial vem de variáveis de ambiente; mantenha um `.env.example` atualizado.
10. **COMMIT E PUSH NO GITHUB (obrigatório, não esqueça):** ao concluir **cada item** do checklist (e depois de build/testes passarem), faça `git add`, **commit** e **`git push`**. Ao fechar uma fase, faça também uma **tag** e dê push nela. Detalhes na seção 4.1. Nunca deixe trabalho concluído só na máquina local. Se o `push` falhar (remoto ausente, autenticação, conflito), pare e avise o dono em vez de seguir sem push.

---

## 1. Visão do produto

Sistema completo de **gerenciamento de eventos e venda de ingressos**, bonito e agradável para o público, simples para quem organiza e completo em recursos.

- **Organizadores** criam eventos (para um local ou online), definem se o evento terá **venda de ingressos** ou apenas **cadastro (inscrição)**, publicam e divulgam.
- **Público** navega por todos os eventos publicados de todos os organizadores, compra ingresso ou faz cadastro, e recebe um QR code.
- **Jurados** e **participantes especiais** são convidados por link gerado pelo organizador. **Participantes (talentos)** também podem se inscrever pela página do evento, conforme o modo escolhido pelo organizador. Cada participante escolhe um tipo de apresentação (cosplay, dança, canto, atuação) com campos específicos.
- **Jurados** acessam o evento, veem todos os participantes e seus detalhes (na Fase 3, também dão notas).
- **Login obrigatório** para identificar jurados, participantes e plateia. Em "Meus eventos" cada pessoa vê os eventos em que tem algum papel, com selo (Jurado, Participante, Participante especial, Ingresso).
- **Modelo de negócio:** taxa fixa por ingresso/cadastro + **Garantia de vaga** opcional (cancelamento livre até o início do evento).

Referências estudadas (o que copiar e o que superar):
- **Sympla:** lotes, cupons, meia-entrada, cortesias, check-in por QR com ingresso nominal, repasse após o evento, categorias (inclui "Games e Geek"). Não tem gestão de jurados/participantes: esse é o diferencial.
- **Even3:** "garantia de reembolso" (só após o evento e por motivos comprovados). A nossa é mais forte: cancelamento livre até o início do evento.
- **Luma:** UX moderna, mobile-first, aprovação de inscrição e lista de espera. É o benchmark de "bonito e simples".
- **Concursos de cosplay:** categorias desfile/apresentação, referência obrigatória, notas por quesito com pesos, desempate, autorização de menores.

---

## 2. Decisões de negócio FECHADAS (não mudar sem consultar o dono)

### 2.1 Taxas (valores em config, nunca hardcoded)
| Item | Valor | Config |
|---|---|---|
| Taxa por **ingresso** | R$ 0,99 por ingresso | `TAXA_INGRESSO_CENTAVOS=99` |
| Taxa por **cadastro** | R$ 0,49 por cadastro | `TAXA_CADASTRO_CENTAVOS=49` |
| **Garantia de vaga** (opcional) | R$ 1,99 por ingresso/cadastro | `GARANTIA_CENTAVOS=199` |

- A taxa é **fixa da plataforma**. O organizador define livremente o **preço** do ingresso/cadastro (pode ser R$ 0,00 em `tipo_acesso=cadastro`).
- **Quem paga a taxa é o comprador**, somada ao preço. O valor final (preço + taxa + garantia opcional) deve aparecer **discriminado e destacado antes do pagamento**, na página do evento ("a partir de R$ X + taxa") e no resumo do checkout. Jamais surgir só na última tela.
- Evento com `tipo_acesso=ingresso` cobra a taxa de ingresso; com `tipo_acesso=cadastro`, a de cadastro. Eventos gratuitos devem usar `cadastro`.
- O organizador **sempre vê no painel** quanto o comprador paga de taxa e quanto ele mesmo recebe (líquido).

### 2.2 Garantia de vaga (o "seguro")
- **Nome na UI:** "Garantia de vaga" ou "Cancelamento flexível". **Nunca chamar de "seguro"** (atividade regulada; validar com contador/advogado).
- **Opt-in por item, desmarcada por padrão**. Nunca pré-selecionada (evitar venda casada).
- Com garantia contratada: o comprador pode **cancelar o item a qualquer momento até o início do evento** (data/hora de início do primeiro dia) e recebe **tudo o que pagou pelo item**: preço + taxa + garantia.
- Ao cancelar, a **vaga volta ao estoque** automaticamente.
- Não é possível cancelar item já utilizado (check-in feito).
- Sem garantia: vale só o **direito de arrependimento do CDC (art. 49)** — ver 7.4. Fora dessa janela, não há reembolso.

### 2.3 Fluxo do dinheiro: CUSTÓDIA
- A **plataforma recebe 100% dos pagamentos** na própria conta Mercado Pago (Checkout Pro, **sem split, sem OAuth de vendedor, sem `marketplace_fee`**).
- O organizador recebe por **repasse após o término do evento** (padrão: **3 dias úteis** após o fim, configurável em `REPASSE_DIAS_UTEIS=3`), via **Pix manual/em lote** feito pelo admin da plataforma, com painel de "a receber / histórico / marcar como pago" (mesmo padrão de repasse manual já usado em outro projeto do dono).
- O organizador **arca com a taxa do processador de pagamento**. Ela é descontada do repasse (ver 7.7). A taxa fixa e a garantia ficam com a plataforma.
- Atenção regulatória: reter dinheiro de terceiros pode ter implicações. Deixe o modelo isolado em um serviço (`repasse_service`) para poder migrar para outro arranjo no futuro. Registre a pendência "validar com contador".

### 2.4 Participantes (talentos do concurso)
- O organizador escolhe **por evento** o `modo_participantes`: `nenhum | convite | inscricao_aberta | ambos`.
- **Convite (participante especial):** link gerado pelo organizador. A pessoa confirma e preenche a ficha.
- **Inscrição aberta:** qualquer usuário logado se inscreve pela página do evento; o organizador **aprova ou rejeita** (com lista de espera opcional). Há limite de vagas por tipo/categoria e prazo de inscrição.
- **Jurados** entram **somente por convite**.

### 2.5 Legal / conformidade (mapeado na pesquisa)
- Taxa de conveniência é aceita pelo STJ se o consumidor for informado **previamente do preço total, com o valor da taxa em destaque**. Alguns estados restringem (levantamento inicial na seção 7.8, a validar com advogado).
- **Meia-entrada** (Lei 12.933/2013): cota de 40% dos ingressos para estudantes, PcD e jovens de baixa renda; idosos sem limite. Implementar na Fase 3.
- **Menores de 18 anos** só participam com autorização do responsável.

---

## 3. Papéis e permissões

**O papel é POR EVENTO, não por conta.** Uma mesma conta pode ser jurada no evento A, plateia no B e organizadora no C. Isso é modelado em `papeis_evento`. Só `admin_plataforma` é um papel global.

| Papel | Como vira | Pode |
|---|---|---|
| Visitante (sem login) | — | Ver home, busca e página do evento. |
| Público | Comprar ingresso/cadastro | Ver "Meus ingressos" (QR), cancelar conforme regras 7.4. |
| Participante | Inscrição aberta aprovada | Editar a própria ficha até o prazo; ver o próprio status. |
| Participante especial | Convite | Igual a Participante, com selo próprio. |
| Jurado | Convite | Ver todos os participantes aprovados do evento e seus detalhes (na Fase 3: avaliar). |
| Staff de portaria | Convite do organizador | Só check-in (leitor de QR e busca). |
| Organizador | Criar perfil de organizador | CRUD dos próprios eventos, convites, aprovação de participantes, vendas, financeiro. |
| Admin da plataforma | Manual | Tudo: organizadores, eventos, repasses, reembolsos manuais, regras regionais, moderação. |

**Privacidade:** o jurado vê nome, nome artístico, Instagram, idade, foto, tipo de apresentação, referência e demais dados da apresentação. **Nunca** vê telefone, e-mail, documentos ou autorização do responsável.

---

## 4. Stack e convenções

- **Backend:** Go (Gin + GORM), PostgreSQL, JWT (access curto + refresh em cookie httpOnly). Jobs agendados via rota protegida por `X-Cron-Secret` (padrão já usado pelo dono) ou goroutines com tabela de outbox.
- **Frontend:** React + TypeScript + Vite + Tailwind + **shadcn/ui**. Magic UI / React Bits / 21st.dev como inspiração e efeitos (apenas em páginas de marketing). React Router, TanStack Query, react-hook-form + zod.
- **Pagamentos:** Mercado Pago Checkout Pro (conta da plataforma), Pix e cartão. Reembolso via API de refunds (parcial por item).
- **E-mail:** Resend. **Imagens:** object storage (Supabase Storage ou Cloudflare R2), com redimensionamento e otimização no upload.
- **Mapas:** Leaflet + OpenStreetMap/Nominatim (sem Google Maps, para evitar custo).
- **QR:** geração no backend; leitor no frontend por câmera, como **PWA** (o check-in roda no celular da portaria).
- **Deploy sugerido:** backend no Render, frontend na Vercel, Postgres no Neon. Docker + `docker-compose` para desenvolvimento local, e GitHub Actions rodando `go vet`, `go test` e o build do frontend.
- **SEO/compartilhamento:** a página `/e/:slug` precisa devolver **meta tags Open Graph** para crawlers (WhatsApp, Instagram). Como o frontend é SPA, implemente via Vercel Edge Middleware ou função que busca o evento na API e injeta as meta tags.
- **Estrutura de pastas (monorepo, um único repositório GitHub):**
  - `backend/` → `cmd/`, `internal/{config,domain,repository,service,handler,middleware,jobs,mercadopago,storage,mail}`
  - `frontend/` → `src/{pages,components,features,lib,hooks}`

### 4.1 Repositório, pastas e Git (fazer na Fase 0)

**Estrutura da raiz do repositório:**
```
/                       (raiz do repositório)
├── backend/            API em Go (Gin + GORM)
├── frontend/           SPA React + TypeScript + Vite
├── plano-eventos.md    este plano (fonte da verdade)
├── CLAUDE.md           resumo curto + referência ao plano
├── README.md           como rodar backend e frontend localmente
├── docker-compose.yml  Postgres local
├── .gitignore
└── .gitattributes
```

**Criação das pastas:**
- `backend/`: `go mod init github.com/WilliamBreno/<nome-do-repo>/backend`, servidor Gin mínimo na porta 8080 com rota `GET /healthz`, config por env e `.env.example`.
- `frontend/`: `npm create vite@latest frontend -- --template react-ts`, depois Tailwind, shadcn/ui, React Router e TanStack Query. Em desenvolvimento, proxy de `/api` para `http://localhost:8080`. Criar `frontend/.env.example`.
- Deploy: no Render o *root directory* é `backend/`; na Vercel o *root directory* é `frontend/`.

**Git e GitHub:**
1. Rode `git status`. Se a pasta ainda não for um repositório, `git init -b main`.
2. Se não houver remoto `origin`, **pergunte ao dono a URL do repositório GitHub** (perfil `github.com/WilliamBreno`). **Não invente URL** e não crie repositório sem ele pedir.
3. Crie `.gitignore` cobrindo: `node_modules/`, `dist/`, `.env`, `.env.*` (exceto `.env.example`), binários Go, `*.log`, `.DS_Store`, `.vscode/` (exceto configs compartilhadas), pastas de upload local e `coverage/`.
4. Crie `.gitattributes` com `* text=auto eol=lf` (o desenvolvimento é em Windows; evita aviso e diff de CRLF).
5. Primeiro commit com a estrutura e **push** (`git push -u origin main`).
6. **Regra permanente:** um commit por item do checklist, mensagem em pt-BR no padrão *Conventional Commits* e com o número do item. Exemplos: `feat(eventos): 1.3 CRUD de eventos em etapas`, `fix(checkout): idempotência do webhook`, `chore: 0.1 estrutura backend e frontend`, `docs: atualiza plano`.
7. **Push imediatamente após cada commit.** Ao fechar cada fase: `git tag fase-N` e `git push --tags`.
8. Antes de cada commit: `git status` para conferir que nenhum `.env`, segredo ou arquivo gerado entrou. Rode build e testes (seção 0, item 6).
9. Trabalho direto na `main` é aceitável (projeto de um único desenvolvedor). Se um item for grande ou arriscado (ex.: 1.7 checkout), use branch `feat/<item>` e faça merge após validar.
10. Ao final de cada resposta que concluir um item, informe: hash do commit, se o push foi feito e o que o dono precisa testar.

---

## 5. Modelo de dados (campos principais)

Todos os valores monetários em **centavos (`int64`)**. Datas em UTC no banco, com `timezone` no evento (padrão `America/Maceio`). IDs: `uuid` ou `bigserial` (escolha um e seja consistente).

| Tabela | Campos principais |
|---|---|
| `usuarios` | id, nome, email (único), senha_hash (nullable p/ login Google), google_id, telefone, avatar_url, email_verificado_em, papel_plataforma (`usuario`\|`admin_plataforma`), criado_em |
| `organizadores` | id, usuario_id, nome_publico, slug, descricao, logo_url, tipo_pessoa (`pf`\|`pj`), documento (CPF/CNPJ), chave_pix, tipo_chave_pix, instagram, site, status (`pendente`\|`ativo`\|`suspenso`) |
| `locais` | id, organizador_id, nome, logradouro, numero, bairro, cidade, uf, cep, latitude, longitude, capacidade, observacoes |
| `eventos` | id, organizador_id, local_id (nullable = online), titulo, slug (único), descricao, categoria, capa_url, inicio_em, fim_em, timezone, classificacao_etaria, visibilidade (`publico`\|`nao_listado`\|`privado`), **tipo_acesso** (`ingresso`\|`cadastro`), status, **modo_participantes** (`nenhum`\|`convite`\|`inscricao_aberta`\|`ambos`), inscricao_talentos_inicio/fim, capacidade_total, **garantia_habilitada** (bool, padrão true), politica_cancelamento_texto, max_itens_por_pedido, publicado_em, cancelado_em, motivo_cancelamento |
| `tipos_ingresso` | id, evento_id, nome, descricao, preco_centavos, quantidade, vendas_inicio, vendas_fim, min_por_pedido, max_por_pedido, ordem, ativo (Fase 3: lote_grupo, meia_entrada) |
| `pedidos` | id, usuario_id, evento_id, status, total_centavos, expira_em, mp_preference_id, criado_em |
| `itens_pedido` (ingressos emitidos) | id, pedido_id, tipo_ingresso_id, titular_nome, titular_email, preco_centavos, taxa_plataforma_centavos, garantia_contratada, garantia_centavos, total_centavos, status, codigo (único), qr_token, utilizado_em, cancelado_em, motivo_cancelamento |
| `pagamentos` | id, pedido_id, mp_payment_id (único), metodo (`pix`\|`cartao`), status, valor_centavos, taxa_processador_centavos, payload_json, criado_em |
| `reembolsos` | id, pagamento_id, item_id, valor_centavos, tipo (`garantia`\|`arrependimento`\|`evento_cancelado`\|`manual`), status, mp_refund_id, motivo, solicitado_por, criado_em |
| `lancamentos` (ledger) | id, tipo, valor_centavos, sinal (+/−), evento_id, organizador_id, pedido_id, item_id, criado_em. Tipos: `venda_preco`, `taxa_plataforma`, `garantia`, `taxa_processador`, `reembolso_preco`, `reembolso_taxa`, `reembolso_garantia`, `custo_processador_perdido`, `repasse` |
| `repasses` | id, evento_id, organizador_id, valor_bruto_centavos, taxa_processador_centavos, valor_liquido_centavos, status (`calculado`\|`pendente`\|`pago`\|`cancelado`), liberar_em, pago_em, comprovante_url, observacao |
| `convites` | id, evento_id, tipo (`jurado`\|`participante_especial`), token_hash, max_usos (nullable = ilimitado), usos, expira_em, revogado_em, criado_por |
| `papeis_evento` | id, evento_id, usuario_id, papel (`organizador`\|`jurado`\|`participante`\|`staff`), origem (`convite`\|`inscricao`), status (`pendente`\|`confirmado`\|`removido`), convite_id. Único (evento, usuario, papel) |
| `fichas_participacao` | id, evento_id, usuario_id, papel (`jurado`\|`participante`), origem, nome, nome_artistico, instagram, data_nascimento, foto_url, telefone (privado), status (`rascunho`\|`pendente`\|`aprovado`\|`rejeitado`\|`lista_espera`\|`desistiu`), motivo_rejeicao, ordem_apresentacao; só participante: **tipo_apresentacao** (`cosplay`\|`danca`\|`canto`\|`atuacao`), dados (jsonb validado por tipo), responsavel_nome, responsavel_contato, autorizacao_responsavel_url (privado) |
| `midias_ficha` | id, ficha_id, tipo (`foto_referencia`\|`foto_cosplay`\|`audio`), url, ordem |
| `checkins` | id, item_id, staff_usuario_id, dispositivo, resultado, criado_em |
| `regras_regionais` | uf (e municipio opcional), permite_taxa, taxa_maxima_percentual, exige_canal_sem_taxa, excecao_publico_ate, observacao, fonte, vigente_desde |
| `config_plataforma` | chave, valor (taxas, garantia, dias de repasse, minutos de reserva, flags) |
| `notificacoes_outbox` | id, usuario_id, tipo, payload, status, tentativas |
| `auditoria` | id, ator_id, acao, entidade, entidade_id, diff, ip, criado_em |
| Fase 3: `avaliacao_criterios`, `avaliacoes` | criterio (evento_id, nome, peso, nota_min, nota_max, passo, tipo_apresentacao opcional); avaliacao (ficha_id, jurado_usuario_id, criterio_id, nota, comentario, finalizada) |

### Campos específicos por `tipo_apresentacao` (validar com zod no front e no Go)
| Tipo | Campos em `dados` |
|---|---|
| **Cosplay** | personagem, obra_origem, categoria (`desfile`\|`apresentacao`), **foto de referência obrigatória** (1 a 3, em `midias_ficha`; fanart não é aceita, só material oficial), foto do cosplay (opcional), música (só se apresentação) |
| **Dança** | estilo, modalidade (`solo`\|`grupo`), num_integrantes, **instagram_grupo** (ou do dançarino/dançarina; **sem foto de referência**), música |
| **Canto** | musica, artista, formato (`ao_vivo`\|`playback`), duracao_estimada |
| **Atuação** | titulo_cena, duracao_estimada, necessidades_palco |

Campos comuns a todos: nome, nome artístico, Instagram, data de nascimento (idade calculada), foto opcional, telefone. Ficha de **jurado**: apenas os comuns.

---

## 6. Máquinas de estado

- **Evento:** `rascunho → publicado → encerrado`; `cancelado` (terminal). "Vendas abertas/encerradas" e "em andamento" são **derivados** das datas. Para publicar: local ou online, `inicio_em` no futuro, ao menos 1 tipo de ingresso, organizador ativo com chave Pix e documento cadastrados (se for evento com valor > 0), regras regionais OK (7.8).
- **Pedido:** `aberto → aguardando_pagamento → pago`; `expirado`, `cancelado`, `reembolsado_parcial`, `reembolsado`.
- **Item (ingresso):** `reservado → pago → utilizado`; `cancelado`, `reembolsado`, `expirado`.
- **Ficha de participante:** `rascunho → pendente → aprovado | rejeitado | lista_espera`; `desistiu`. Em modo convite, a ficha nasce já como `aprovado` ao ser confirmada (o organizador pode remover).
- **Convite:** ativo, esgotado, expirado, revogado.
- **Repasse:** `calculado → pendente → pago`; `cancelado`.

---

## 7. Regras de negócio e financeiras (CRÍTICO)

### 7.1 Cálculo de preço (por item)
`total_item = preco_organizador + taxa_plataforma (99 ou 49) + garantia (199 se contratada)`
`total_pedido = soma(total_item)`
- O preço de cada item é **recalculado no servidor** a partir do `tipo_ingresso` e da config. Ignorar qualquer valor vindo do cliente.
- Valide no sandbox do Mercado Pago o **valor mínimo de pagamento** (Pix e cartão) para o caso de cadastro gratuito (total R$ 0,49).

### 7.2 Estoque e reserva
- Ao iniciar o checkout, **reservar** os itens por `RESERVA_MINUTOS=15` (itens `reservado`, pedido com `expira_em`). Use operação atômica/lock (`SELECT ... FOR UPDATE` ou `UPDATE ... WHERE disponivel >= n`) para impedir overselling.
- Pedido não pago expira: itens `expirado`, estoque devolvido (job a cada minuto). O prazo do Pix deve casar com a reserva.
- Respeitar `max_por_pedido`, janela de vendas e capacidade total do evento.
- Cada ingresso é **nominal**: `titular_nome` e `titular_email` (padrão: o comprador no primeiro item; os demais podem ser preenchidos depois).

### 7.3 Pagamento (Mercado Pago, conta da plataforma)
- Criar `preference` com `external_reference = pedido_id`, sem split e sem `marketplace_fee`. Métodos: Pix e cartão (à vista no MVP).
- **Webhook:** validar assinatura/segredo, **consultar o pagamento na API** (nunca confiar no corpo da notificação), processar de forma **idempotente** por `mp_payment_id`.
- Ao aprovar: itens → `pago`, gerar QR, enviar e-mail com ingresso, registrar `lancamentos` (`venda_preco`, `taxa_plataforma`, `garantia`) e guardar `taxa_processador_centavos` a partir dos detalhes de taxa do pagamento.

### 7.4 Cancelamento pelo comprador (por item)
Avaliar nesta ordem:
1. Item `utilizado` → **negado**.
2. `garantia_contratada` e `agora < evento.inicio_em` → **reembolso total do item** (preço + taxa + garantia). Tipo `garantia`.
3. Sem garantia, mas **≤ 7 dias corridos da compra** **e** `agora ≤ inicio_em − 48h` → **reembolso total do item** (direito de arrependimento, CDC art. 49). Tipo `arrependimento`. Config `CDC_REEMBOLSA_TAXA=true` (padrão: devolve também a taxa; validar com advogado se a taxa pode ser retida).
4. Caso contrário → **negado**, com mensagem clara na UI (e, se o evento tem garantia habilitada, explicar que ela só pode ser contratada na compra).

O organizador pode oferecer política **mais generosa**, nunca menos que o CDC. Ao cancelar: item → `cancelado`, **estoque liberado**, reembolso **parcial** via API do MP (um pagamento pode conter vários itens), lançamentos de estorno, e-mail ao comprador. **Invariante:** a soma dos reembolsos de um pagamento nunca excede o valor pago.

### 7.5 Evento cancelado ou alterado pelo organizador
- **Cancelado:** todos os itens pagos são **reembolsados integralmente** (preço + taxa + garantia) por job com retry, e painel de falhas para o admin. O repasse do evento considera apenas itens ativos.
- **Data ou local alterado:** notificar todos os compradores e abrir **janela de 7 dias** de cancelamento com reembolso integral, mesmo sem garantia.

### 7.6 Repasse ao organizador
- Gerado por job quando `agora ≥ fim do último dia do evento + REPASSE_DIAS_UTEIS`.
- `valor_bruto = soma(preco_organizador dos itens em pago|utilizado)`
- `valor_liquido = valor_bruto − taxa_processador_alocada` (7.7)
- O admin paga via Pix (chave do organizador) e marca como `pago` com comprovante. O organizador acompanha em painel dedicado (extrato: a receber, histórico).
- Chargeback/disputa após o repasse: registrar em `lancamentos` e alertar o admin (MVP: tratamento manual; possível desconto em repasses futuros).
- **Receita da plataforma** (relatório admin): `taxa_plataforma + garantia` dos itens `pago|utilizado` cujo evento já começou.

### 7.7 Taxa do processador (`POLITICA_TAXA_PROCESSADOR=organizador`)
- A taxa real do MP de cada pagamento é **alocada proporcionalmente** ao `total_item` de cada item; a parcela dos itens **ativos** é descontada do organizador.
- Itens cancelados/reembolsados: se o MP **não devolver** a taxa no estorno, o custo é da **plataforma**, via lançamento `custo_processador_perdido`.
- **Tarefa de verificação (sandbox/produção):** confirmar se o MP devolve a taxa em estornos e ajustar o cálculo.

### 7.8 Regras regionais (levantamento inicial, **validar com advogado**)
Semear `regras_regionais` com: **AC** e **RR** (proíbem taxa de conveniência online); **ES** (proíbe, exceto se houver canal alternativo sem taxa; eventos até 200 pessoas dispensados); **AL** (limite de 10% do valor do ingresso; canal sem taxa para eventos maiores); **Fortaleza/CE** (exige ao menos um canal sem taxa). Não encontrei lei específica de **SE**.
Comportamento no MVP: ao publicar evento pago numa UF com restrição, **bloquear a publicação e exibir o motivo** ao organizador e ao admin (o admin pode liberar manualmente). Onde há limite percentual, alertar se `taxa > limite × preço`.

### 7.9 Meia-entrada (Fase 3)
Tipo de ingresso "Meia". Cota de 40% do total para estudantes/PcD/jovem baixa renda (sem limite para idosos). Comprovação **na entrada**: o check-in exibe alerta "conferir documento". A taxa fixa não muda.

### 7.10 Check-in
- O QR contém `codigo` + assinatura (HMAC, `qr_token`). O leitor chama endpoint que faz **`UPDATE itens SET status='utilizado' WHERE id=? AND status='pago'`** de forma atômica.
- Respostas: ✅ válido / ⚠️ já utilizado (mostrar quando e por quem) / ❌ cancelado / ❌ outro evento / ❌ não encontrado. Mostrar nome do titular e tipo do ingresso.
- Busca manual por nome, e-mail ou código. Contador de entradas em tempo real. Operado por `staff` ou `organizador`.

### 7.11 Convites
- Link `/convite/:token` (token de 32 bytes, base64url; **salvar apenas o hash**). Um link por tipo (`jurado` ou `participante_especial`), com `max_usos` opcional, `expira_em` e revogação.
- Fluxo: abrir link → ver evento e tipo do convite → login ou cadastro → preencher ficha → **confirmar** → papel `confirmado` e evento aparece em "Meus eventos" com selo. Reuso pelo mesmo usuário não duplica.
- O organizador vê a lista de convidados, pode remover pessoas e revogar links.

### 7.12 Participantes
- Inscrição aberta: respeitar prazo, **vagas por tipo/categoria** e aprovação manual. E-mail ao participante em cada mudança de status.
- **Menores de 18** (calculado por `data_nascimento`): exigir nome e contato do responsável e **upload da autorização** (arquivo privado, acesso por URL assinada).
- Jurado vê apenas fichas `aprovado`. Ordem de apresentação editável pelo organizador (Fase 3).
- Inscrição de talento **não paga taxa da plataforma** no MVP (ver seção 14).

---

## 8. API (prefixo `/api/v1`, resumo por módulo)

- **Auth:** `POST /auth/cadastro`, `/auth/login`, `/auth/google`, `/auth/refresh`, `/auth/logout`, `/auth/verificar-email`, `/auth/esqueci-senha`, `/auth/redefinir-senha`
- **Público:** `GET /eventos` (filtros: q, cidade, uf, categoria, data_de/data_ate, atalho `hoje|amanha|fim-de-semana`, gratuito, ordenação, paginação), `GET /eventos/:slug`, `GET /organizadores/:slug`, `GET /categorias`
- **Conta:** `GET/PUT /me`, `GET /me/eventos` (com papéis/selos), `GET /me/ingressos`, `GET /me/ingressos/:id` (QR)
- **Checkout:** `POST /eventos/:id/pedidos` (reserva + cálculo), `GET /pedidos/:id`, `POST /pedidos/:id/pagar`, `POST /webhooks/mercadopago`
- **Cancelamento:** `GET /itens/:id/cancelamento` (simula: pode? quanto volta?), `POST /itens/:id/cancelar`
- **Convites/fichas:** `GET /convites/:token`, `POST /convites/:token/aceitar`, `GET/PUT /eventos/:id/minha-ficha`, `POST /eventos/:id/inscricao` (inscrição aberta), `POST /uploads` (URL assinada)
- **Jurado:** `GET /eventos/:id/participantes` (somente jurado confirmado), `GET /eventos/:id/participantes/:fichaId`
- **Organizador:** CRUD `/org/locais`, `/org/eventos`, `/org/eventos/:id/ingressos`, `/org/eventos/:id/convites`, `/org/eventos/:id/participantes` (+ aprovar/rejeitar/reordenar), `/org/eventos/:id/vendas`, `/org/eventos/:id/financeiro`, `/org/eventos/:id/staff`, `/org/eventos/:id/cancelar`, `/org/eventos/:id/exportar.csv`
- **Check-in:** `POST /checkin/validar`, `GET /checkin/eventos/:id/resumo`, `GET /checkin/eventos/:id/busca`
- **Admin:** `/admin/organizadores`, `/admin/eventos`, `/admin/repasses` (+ marcar pago), `/admin/reembolsos`, `/admin/regras-regionais`, `/admin/config`, `/admin/relatorios`
- **Jobs (X-Cron-Secret):** `/jobs/expirar-reservas`, `/jobs/gerar-repasses`, `/jobs/lembretes`, `/jobs/reprocessar-reembolsos`, `/jobs/outbox`

---

## 9. Telas

**Públicas:** Home (busca, filtros, destaques por categoria e "este fim de semana") · Lista/busca de eventos · Página do evento (capa, descrição, programação, local com mapa, ingressos, organizador, botão compartilhar) · Página do organizador · Login/Cadastro · Recuperar senha · Termos e Privacidade.

**Checkout:** seleção de ingressos → dados do titular → resumo com **preço + taxa + garantia opcional (desmarcada)** e total destacado → pagamento (Pix/cartão) → confirmação com QR.

**Conta:** Meus ingressos (QR, cancelar, quanto volta) · **Meus eventos** (cards com selo: Jurado / Participante / Participante especial / Ingresso) · Minha ficha em cada evento · Perfil.

**Convite:** `/convite/:token` (evento + tipo) → ficha (jurado: campos comuns; participante: comuns + tipo de apresentação com campos condicionais) → confirmação.

**Área do jurado:** lista de participantes do evento (filtro por tipo, busca) → detalhe lado a lado: **foto do participante · foto de referência (cosplay) · tipo de apresentação** e demais dados.

**Painel do organizador:** visão geral · locais · eventos (lista/criar/editar em etapas: básico → local e data → ingressos → participantes/convites → revisão e publicar) · vendas e participantes do evento · convites · aprovação de inscrições · financeiro (bruto, taxa do processador, líquido, repasses) · staff e check-in.

**Check-in (PWA):** leitor de QR por câmera, resultado grande e colorido, busca manual, contador.

**Admin:** organizadores · eventos · repasses (a pagar/pagos) · reembolsos e falhas · regras regionais · configurações · relatórios (receita da plataforma).

---

## 10. Design e UX

- **Mobile-first**, rápido, sem fricção. Checkout em no máximo 3 passos.
- Cards de evento com capa grande, data em destaque, local, preço "a partir de R$ X + taxa" e selo de categoria. Tipografia forte, muito respiro, cantos arredondados, sombras leves.
- **Tema claro e escuro** por tokens CSS. Cor de destaque única e consistente. Referência de estética: Luma; de robustez: Sympla.
- **Transparência de preço**: sempre discriminar preço, taxa e garantia. Garantia explicada em uma frase ("cancele até o início do evento e receba tudo de volta").
- Estados vazios amigáveis, skeletons no carregamento, toasts de sucesso/erro, confirmações antes de cancelar.
- Acessibilidade: contraste AA, foco visível, `label` em todo campo, navegação por teclado.
- Formulários de participante com **campos condicionais** por tipo, upload com preview e recorte de foto.

---

## 11. Fases e checklist

> **Lembrete em TODOS os itens:** depois de concluir, rodar build/testes, **commit e push no GitHub** (seção 4.1). Item só conta como concluído com o push feito.

### Fase 0 — Fundação
- [x] 0.1 **Criar as duas pastas `backend/` e `frontend/`** (seção 4.1): backend Go com `GET /healthz`, frontend Vite + React + TS rodando, `.env.example` em cada uma
- [x] 0.2 **Git/GitHub:** `git init` se necessário, configurar `origin` (perguntar a URL ao dono), `.gitignore`, `.gitattributes`, `README.md`, `CLAUDE.md` referenciando este plano, **primeiro commit e push**
- [x] 0.3 Config por env, logging, tratamento de erros padronizado
- [x] 0.4 Docker + `docker-compose` (Postgres local), migrations/seeds sem IDs fixos
- [x] 0.5 GitHub Actions: `go vet`, `go test`, build do frontend
- [x] 0.6 Design tokens, layout base, shadcn/ui, tema claro/escuro
- [x] 0.7 `config_plataforma` com taxas, garantia, dias de repasse e minutos de reserva
- [x] 0.8 **Fechar a fase:** commit, push e `git tag fase-0` + `git push --tags`

### Fase 1 — MVP (não abrir vendas reais antes de concluir 1.7 e 1.8)
- [x] 1.1 Auth (e-mail/senha + Google), verificação de e-mail, recuperação de senha, papéis por evento
- [ ] 1.2 Perfil de organizador (dados de recebimento) e CRUD de locais com mapa (Leaflet)
- [ ] 1.3 CRUD de eventos em etapas, `tipo_acesso`, `modo_participantes`, tipos de ingresso/cadastro, publicação com validações
- [ ] 1.4 Site público: home, busca com filtros, página do evento, página do organizador, meta tags OG
- [ ] 1.5 Convites (jurado e participante especial) + fichas + upload de mídia
- [ ] 1.6 Inscrição aberta de participantes + aprovação/rejeição pelo organizador + área do jurado (leitura)
- [ ] 1.7 Checkout completo: reserva de estoque, cálculo no servidor, Mercado Pago (custódia), webhook idempotente, ledger, QR + e-mail
- [ ] 1.8 Cancelamento com **direito de arrependimento (CDC)** + cancelamento de evento pelo organizador com reembolso integral
- [ ] 1.9 Meus ingressos, **Meus eventos** com selos, regras regionais (semear e bloquear publicação)
- [ ] 1.10 Check-in PWA (QR, busca, contador) e papel `staff`
- [ ] 1.11 Painel básico do organizador (vendas, participantes) e e-mails transacionais
- [ ] 1.12 **Fechar a fase:** commit, push e `git tag fase-1` + `git push --tags`

### Fase 2 — Diferencial e dinheiro
- [ ] 2.1 **Garantia de vaga** (opt-in, cancelamento até o início do evento, estoque devolvido)
- [ ] 2.2 Painel financeiro do organizador (bruto, taxa do processador, líquido) + **repasses** (job, painel admin, marcar como pago, extrato)
- [ ] 2.3 Relatórios admin (receita da plataforma, reembolsos, falhas) e reprocessamento de reembolsos
- [ ] 2.4 Cupons de desconto e lotes com virada automática por data ou quantidade
- [ ] 2.5 Cortesias (sem taxa) e exportação CSV de participantes/compradores
- [ ] 2.6 Transferência de titularidade do ingresso
- [ ] 2.7 **Fechar a fase:** commit, push e `git tag fase-2` + `git push --tags`

### Fase 3 — Concurso completo e escala
- [ ] 3.1 Meia-entrada (cota de 40%, alerta na portaria)
- [ ] 3.2 Aprovação manual para plateia + lista de espera
- [ ] 3.3 **Notas dos jurados**: critérios com pesos, escala configurável, média, ranking, desempate, resultado oculto até o organizador liberar
- [ ] 3.4 Cronograma e ordem de apresentações, upload de áudio
- [ ] 3.5 Equipe do organizador (múltiplos membros) e relatórios avançados
- [ ] 3.6 **Fechar a fase:** commit, push e `git tag fase-3` + `git push --tags`

### Fase 4 — Extras
- [ ] WhatsApp (API oficial), links de divulgadores/afiliados, temas por evento, eventos recorrentes/multi-sessão, assento ou mesa marcada, QR rotativo anti-print, check-in offline, API pública/integrações

---

## 12. Segurança e LGPD

- Senhas com bcrypt/argon2, rate limit em login e cadastro, CORS restrito, validação de entrada em todas as rotas, autorização por papel **e por evento** em toda rota (nunca só no frontend).
- Uploads: validar tipo e tamanho, remover EXIF, redimensionar. Arquivos sensíveis (autorização de responsável, documentos) em bucket **privado** com URL assinada e expiração curta.
- Minimização de dados: jurados nunca recebem contato/documentos de participantes. Fotos de menores só visíveis a jurados e organizador do evento.
- Termos de uso e política de privacidade com aceite registrado. Consentimento para uso de imagem no evento. Exclusão de conta e dados (mantendo o que a lei exigir para registros financeiros).
- `auditoria` em ações sensíveis (reembolsos, repasses, mudanças de status, acesso a dados privados).
- Webhooks com validação de assinatura. Idempotência em qualquer operação que mexa em dinheiro.

---

## 13. Cenários de aceitação críticos (viram testes)

1. Ingresso R$ 30 sem garantia → total R$ 30,99; com garantia → R$ 32,98. Cadastro gratuito → R$ 0,49; cadastro R$ 10 com garantia → R$ 12,48.
2. Duas pessoas tentam comprar o último ingresso ao mesmo tempo: só uma conclui; a outra vê "esgotado".
3. Webhook do MP chega duplicado: pedido processado uma única vez.
4. Comprador com garantia cancela 1 minuto antes do início: recebe preço + taxa + garantia; vaga volta ao estoque; ledger consistente.
5. Comprador sem garantia cancela no dia 3 (evento daqui a 20 dias): reembolso integral (CDC). No dia 10: negado.
6. Comprador sem garantia cancela no dia 3, mas o evento começa em 24h: negado (regra das 48h).
7. Item já com check-in não pode ser cancelado nem reutilizado; QR repetido mostra "já utilizado" com hora.
8. Evento cancelado pelo organizador: todos os pagos reembolsados integralmente; falha de reembolso aparece no painel do admin; nada some em silêncio.
9. Repasse: bruto e líquido batem com o ledger; itens cancelados não entram; taxa do processador é descontada do organizador.
10. Convite: link revogado ou expirado não funciona; mesmo usuário não duplica papel; jurado só vê fichas `aprovado` e nunca campos privados.
11. Participante menor de 18 sem autorização do responsável não consegue enviar a ficha.
12. Publicação de evento pago em UF restrita (ex.: AC) é bloqueada com mensagem clara.
13. Usuário com papéis diferentes em eventos diferentes vê os selos corretos em "Meus eventos".

---

## 14. Decisões assumidas e pendências (o dono deve revisar)

**Assumidas pelo desenho (mudar só se o dono pedir):**
- Garantia e cancelamento são **por item**, não por pedido inteiro.
- Cancelamento com garantia devolve **tudo**, inclusive a taxa fixa de R$ 0,99/0,49.
- Inscrição de **talento** (participante) **não paga taxa da plataforma** no MVP. O organizador pode cobrar um valor de inscrição na Fase 2 (a definir).
- Ingresso gratuito com `tipo_acesso=ingresso` também paga R$ 0,99. Eventos gratuitos devem usar `cadastro` (R$ 0,49).
- Link de convite é reutilizável por tipo, com limite de usos opcional e revogável.
- Alteração de data/local abre janela de 7 dias de cancelamento com reembolso integral.
- Pagamento à vista (Pix e cartão) no MVP; parcelamento fica para depois.
- O jurado só vê participantes `aprovado`.
- **IDs (seção 5):** `bigserial` (inteiro autoincremento) em todas as tabelas, em vez de `uuid`. Consistente com o alerta do item 0.4 sobre `setval()` em sequences após seeds — só se aplica a IDs numéricos com sequence.
- **Migrations:** ferramenta `golang-migrate` (SQL puro em `backend/migrations/`, arquivos `NNNNNN_descricao.up.sql`/`.down.sql`), executada via `go run ./cmd/migrate up|down`. Consultas via GORM.
- **Item 1.1 — `papéis por evento` adiado:** `papeis_evento` referencia `eventos(id)`, que só existe a partir do item 1.3. Implementei apenas o papel global (`usuarios.papel_plataforma`, `usuario`\|`admin_plataforma`); a tabela `papeis_evento` fica para quando `eventos` existir.
- **Item 1.1 — tokens de verificação/reset:** guardados como colunas (`*_token_hash`, `*_expira_em`) direto em `usuarios`, em vez de tabelas separadas — mais simples, um token ativo por vez é suficiente no MVP.
- **Item 1.1 — sessão:** access token JWT (HS256, 15 min, `JWT_SECRET`) + refresh token opaco (32 bytes, hash sha256 salvo em `refresh_tokens`) em cookie httpOnly, `SameSite=Lax`, escopado a `/api/v1/auth`, com rotação a cada refresh.
- **Item 1.1 — Google:** verificação do `id_token` via `google.golang.org/api/idtoken` contra `GOOGLE_CLIENT_ID`. Sem essa env, `POST /auth/google` responde 503 (não quebra o resto do auth) — o dono precisa criar as credenciais OAuth no Google Cloud Console.
- **Item 1.1 — e-mail:** `RESEND_API_KEY` vazio faz o envio só logar no console (dev sem conta Resend) em vez de falhar.
- **Item 1.1 — rate limit:** em memória, por IP, nas rotas `/auth/cadastro`, `/auth/login`, `/auth/google`, `/auth/esqueci-senha`. Não é compartilhado entre instâncias — se o backend escalar horizontalmente, precisa migrar para um store como Redis.

**Pendências para validar fora do código:**
- Contador/advogado: custódia de recursos de terceiros, nome "Garantia de vaga" (vs. "seguro"), retenção ou não da taxa no arrependimento, regras regionais por UF (incluindo Sergipe), emissão de nota fiscal e tributação da taxa/garantia.
- Mercado Pago: valor mínimo de pagamento, devolução da taxa do processador em estornos, prazo de liberação do dinheiro na conta da plataforma.
- Nome do produto e domínio.
- Credenciais reais: `GOOGLE_CLIENT_ID`/secret (Google Cloud Console, para login com Google) e `RESEND_API_KEY` (conta Resend, para e-mails saírem de verdade). Sem elas, o backend funciona normalmente em modo degradado (Google desabilitado, e-mails só logados).
