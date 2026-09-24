# Deploy do Evve — passo a passo

Arquitetura: **Vercel** (frontend + proxy `/api`) → **Railway** (API Go em Docker + Postgres) · **Cloudflare R2** (uploads) · **Resend** (e-mail) · **Mercado Pago** (pagamentos) · **GitHub Actions** (jobs periódicos).

> Os nomes de botões e menus mudam com o tempo nesses serviços. Se algo não estiver onde o guia diz, procure pelo termo em destaque.

Guarde os valores que forem sendo gerados numa nota **privada** (gerenciador de senhas). Nunca no repositório.

Ordem: 1 R2 → 2 Resend → 3 Railway → 4 Mercado Pago → 5 Vercel → 6 GitHub → 7 Admin → 8 Testes.

---

## 1. Cloudflare R2 (imagens e áudios)

1. Crie conta em cloudflare.com → menu **R2 Object Storage** (pede cartão, mas o uso pequeno cabe no plano gratuito).
2. **Create bucket** → nome `evve-uploads`.
3. Abra o bucket → **Settings** → **Public access**: ative o domínio `r2.dev` (mais rápido) ou conecte um **Custom Domain** (melhor, ex.: `cdn.seudominio.com`). Copie a URL pública → será `UPLOADS_BASE_URL`.
4. Volte para **R2 Object Storage** → **Manage API Tokens** → **Create API token**, permissão **Object Read & Write**, restrinja ao bucket `evve-uploads`.
5. Copie **Access Key ID** (`S3_ACCESS_KEY`) e **Secret Access Key** (`S3_SECRET_KEY`, aparece uma vez só).
6. O endpoint é `<ID_DA_CONTA>.r2.cloudflarestorage.com` (o ID aparece na página do R2). Sem `https://`. Será `S3_ENDPOINT`.

Guardado: `S3_ENDPOINT`, `S3_BUCKET=evve-uploads`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `UPLOADS_BASE_URL`.

## 2. Resend (e-mails)

1. Crie conta em resend.com → **Domains** → **Add Domain** (seu domínio, ex.: `seudominio.com`).
2. O Resend mostra registros DNS (SPF, DKIM). Adicione-os no painel DNS de onde o domínio foi registrado e clique **Verify**. Pode levar de minutos a horas.
3. **API Keys** → **Create API Key** (permissão *Sending access*) → copie. Será `RESEND_API_KEY`.
4. Defina o remetente: `EMAIL_REMETENTE=Evve <no-reply@seudominio.com>`.

Sem domínio próprio o Resend só envia para o seu próprio e-mail: para produção o domínio é obrigatório.

## 3. Railway (API + banco)

### 3.1 Projeto e banco
1. Crie conta em railway.com (login com GitHub) e assine um plano (o gratuito é limitado).
2. **New Project** → **Deploy PostgreSQL**. Aguarde ficar ativo.
3. No serviço Postgres, aba **Backups** (se disponível no seu plano): ative backup diário. Se não houver, agende `pg_dump` por outro meio. O banco guarda dinheiro: não pule este passo.

### 3.2 A API
1. No mesmo projeto: **New** → **GitHub Repo** → autorize e escolha o repositório do Evve.
2. Abra o serviço criado → **Settings**:
   - **Root Directory**: `backend`
   - **Config-as-code / Railway Config File**: `/backend/railway.json` (se não detectar sozinho)
   - O builder Dockerfile é detectado automaticamente.
3. Aba **Variables** → adicione (cada linha `NOME=valor`):

   | Variável | Valor |
   |---|---|
   | `AMBIENTE_APP` | `production` |
   | `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` (referência ao serviço do banco; use o nome exato do serviço) |
   | `JWT_SECRET` | 64 caracteres aleatórios (gere abaixo) |
   | `CRON_SECRET` | 32+ caracteres aleatórios (gere abaixo) |
   | `FRONTEND_URL` | `https://seudominio.com` (preencha depois do passo 5) |
   | `BACKEND_URL` | `https://<dominio-da-api>` (passo 3.3) |
   | `STORAGE_DRIVER` | `s3` |
   | `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` | do passo 1 |
   | `UPLOADS_BASE_URL` | do passo 1 |
   | `RESEND_API_KEY`, `EMAIL_REMETENTE` | do passo 2 |
   | `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_WEBHOOK_SECRET` | passo 4 |
   | `GOOGLE_CLIENT_ID` | opcional |

   Gerar segredos no PowerShell:
   ```powershell
   -join ((48..57)+(65..90)+(97..122) | Get-Random -Count 64 | % {[char]$_})
   ```
   Rode duas vezes: um para `JWT_SECRET`, outro para `CRON_SECRET`.

### 3.3 Domínio da API
1. **Settings → Networking → Generate Domain** → copia algo como `evve-api-production.up.railway.app`. Esse é o `BACKEND_URL` (`https://…`). Opcional: **Custom Domain** `api.seudominio.com` (crie o CNAME que o Railway indicar).
2. Faça o **Deploy**. Nos **Logs** você deve ver "migrations aplicadas com sucesso" e a API subindo. Se aparecer "configuração de produção inválida", a mensagem diz qual variável corrigir.
3. Teste: abra `https://<dominio-da-api>/healthz` → deve responder ok.

## 4. Mercado Pago (conta do Evve)

1. Acesse mercadopago.com.br/developers → **Suas integrações** → **Criar aplicação** (produto: **Checkout Pro**).
2. Em **Credenciais de produção** (pode exigir ativar a conta/enviar dados do negócio), copie o **Access Token** → `MERCADOPAGO_ACCESS_TOKEN`.
3. **Webhooks** (Notificações) → configure a URL de produção: `https://<dominio-da-api>/api/v1/webhooks/mercadopago`, evento **Pagamentos**.
4. Copie a **assinatura secreta** do webhook → `MERCADOPAGO_WEBHOOK_SECRET`.
5. Cole ambos nas Variables do Railway (a API reinicia sozinha).
6. **Importante:** rotacione o token do projeto drenux que foi exposto (no painel dele, gere novas credenciais). Não use aquele token no Evve.

## 5. Vercel (frontend)

1. **Antes**, edite `frontend/vercel.json`: troque `SEU-SERVICO.up.railway.app` pelo domínio da API (sem `https://`). Faça commit e push (posso fazer por você).
2. Crie conta em vercel.com (login com GitHub) → **Add New → Project** → importe o repositório.
3. Configure:
   - **Root Directory**: `frontend`
   - **Framework Preset**: Vite (detecta sozinho)
4. **Environment Variables**:
   - `BACKEND_API_URL` = `https://<dominio-da-api>/api` (usada pelas prévias de link)
   - `VITE_NOME_PLATAFORMA` = `Evve` (opcional)
   - Não defina `VITE_API_URL`: o padrão `/api` é o correto por causa do proxy.
5. **Deploy**. Depois, **Settings → Domains** → adicione `seudominio.com` e siga as instruções de DNS.
6. Volte ao Railway e ajuste `FRONTEND_URL=https://seudominio.com` (sem barra final). A API reinicia.

## 6. GitHub (jobs periódicos)

1. No repositório: **Settings → Secrets and variables → Actions → New repository secret**:
   - `BACKEND_URL` = `https://<dominio-da-api>` (sem barra final)
   - `CRON_SECRET` = o mesmo valor do Railway
2. Aba **Actions** → workflow **jobs** → **Run workflow** para testar. Deve ficar verde.
3. Os agendamentos passam a rodar sozinhos (reservas a cada 5 min, repasses diários, reembolsos por hora). O GitHub pode atrasar alguns minutos, o que é aceitável para esses jobs.

## 7. Primeiro admin

1. No site, cadastre sua conta e confirme o e-mail.
2. No Railway: serviço **Postgres** → aba **Data** (ou **Connect** para usar um cliente SQL) e execute:
   ```sql
   UPDATE usuarios SET papel_plataforma = 'admin_plataforma' WHERE email = 'seu@email.com';
   ```
3. Saia e entre de novo no site para o papel valer.

## 8. Testes de aceite (com você)

- [ ] Cadastro e e-mail de verificação chegam (confira o spam).
- [ ] Criar evento com imagem: a imagem abre (está no R2).
- [ ] **Compra real de valor baixo** (ex.: R$ 5): pagamento aprovado, e-mail com QR chega, ingresso abre.
- [ ] Reembolsar essa compra pelo painel; conferir no Mercado Pago.
- [ ] Check-in pelo celular (a câmera exige HTTPS, que já está ativo).
- [ ] Actions verde; nos logs do Railway aparecem as chamadas de jobs.
- [ ] Confira o uso e o custo no Railway nos primeiros dias.

## Se algo falhar

- **API não sobe:** leia os Logs do Railway; a validação de produção diz o que falta.
- **Login não mantém sessão:** confira o rewrite `/api` no `vercel.json` e `FRONTEND_URL` exato (https, sem barra).
- **Webhook do MP não chega:** URL exata, e `BACKEND_URL` https; veja o histórico de entregas no painel do MP.
- **E-mail não chega:** domínio ainda não verificado no Resend, ou `EMAIL_REMETENTE` com outro domínio.
- **Upload falha:** `S3_*` errados ou token sem permissão de escrita no bucket.

## Ainda não testado em ambiente real

Entrega do e-mail com QR inline pelo Resend, pagamentos e reembolsos reais do Mercado Pago, upload no R2, câmera e modo offline em celular.
