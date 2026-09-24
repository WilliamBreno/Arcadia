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
- [x] 1.2 Perfil de organizador (dados de recebimento) e CRUD de locais com mapa (Leaflet)
- [x] 1.3 CRUD de eventos em etapas, `tipo_acesso`, `modo_participantes`, tipos de ingresso/cadastro, publicação com validações
- [x] 1.4 Site público: home, busca com filtros, página do evento, página do organizador, meta tags OG
- [x] 1.5 Convites (jurado e participante especial) + fichas + upload de mídia
- [x] 1.6 Inscrição aberta de participantes + aprovação/rejeição pelo organizador + área do jurado (leitura)
- [x] 1.7 Checkout completo: reserva de estoque, cálculo no servidor, Mercado Pago (custódia), webhook idempotente, ledger, QR + e-mail
- [x] 1.8 Cancelamento com **direito de arrependimento (CDC)** + cancelamento de evento pelo organizador com reembolso integral
- [x] 1.9 Meus ingressos, **Meus eventos** com selos, regras regionais (semear e bloquear publicação)
- [x] 1.10 Check-in PWA (QR, busca, contador) e papel `staff`
- [x] 1.11 Painel básico do organizador (vendas, participantes) e e-mails transacionais
- [x] 1.12 **Fechar a fase:** commit, push e `git tag fase-1` + `git push --tags`

### Fase 2 — Diferencial e dinheiro
- [x] 2.1 **Garantia de vaga** (opt-in, cancelamento até o início do evento, estoque devolvido)
- [x] 2.2 Painel financeiro do organizador (bruto, taxa do processador, líquido) + **repasses** (job, painel admin, marcar como pago, extrato)
- [x] 2.3 Relatórios admin (receita da plataforma, reembolsos, falhas) e reprocessamento de reembolsos
- [x] 2.4 Cupons de desconto e lotes com virada automática por data ou quantidade
- [x] 2.5 Cortesias (sem taxa) e exportação CSV de participantes/compradores
- [x] 2.6 Transferência de titularidade do ingresso
- [x] 2.7 **Fechar a fase:** commit, push e `git tag fase-2` + `git push --tags`

### Fase 3 — Concurso completo e escala
- [x] 3.1 Meia-entrada (cota de 40%, alerta na portaria)
- [x] 3.2 Aprovação manual para plateia + lista de espera
- [x] 3.3 **Notas dos jurados**: critérios com pesos, escala configurável, média, ranking, desempate, resultado oculto até o organizador liberar
- [x] 3.4 Cronograma e ordem de apresentações, upload de áudio
- [x] 3.5 Equipe do organizador (múltiplos membros) e relatórios avançados
- [x] 3.6 **Fechar a fase:** commit, push e `git tag fase-3` + `git push --tags`

### Fase 4 — Extras
- [x] 4.1 QR rotativo anti-print
- [x] 4.2 Temas por evento
- [x] 4.3 Links de divulgadores/afiliados (atribuição e estatística)
- [x] 4.4 API pública/integrações (somente leitura)
- [x] 4.5 Eventos com várias sessões (multi-data)
- [x] 4.6 Dias por ingresso (todos / alguns / um por dia) e preço por sessão
- [ ] WhatsApp (API oficial), assento ou mesa marcada, check-in offline

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
- **Item 1.1 — auth no frontend:** access token só em memória (nunca em localStorage), renovado via `POST /auth/refresh` (cookie httpOnly) ao carregar a página. `AuthProvider` em `src/hooks/use-auth.tsx`, rota protegida via `<ProtectedRoute>`.
- **Item 1.2 — um perfil de organizador por usuário** (não múltiplos), com validação simples de CPF/CNPJ (contagem de dígitos, sem dígito verificador — ver seção 14 se precisar de validação forte).
- **Padrão de frontend — `Button` não suporta `asChild`:** o componente `Button` deste projeto usa `@base-ui/react`, não Radix, e não tem a prop `asChild`. Para um link com estilo de botão, usar `buttonVariants({...})` como `className` de um `<Link>`, não `<Button asChild>`.
- **Item 1.3 — "em etapas" simplificado:** em vez de um wizard rígido de 5 passos, a edição do evento é uma página única com seções (básico, local/data, ingressos, publicar) — mais simples de manter e o organizador pode ir e voltar livremente. A etapa "participantes/convites" fica só com o campo `modo_participantes` por enquanto; convites/fichas de verdade entram no item 1.5.
- **Item 1.3 — publicar sem checar `organizador.status == 'ativo'`:** não existe ainda fluxo de aprovação de organizador pelo admin (não está em nenhum item do checklist até agora). A validação de publicação checa `chave_pix` preenchida (quando há ingresso pago), não o status. Quando o painel admin de organizadores existir, reforçar essa checagem.
- **Item 1.3 — regras regionais não checadas na publicação ainda** (tabela `regras_regionais` só existe a partir do item 1.9, que também cuida de semear os dados).
- **Item 1.4 — "atalho" de data (hoje/amanhã/fim de semana) usa o fuso do servidor (UTC)**, não o do evento nem o do visitante. Simplificação aceitável para o MVP; ajustar se virar problema real.
- **Item 1.4 — meta tags OG via Vercel Edge Middleware** (`frontend/middleware.ts`), só reescreve a resposta para user-agents de crawler conhecidos (WhatsApp, Facebook, Twitter/X, Telegram, Discord, Slack, LinkedIn, Pinterest); para humanos a SPA carrega normal. **Não testável localmente** — só roda de verdade num deploy Vercel. Precisa da env `BACKEND_API_URL` configurada no projeto Vercel (diferente de `VITE_API_URL`, que só existe no bundle do cliente).
- **Item 1.5 — `papeis_evento` implementado** (estava adiado desde o item 1.1). Único por (evento, usuario, papel); em modo convite a ficha já nasce `aprovado` ao confirmar (seção 7.11).
- **Item 1.5 — uploads em disco local** (`internal/storage`), não S3. Funciona para dev, **não funciona em host com filesystem efêmero** (Render, por exemplo) — trocar por Supabase Storage/Cloudflare R2 antes de produção (interface `storage.Armazenamento` já isola essa troca). Imagens são decodificadas e recodificadas como JPEG (remove EXIF), WebP não é aceito (sem decoder na stdlib do Go). Sem redimensionamento ainda.
- **Item 1.5 — aceitar convite não usa transação de banco:** cria `papel_evento` e depois a `ficha_participacao` em passos separados. Se o segundo passo falhar, o primeiro fica persistido — mas o fluxo é auto-recuperável: uma nova tentativa encontra o papel já criado (não duplica, não reincrementa `usos` do convite) e só tenta de novo a parte que faltou. Encontrado e verificado via teste manual (bug real de `dados` JSONB nulo, corrigido).
- **Item 1.5 — validação de "menor de 18" simplificada:** checa só se `responsavel_nome`, `responsavel_contato` e `autorizacao_responsavel_url` estão preenchidos, sem validar o conteúdo do arquivo de autorização.
- **Item 1.5 — campos condicionais por `tipo_apresentacao`:** implementados só os principais de cada tipo (não os 100% do detalhamento da seção 5.1) para não alongar demais o formulário; o campo `dados` é um JSON livre, então dá pra completar depois sem migration.
- **Item 1.6 — bug real corrigido: jurado via a própria ficha na lista de participantes.** `ListarParaJurado`/`ObterParaJurado` filtravam só por status, não por papel — um jurado aparecia na própria lista de "participantes" dele. Corrigido com `ListarPorEventoEPapel` (papel=participante sempre). Encontrado em teste manual ponta a ponta.
- **Item 1.6 — "vagas por tipo/categoria" não é checada na inscrição.** A seção 2.4/7.12 menciona limite de vagas por tipo/categoria de participante, mas não há um campo de configuração pra isso ainda (só `capacidade_total`, que é do evento como um todo). `Inscrever()` só checa `modo_participantes` e o prazo (`inscricao_talentos_inicio/fim`). Se isso virar necessário, precisa de um novo campo (ex.: `vagas_por_tipo` no evento ou nos tipos de apresentação).
- **Item 1.6 — organizador pode editar evento já publicado** (`EventoService.Atualizar` não bloqueia por status). Não é o foco do item, mas é um comportamento existente desde o 1.3 — sem trava alguma, incluindo `tipo_acesso`. Vale revisar quando o checkout (1.7) estiver valendo, pra não deixar mudar `tipo_acesso` de um evento com vendas em andamento.
- **Item 1.7 — ⚠️ testado com `MERCADOPAGO_ACCESS_TOKEN` de PRODUÇÃO** (`APP_USR-...`) do projeto "drenux", a pedido explícito do dono, depois de confirmar que era intencional (token de teste `TEST-...` seria o normal). Só testei o que **não move dinheiro real**: reserva de estoque (com trava de concorrência via `SELECT...FOR UPDATE`), cálculo de preço (bate com os 4 cenários da seção 13), criação de *preference* real no Mercado Pago (retornou `checkout_url` de verdade), webhook com pagamento inexistente (404 tratado sem quebrar), job de expiração de reserva (libera estoque de verdade, testado forçando `expira_em` no passado). **Não disparei nenhum pagamento real** — a aprovação de fato (webhook com `status=approved`) fica para o dono testar manualmente com um Pix real quando quiser.
- **Item 1.7 — `auto_return` removido da criação da *preference*.** O Mercado Pago rejeita `auto_return=approved` quando `back_urls.success` não é uma URL pública (erro real encontrado testando: "back_url.success must be defined" mesmo com a URL preenchida — é `localhost` que ele não aceita). Sem isso, o comprador só precisa clicar em "voltar ao site" após pagar em vez de ser redirecionado automático; o webhook (que de fato confirma o pagamento) não depende disso.
- **Item 1.7 — garantia de vaga NÃO está no checkout ainda** (é o item 2.1, Fase 2). O schema (`garantia_contratada`, `garantia_centavos` em `itens_pedido`) já existe porque é o modelo de dados da seção 5, mas o checkout sempre cria os itens com `garantia_contratada=false`. `calcularTotalItem()` já aceita o parâmetro — o item 2.1 só precisa expor a opção no checkout e mudar esse `false` por um valor vindo do request.
- **Item 1.7 — QR gerado como token HMAC, sem imagem.** `codigo` (8 chars, sem O/0/I/1) + `qr_token` (HMAC-SHA256 assinado com `JWT_SECRET`) ficam prontos no banco; renderizar o QR como imagem visual fica para o frontend (item 1.9, "Meus ingressos") usando uma lib client-side — evita adicionar dependência de geração de imagem no Go.
- **Item 1.7 — e-mail de confirmação é texto simples** (lista os códigos + link para "Meus ingressos"), sem embutir a imagem do QR — consistente com a decisão acima.
- **Item 1.7 — cada unidade comprada é seu próprio item_pedido** (não agrega quantidade num só registro), e todos os itens de um pedido nascem com o nome/e-mail do comprador como titular — trocar o titular de itens além do primeiro é uma tela futura (não estava no escopo do 1.7).
- **Item 1.7 — webhook sempre responde 200**, mesmo quando `ProcessarWebhook` retorna erro (loga e segue) — evita loop de reenvio do MP por erro nosso; se for transitório (banco fora do ar), o reenvio automático do MP ainda ajuda porque o pagamento não ficou registrado.
- **Item 1.7 — `.env` local:** `EMAIL_REMETENTE` precisou virar `"Evve <no-reply@evve.local>"` (com aspas) porque um `source .env` direto no bash quebra sem isso (os `<>` viram redirecionamento) — corrigido no `.env` e no `.env.example`.
- **Item 1.8 — bug real corrigido: janela das 48h era exclusiva, o plano pede inclusiva.** O texto da seção 7.4 é literal: "agora ≤ inicio_em − 48h". Minha primeira implementação usava `<` estrito (excluía o instante exato de 48h antes do evento); o teste unitário `TestDentroDoPrazoCDC` pegou a divergência e foi corrigido para `≤`.
- **Item 1.8 — reembolso nunca muda o status do item antes de confirmar no Mercado Pago.** `executarReembolso` só marca `cancelado` e grava lançamentos (`reembolso_preco`/`reembolso_taxa`/`reembolso_garantia`) depois do MP confirmar o estorno; se o MP falhar, fica registrado em `reembolsos` com `status=falhou` e nada mais muda — verificado manualmente forçando uma falha real (pagamento inexistente): o item continuou "pago" e nenhum lançamento foi criado.
- **Item 1.8 — cancelamento de evento não trava em falha individual.** Se o reembolso de um item falhar, o evento cancela mesmo assim e a falha entra na lista `falhas` da resposta — não existe ainda o "painel de falhas" citado no plano (fica pendente para quando houver painel admin, Fase 2/3), por ora as falhas só aparecem na resposta da chamada e ficam registradas em `reembolsos` para auditoria manual.
- **Item 1.8 — "data da compra" usa `pagamentos.criado_em`** (quando o pagamento foi aprovado), não `itens_pedido.criado_em` (quando a reserva começou) — mais fiel ao "7 dias da compra" do CDC, já que a reserva pode ser feita minutos antes do pagamento cair.
- **Item 1.9 — selo "organizador" vem de `organizadores.usuario_id`, não de `papeis_evento`.** O modelo da seção 5 sugere `papeis_evento.papel` incluir `'organizador'`, mas a coluna `origem` da tabela só aceita `'convite'` ou `'inscricao'` (nenhum dos dois descreve como alguém vira organizador) — mais simples derivar o selo "Organizador" direto de `organizadores.usuario_id = eventos.organizador_id` (que já é a fonte de verdade usada em todo o resto do código pra dono de evento) do que forçar isso dentro de `papeis_evento`.
- **Item 1.9 — regras regionais tratadas como bloqueio sempre** (não só nas UFs com `permite_taxa=false`, também nas com `exige_canal_sem_taxa=true` e no limite percentual excedido). O texto do plano usa "bloquear" pro caso geral e "alertar" pro limite percentual — simplifiquei pra um único comportamento (bloqueio com mensagem clara) em vez de dois níveis de severidade, e porque ainda não existe painel admin pra liberar manualmente (também pendente, mencionado na seção 7.8).
- **Item 1.9 — checagem regional só considera a UF/cidade do local do evento**, não verifica de fato se o organizador tem um "canal de venda sem taxa" alternativo (`exige_canal_sem_taxa`) — isso não é algo que dê pra confirmar programaticamente; o bloqueio só avisa que a UF exige isso, cabe ao organizador confirmar por fora.
- **Item 1.9 — "Meus ingressos" mostra pago + utilizado**, não itens reservados (ainda não pagos, aparecem só na página do pedido) nem cancelados/expirados (sem função pro comprador depois de resolvidos).
- **Item 1.9 — QR renderizado no cliente** (`qrcode.react`) a partir de `codigo:qr_token` — o backend nunca gera imagem, só os dois valores que a leitora do check-in (item 1.10) vai validar.
- **Item 1.10 — origem `'manual'` adicionada ao `CHECK` de `papeis_evento.origem`.** Staff é atribuído direto pelo organizador (por e-mail de alguém que já tem conta), sem o fluxo público de link/token que `'convite'` e `'inscricao'` descrevem — nova migração (`000018`) estende o constraint para `('convite', 'inscricao', 'manual')`.
- **Item 1.10 — tabela `checkins` da seção 5 não foi criada.** O plano lista uma tabela separada (`id, item_id, staff_usuario_id, dispositivo, resultado, criado_em`) só para auditoria de cada leitura; implementei a validação como `UPDATE itens_pedido SET status='utilizado' WHERE id=? AND status='pago'` atômico direto (a fonte de verdade já exigida pela seção 7.10), sem log de cada tentativa de leitura. Auditoria fina por leitura (inclusive tentativas inválidas) fica pendente — não bloqueia o check-in funcionar, mas se o organizador precisar depois de "quem leu esse QR e quando" isso não existe ainda.
- **Item 1.10 — busca manual (`GET /checkin/eventos/:id/busca`) é só consulta, não faz check-in.** Ela devolve nome/e-mail/código/status pra portaria localizar alguém visualmente, mas nunca devolve `qr_token` (ficaria exposto pra qualquer staff) — então não dá pra "confirmar entrada" batendo só na busca; o check-in de verdade só acontece via `POST /checkin/validar` com o par `codigo`+`qr_token` que vem do QR. Isso bate com o texto da seção 7.10 ("busca manual… contador"), que não descreve a busca como um gatilho de entrada.
- **Item 1.10 — acesso ao check-in é "dono do evento OU staff confirmado nesse evento"** (`CheckinService.TemAcesso`), checado em toda chamada (`validar`, `busca`, `resumo`) — jurado, participante ou qualquer outro papel sem ser `staff`/organizador toma 403.
- **Item 1.10 — leitor de QR no frontend usa `html5-qrcode`** (câmera via `navigator.mediaDevices`, decodifica pra `codigo:qr_token` no formato que `qrcode.react` já gera desde o item 1.9). Rota `/checkin/:eventoId`, acessível a qualquer usuário autenticado — o backend é quem decide se a pessoa tem acesso (mesmo padrão já usado em `/e/:slug/jurado`). Testado o build e o fluxo via curl; a leitura de câmera de verdade (permissão do navegador, foco, iluminação) precisa ser validada num celular real pela portaria antes do evento.
- **Item 1.11 — "vendas" é receita bruta simples (soma de `preco_centavos`), sem taxa do processador nem repasse.** O painel financeiro completo (bruto/taxa/líquido/repasses) é o item 2.2 da Fase 2 — aqui é só "quanto vendeu e pra quem", contagem e receita por tipo de ingresso + lista de compradores (pago/utilizado). Não expõe `qr_token` na lista (mesmo cuidado do item 1.10: só o dono/staff do evento vê, mas não precisa do token pra nada aqui).
- **Item 1.11 — "participantes" do painel básico reaproveita o que já existia desde o item 1.6** (`ParticipantesCard`, aprovação/rejeição de fichas) — não foi preciso criar nada novo pra essa parte, só os e-mails transacionais que faltavam.
- **Item 1.11 — e-mails transacionais que faltavam: aprovação/rejeição de ficha de participante/jurado** (seção 7.12 pede "e-mail ao participante em cada mudança de status", mas isso nunca tinha sido implementado desde o item 1.5). `FichaService` ganhou `usuarios *repository.UsuarioRepository` e `mailCliente *mail.Cliente`; `Aprovar`/`Rejeitar` disparam e-mail best-effort (erro de envio não desfaz a aprovação/rejeição, mesmo padrão dos outros e-mails do sistema). Os demais e-mails transacionais (verificação de cadastro, redefinição de senha, ingresso confirmado, cancelamento confirmado) já existiam desde os itens 1.1/1.7/1.8.
- **Item 2.1 — garantia é escolhida por requisição de tipo de ingresso (checkbox único na tela do evento aplica a todos os itens do pedido).** O plano diz "por item", e o backend guarda `garantia_contratada` por item, mas a UI não permite misturar garantia/sem garantia dentro do mesmo pedido; quem quiser isso faz dois pedidos. `ItemRequisitado.GarantiaContratada` já aceita valores distintos por tipo de ingresso na API.
- **Item 2.1 — valor da garantia e da taxa vêm de `config_plataforma` e são expostos em `GET /eventos/:slug`** (`garantia_centavos`, `taxa_plataforma_centavos`) só para exibição; o total cobrado é sempre recalculado no servidor. Pedir garantia em evento com `garantia_habilitada=false` retorna 422.
- **Item 2.1 — a parte de cancelamento/estoque já estava pronta desde o 1.8** (`avaliar()` trata garantia, reembolso total e ledger; item `cancelado` sai da contagem de estoque). Verificado: R$ 30,00 + 0,99 + 1,99 = R$ 32,98, lançamentos `venda_preco`/`taxa_plataforma`/`garantia` gravados na aprovação e simulação de cancelamento devolvendo R$ 32,98. O estorno real no Mercado Pago não foi executado (sem pagamento real).

- **Item 2.2 — repasse gerado pelo job `POST /jobs/gerar-repasses` já nasce `pendente`** (o job só cria quando `liberar_em` já passou; o status `calculado` existe no schema mas não é usado). Um repasse por evento (índice único parcial); evento com bruto zero ou cancelado não gera repasse. Ao gerar: lançamento `taxa_processador` (−); ao marcar pago: lançamento `repasse` (−) na mesma transação, e a segunda chamada devolve 409.
- **Item 2.2 — "dias úteis" = segunda a sexta, sem feriados**; "fim do último dia" = 23:59:59 do dia de `fim_em` (ou `inicio_em` se não houver fim), no fuso do servidor.
- **Item 2.2 — taxa do processador alocada por divisão inteira** (`taxa × total_item ÷ valor_pago`); sobra de centavos fica com o organizador. Só itens `pago|utilizado` entram; itens cancelados não geram lançamento `custo_processador_perdido` ainda — depende de confirmar se o MP devolve a taxa em estornos (tarefa de verificação da seção 7.7).
- **Item 2.2 — painel admin mínimo:** `/admin/repasses` (lista + marcar pago) protegido por `papel_plataforma=admin_plataforma`; não existe tela/fluxo para promover um usuário a admin (feito por SQL). O organizador vê `GET /org/eventos/:id/financeiro` (ao vivo até o repasse existir) e o extrato em `/organizador/repasses`. Chargeback pós-repasse continua tratamento manual (sem código).

- **Item 2.3 — reprocessar reembolso reusa `executarReembolso` com a mesma linha de `reembolsos`** (não cria outra): só reembolsos `falhou` cujo item ainda está `pago`; se o MP falhar de novo, tudo continua como estava (verificado com pagamento inexistente → 502, reembolso segue `falhou`, item segue `pago`). Vale para o botão do admin (`POST /admin/reembolsos/:id/reprocessar`) e para o job `POST /jobs/reprocessar-reembolsos` (sem limite de tentativas — pendência).
- **Item 2.3 — nova checagem em todo reembolso:** soma dos reembolsos concluídos + valor novo nunca excede o valor do pagamento (invariante da seção 7.4, antes só documentada).
- **Item 2.3 — relatório admin** (`GET /admin/relatorios`, tela `/admin/relatorios`): receita = taxa + garantia dos itens `pago|utilizado` de eventos já iniciados, resumo de reembolsos por tipo/status e lista de falhas. Reembolsos de dev antigos com status `falhou` aparecem nele (vieram dos testes do item 1.8).

- **Item 2.4 — lotes = tipos de ingresso com o mesmo `lote_grupo`** (coluna adiantada da Fase 3), em sequência por `ordem`, `id`. Só o primeiro lote disponível (ativo + dentro de `vendas_inicio/fim` + com estoque) aparece na página pública e pode ser comprado; virada por data e por quantidade saem da mesma regra (`LoteAtual`, testada). Comprar um lote que não é o atual retorna 422. Virada é calculada na hora, sem job.
- **Item 2.4 — cupom:** um código por pedido, vale para todos os itens, desconta só do **preço** (taxa e garantia intactas), `percentual` (1–100) ou `valor` (centavos, limitado ao preço). O desconto é custo do organizador: `itens_pedido.preco_centavos` já guarda o preço com desconto (então ledger, repasse e reembolso usam o valor efetivamente pago) e `desconto_centavos` guarda o quanto foi abatido. `max_usos` conta itens, com a linha do cupom travada (`FOR UPDATE`) na reserva; reserva expirada devolve os usos; item cancelado depois de pago **não** devolve uso (decisão simples, revisar se o organizador pedir).
- **Item 2.4 — sem cupom por tipo de ingresso** (vale para o pedido inteiro) e sem edição de cupom (só criar/desativar).

- **Item 2.5 — cortesia = item `pago` com `cortesia=true`, total 0, sem taxa e sem pagamento**, dentro de um pedido `pago` que pertence ao usuário do organizador (por causa do `usuario_id` obrigatório). Ocupa estoque do tipo (checado com lock), exige evento publicado, 1–50 por emissão. Fica fora de "Meus ingressos"/selo "Ingresso" do organizador e não entra no repasse (não tem pagamento); continua aparecendo no check-in e no CSV. Revogar (`DELETE`) só se ainda `pago` e sem reembolso (não há dinheiro), devolvendo a vaga. Em "Vendas" (item 1.11) a cortesia entra na contagem de vendidos com preço 0 — ajuste se incomodar.
- **Item 2.5 — quem recebe cortesia pode não ter conta:** o e-mail leva a `/ingresso/:codigo/:token`, página pública que mostra o QR e exige o `qr_token` na URL (comparado em tempo constante). Quem tiver o link tem o ingresso, como no ingresso por e-mail de qualquer plataforma.
- **Item 2.5 — CSV** (`GET /org/eventos/:id/exportar.csv?tipo=compradores|participantes`, separador `;`, UTF-8 com BOM, células que começam com `= + - @` ganham apóstrofo contra formula injection). O CSV de participantes **não inclui telefone, data de nascimento nem dados do responsável de menores** (dados pessoais; o organizador continua vendo na tela de aprovação) — se o organizador precisar deles no arquivo, decidir com o dono antes.

- **Item 2.6 — transferir troca o titular nominal, não o dono do pedido.** Quem comprou continua com o ingresso em "Meus ingressos", o direito de cancelar/reembolso e o pedido; muda só quem entra no evento. A transferência **regenera `codigo` e `qr_token`** (o QR antigo vira `nao_encontrado` no check-in — verificado), envia e-mail com o link do novo QR (`/ingresso/:codigo/:token`) ao novo titular e avisa o titular anterior. `UPDATE` condicional (`status='pago'` e código antigo) evita corrida com check-in/cancelamento. Só item `pago`, de pedido próprio (cortesia não transfere), evento não cancelado e ainda não iniciado. Sem limite de transferências e sem taxa (não definidos no plano — decidir se aparecer revenda abusiva). Cada troca grava em `transferencias_ingresso` (auditoria: quem fez, de/para, código anterior). Dados do novo titular (nome, e-mail) são informados por quem transfere — sem confirmação de aceite pelo destinatário.

- **Item 3.1 — meia-entrada = flag `meia_entrada` num tipo de ingresso.** Cota: soma das quantidades dos tipos **ativos** marcados como meia ≤ 40% da soma de todos os tipos ativos do evento (validado ao criar/editar tipo — 422 — e como problema na publicação). Base é a lotação declarada nos tipos, não `capacidade_total`. Excluir/desativar outros tipos depois de publicado pode reabrir o estouro (só a publicação revalida). Idosos (sem cota) não têm marca própria: o organizador cria um tipo comum. Alerta na portaria: check-in válido de tipo meia mostra "MEIA-ENTRADA — conferir documento" e a busca manual marca o item. A taxa fixa não muda; nenhuma comprovação é armazenada (conferência é presencial).

- **Item 3.2 — `eventos.aprovacao_manual` + tabela `solicitacoes_plateia`** (pendente → aprovada | rejeitada | lista_espera). Com o flag ligado, `Reservar` exige solicitação `aprovada` (422 caso contrário); sem o flag nada muda. A pessoa solicita pela página do evento; o organizador aprova/rejeita/manda para a fila. **Lista de espera = "aprovado pelo organizador, mas sem vaga agora"**: aprovar quando não há vaga livre (vagas livres dos tipos ativos − aprovados que ainda não compraram) vira `lista_espera` automaticamente. Quando uma vaga volta (reserva expirada pelo job, cancelamento pelo comprador, reprocesso de reembolso, cortesia revogada, ou rejeição/rebaixamento de alguém já aprovado), a fila é promovida em ordem de decisão e cada promovido recebe e-mail. Verificado: bruno aprovado segura a vaga mesmo com a reserva expirada (jurada continua na fila) e só quando o bruno é rejeitado a jurada é promovida.
- **Item 3.2 — escopo:** a lista de espera só existe em evento com aprovação manual (não há fila de "esgotado" em evento aberto); a promoção não reserva estoque (só libera o direito de comprar); como só aprovados compram e aprovar respeita as vagas livres, aprovados nunca passam das vagas. Cancelar evento não dispara promoção.
- **Item 3.2 — bug antigo corrigido junto:** o formulário do organizador não enviava `garantia_habilitada`, então todo "Salvar" resetava para `true`. Agora `garantia_habilitada` e `aprovacao_manual` são opcionais (`nil` = manter na edição; padrão true/false na criação).

- **Item 3.3 — critérios** (`criterios_avaliacao`: peso, `nota_min`/`nota_max`/`passo`, `tipo_apresentacao` opcional) e **avaliações** (`avaliacoes`, uma linha por ficha × jurado × critério, com `comentario` privado). Critério que já tem nota lançada não pode ser editado nem excluído (409). O jurado só avalia participantes **aprovados**, só com os critérios aplicáveis ao tipo da ficha, nota validada contra a escala (limites + passo, 422). Notas ficam em **rascunho** até "Finalizar" (exige todos os critérios aplicáveis; depois trava — 409). Jurado só lê as próprias notas.
- **Item 3.3 — cálculo (`CalcularRanking`, testado):** só avaliações finalizadas; cada nota é **normalizada para 0–10** (para escalas diferentes conviverem) e o jurado dá a média ponderada por peso; nota final = média dos jurados que finalizaram; ranking **separado por categoria** (`tipo_apresentacao`). **Desempate:** maior média no critério de maior peso, depois o seguinte (peso desc, ordem, id); persistindo o empate, dividem a posição. Sem ajuste de outliers/descarte de maior e menor nota (não pedido).
- **Item 3.3 — resultado oculto:** o organizador vê o ranking parcial a qualquer momento; o público só vê depois de `POST /org/eventos/:id/resultado {liberar:true}` (`resultado_liberado_em`; dá para ocultar de novo), com **nome protegido** — nome artístico, ou primeiro nome + inicial do sobrenome, e **menor de idade só nome artístico/primeiro nome** — mais nota e posição (sem jurados, sem comentários). Se o dono quiser nome completo publicado, decidir antes (dado pessoal de menores). Detalhes por critério não são divulgados a participantes.

- **Item 3.4 — cronograma** (`cronograma_itens`: título, descrição, local/palco, início, fim opcional) é a programação **pública** do evento (aparece na página do evento; CRUD do organizador). **Ordem de apresentação** é uma lista única 1..n dos participantes aprovados (`PUT /org/eventos/:id/ordem-apresentacao`, transação; quem fica fora da lista perde a ordem); é visível para jurados (que já recebiam a lista por `ordem_apresentacao`) e organizador — **não é publicada** (nomes de participantes/menores) e não gera horário por apresentação (só ordem).
- **Item 3.4 — áudio:** o upload de áudio (MP3/WAV/M4A, 10 MB) já existia desde o 1.5; agora o formulário do participante (convite e inscrição aberta) tem o campo, o arquivo fica em `fichas.dados.audio` (não usei a tabela `midias_ficha` do modelo) e o jurado ouve no player. **Correção de segurança:** o backend confiava no `Content-Type` declarado pelo cliente; agora confere a assinatura do arquivo (ID3/frame MPEG, RIFF…WAVE, ftyp) antes de gravar. Os arquivos são servidos publicamente em `/uploads/<nome aleatório>` (URL não listável, mas sem autenticação — mesma decisão dos demais uploads).

- **Item 3.5 — equipe (`organizador_membros`, papel único `gestor`).** O dono adiciona por e-mail alguém que já tem conta. `ObterOrganizadorAtual` agora resolve: header `X-Organizador-ID` (dono ou membro, senão 403) → perfil próprio → primeira equipe da qual participa. Assim **todas** as rotas `/org/*` já valem para membros (eventos, ingressos, cupons, cortesias, participantes, convites, staff, check-in, cronograma, critérios...). **Só o dono** (`ObterOrganizadorDono`) acessa: perfil/Pix, financeiro do evento, repasses, relatórios (têm receita), cancelar evento (dispara reembolso em massa) e a própria equipe — verificado (403). Limitações: um papel só (sem "somente leitura"/"financeiro"), sem seletor de organizador na interface (o frontend não envia o header; quem é dono e membro ao mesmo tempo age como dono), e o membro ainda vê o card de Vendas com receita do evento (item 1.11) — restringir se o dono quiser.
- **Item 3.5 — relatórios avançados do organizador** (`GET /org/relatorios?de&ate`, tela `/organizador/relatorios`, só dono): totais, por evento (vendidos sem cortesia, cortesias, receita bruta = preço, check-ins e % de comparecimento, cancelados) e vendas por dia (por data de criação do item, sem cortesias). Receita é o preço do ingresso, sem taxas/repasse (isso segue no painel financeiro do 2.2).

- **Item 4.1 — QR rotativo (`eventos.qr_rotativo`, opt-in do organizador).** O QR passa a ser `codigo:R<janela>.<HMAC>` com janela de 30s (`unix/30`), assinado com o mesmo segredo do QR estático; o servidor aceita a janela atual e as vizinhas (±30s de relógio) e devolve `qr_expirado` (laranja no leitor: "QR desatualizado") se vier token antigo **ou o estático** — em evento rotativo, print/foto do ingresso não entra (verificado, incl. reuso: `ja_utilizado`). O portador obtém o payload em `GET /me/ingressos/:id/qr` (só o dono do pedido) ou `GET /ingressos/:codigo/:token/qr` (link do e-mail), com `Cache-Control: no-store`, e o frontend renova sozinho (`useQR`). Consequência: com o flag ligado o participante **precisa de internet no celular** para abrir o ingresso na portaria. Eventos sem o flag continuam com o QR estático (o leitor aceita os dois formatos). O check-in offline (também da Fase 4) é incompatível por construção com este modo e **não foi feito**: exigiria service worker e distribuir o segredo de verificação aos leitores.

- **Item 4.2 — tema por evento = uma cor (`eventos.cor_tema`, `#RRGGBB`).** Vira as variáveis `--primary`/`--ring` do shadcn só dentro da página do evento (texto do botão escolhido pela luminância, preto ou branco). O backend só aceita o formato hexadecimal — valor com CSS embutido é ignorado e mantém o anterior (verificado); vazio volta ao padrão. Sem logo/fonte/banner próprios além do que já existia (capa).
- **Item 4.3 — afiliados (`afiliados`, `pedidos.afiliado_id`).** O organizador cria um divulgador e recebe o link `/e/:slug?ref=<código>`; o front guarda o `ref` na sessão e o manda na criação do pedido; o servidor atribui se o código existir e estiver ativo naquele evento (senão ignora, sem bloquear a compra). O painel mostra pedidos pagos, ingressos e receita (preço, sem cortesias) por divulgador. **Sem comissão:** calcular/pagar comissão mexe em dinheiro (quem paga, sobre preço ou taxa, o que fazer em reembolso) e depende de decisão do dono — por isso é só atribuição. Atribuição é por último clique na sessão do navegador; sem contagem de cliques.

- **Item 4.4 — API pública somente leitura** (`/api/public/v1/*`, sem JWT). Chaves (`api_keys`) criadas só pelo **dono** (`/org/api-keys`; membro da equipe recebe 403): formato `evve_<48 hex>`, exibida **uma vez**, guardada apenas como SHA-256, com prefixo para identificação e `ultimo_uso_em`; revogação imediata (401 depois — verificado). Escopo = eventos do dono da chave (evento alheio → 404). Endpoints: eventos, evento, ingressos (tipos + ocupados/disponíveis), participantes (portadores pagos/utilizados: nome, e-mail, código, status — **dados pessoais**, entregues ao próprio organizador; nunca inclui `qr_token`) e resumo de check-in. Rate limit por IP (5 req/s, burst 20, em memória). Sem escrita, sem webhooks de saída, sem escopos por chave e sem paginação (adicionar se algum evento tiver milhares de ingressos). A chave é para servidores; o CORS do backend só libera o frontend.

- **Renomeação Arcadia → Evve.** Trocado o nome de marca em tudo que o usuário vê: `NOME_PLATAFORMA` (padrão e `.env.example`), remetente de e-mail, título/rodapé/cabeçalho/home do frontend, textos, README e CLAUDE.md, chave do tema no `localStorage` (`evve-tema` — quem tinha tema salvo volta ao padrão uma vez) e prefixo das chaves de API (`evve_`; chaves antigas `arc_` deixam de valer — só existiam de teste). **Mantidos de propósito:** caminho do módulo Go e imports (`github.com/WilliamBreno/Arcadia/...`), URL do repositório GitHub e usuário/senha/nome do banco Docker (`arcadia`) — mudar isso exige renomear o repositório e recriar o banco, sem ganho visível. Para completar: renomear o repo no GitHub (Settings → Rename) e, se quiser, trocar o módulo com `go mod edit -module` + ajuste dos imports.

- **Item 4.5 — eventos com várias sessões (`sessoes`, `tipos_ingresso.sessao_id`, `checkins_sessao`).** Decisões do dono: (1) **estoque único** — continua contado por tipo de ingresso, sem estoque por sessão; (2) **ingresso vale para o evento todo** (uma entrada por sessão), salvo tipo **restrito a uma sessão** (`sessao_id`; só aceita sessão ativa do mesmo evento); (3) **repasse uma vez, depois da última sessão** — `evento.inicio_em/fim_em` passam a ser derivados das sessões ativas (primeira e última), então o job de repasse (fim + dias úteis) e as regras de cancelamento/garantia/CDC (contadas a partir do início) usam esses limites automaticamente; (4) **cancelar sessão reembolsa só os ingressos restritos àquela sessão** (integral: preço + taxa + garantia, mesmo caminho do cancelamento de evento, com falhas indo para o painel admin); ingresso do evento todo **continua válido** e o titular recebe e-mail; cancelar o **evento** inteiro segue reembolsando tudo. Cancelar sessão é **só do dono** (mexe em dinheiro) e desativa os tipos restritos a ela.
- **Item 4.5 — check-in por sessão:** com sessões, o leitor descobre a sessão em andamento (janela: 3h antes do início até o fim; sem fim, 12h após o início; sessão cancelada nunca conta) e registra a entrada em `checkins_sessao` com UNIQUE (item, sessão) — a 2ª leitura na mesma sessão dá `ja_utilizado`, na sessão seguinte dá `valido` de novo (verificado). Fora da janela de qualquer sessão, ou ingresso restrito a outra sessão: `fora_da_sessao`. O contador do check-in passa a ser o da sessão em andamento (válidos para ela × já entraram). O item vira `utilizado` na primeira entrada (então não dá mais para o comprador cancelar depois disso, como antes). Eventos sem sessões seguem exatamente como antes.
- **Item 4.5 — limitações:** (a parte de "dias parciais" e "preço por sessão" foi resolvida no item 4.6 abaixo); sessões sobrepostas escolhem a que começa mais perto de agora; a lista de espera/aprovação manual e o QR rotativo funcionam no nível do evento; o cronograma (3.4) continua separado das sessões. Cancelar a última sessão ativa deixa o período do evento como estava (cancele o evento se for o caso).

- **Item 4.6 — dias por ingresso e preço por sessão (substitui o `sessao_id` único do 4.5 por `tipo_ingresso_sessoes`).** Cada tipo de ingresso/cadastro escolhe: **todos os dias** (nenhuma sessão listada: um ingresso, uma entrada por dia), **só alguns dias** (lista de sessões, ex.: sábado+domingo) ou **um ingresso por dia** (`POST /org/eventos/:id/ingressos/por-sessao`: cria um tipo por sessão, cada um com preço e quantidade próprios — preço por sessão é opcional, o organizador decide). Os dias do evento são as sessões: criação unitária ou **em lote** (período + dias da semana + horário, até 100, validadas antes de gravar). Estoque continua por tipo (no "por dia" cada dia tem o seu). Migração: o `sessao_id` antigo virou uma linha na tabela nova.
- **Item 4.6 — cancelar uma sessão (regra do 4.5 generalizada):** reembolsa integralmente **só** os ingressos pagos e ainda não usados cujos dias listados **não têm mais nenhuma sessão ativa** (ex.: "sábado+domingo" só é reembolsado quando os dois forem cancelados — verificado); ingresso do evento todo, ou com outros dias ainda ativos, continua válido e o titular é avisado por e-mail; tipos sem sessão ativa deixam de ser vendidos. Ingresso já utilizado não é reembolsado. Check-in: o ingresso só entra nos dias listados (`fora_da_sessao` nos demais) e uma vez por dia.
- **Item 4.6 — limitações:** ao editar um tipo, o conjunto de dias é substituído (sem histórico); "um por dia" cria tipos independentes (não há vínculo entre eles nem desconto de pacote — o "passe geral" é um tipo à parte com o preço que o organizador quiser); trocar os dias de um tipo que já vendeu não reembolsa nem avisa ninguém (pendência).

**Pendências para validar fora do código:**
- Contador/advogado: custódia de recursos de terceiros, nome "Garantia de vaga" (vs. "seguro"), retenção ou não da taxa no arrependimento, regras regionais por UF (incluindo Sergipe), emissão de nota fiscal e tributação da taxa/garantia.
- Mercado Pago: valor mínimo de pagamento, devolução da taxa do processador em estornos, prazo de liberação do dinheiro na conta da plataforma.
- Nome do produto e domínio.
- Credenciais reais: `GOOGLE_CLIENT_ID`/secret (Google Cloud Console, para login com Google) e `RESEND_API_KEY` (conta Resend, para e-mails saírem de verdade). Sem elas, o backend funciona normalmente em modo degradado (Google desabilitado, e-mails só logados).
- **🔴 Urgente: trocar `MERCADOPAGO_ACCESS_TOKEN`.** O `backend/.env` local está com o token de **produção** do projeto "drenux" (não commitado, mas apareceu em texto puro no chat desta sessão — vale rotacionar esse token no painel do drenux por segurança). Antes de qualquer teste de pagamento real da Evve, trocar por um token de **teste** (`TEST-...`) dedicado à Arcadia, criado no painel do Mercado Pago Developers.
