# Deploy do Evve em produção

Arquitetura: **Vercel** (frontend + proxy `/api`) → **Render** (API Go em Docker + Postgres) · **Cloudflare R2** (uploads) · **Resend** (e-mail) · **Mercado Pago** (pagamentos) · **GitHub Actions** (jobs periódicos).

O frontend e a API precisam parecer a mesma origem: o cookie de sessão é `SameSite=Lax`, por isso a Vercel faz *rewrite* de `/api/*` para o Render (`frontend/vercel.json`).

## O que só o dono pode fazer (contas e chaves)

1. **Render**: criar conta → *New Blueprint* apontando para este repositório (`render.yaml`). Preencher as variáveis `sync:false` no painel. Anotar a URL final da API.
2. **Vercel**: importar o repositório, *Root Directory* = `frontend`. Variável `BACKEND_API_URL` = `https://<api>/api` (usada pelo `middleware.ts` das prévias de link). Ajustar o destino do rewrite em `frontend/vercel.json` se a URL da API não for `evve-api.onrender.com`.
3. **Cloudflare R2**: criar bucket, habilitar acesso público (domínio próprio ou `r2.dev`), criar token S3 → `S3_*` e `UPLOADS_BASE_URL`.
4. **Resend**: verificar o domínio de envio → `RESEND_API_KEY` e `EMAIL_REMETENTE`.
5. **Mercado Pago (conta do Evve)**: credenciais de **produção** → `MERCADOPAGO_ACCESS_TOKEN`; cadastrar o webhook `https://<api>/api/v1/webhooks/mercadopago` e copiar a assinatura secreta → `MERCADOPAGO_WEBHOOK_SECRET`. **Não use o token do projeto drenux (exposto anteriormente): rotacione-o.**
6. **GitHub → Settings → Secrets**: `BACKEND_URL` (https://…, sem barra final) e `CRON_SECRET` (o mesmo valor gerado no Render).
7. (Opcional) `GOOGLE_CLIENT_ID` para login Google (também `VITE_GOOGLE_CLIENT_ID` na Vercel, se usado no front).

## Variáveis obrigatórias da API

Com `AMBIENTE_APP=production` a API **não sobe** se: `JWT_SECRET` < 32 caracteres ou padrão, `CRON_SECRET` < 16, `FRONTEND_URL`/`BACKEND_URL` sem https, `DATABASE_URL` em localhost, `STORAGE_DRIVER` ≠ `s3`. Avisos (sobe, mas registra): token MP de teste/vazio, sem webhook secret, sem Resend, remetente de desenvolvimento.

## Ordem

1. Criar R2, Resend e conta MP.
2. Blueprint no Render (o container roda `migrate up` antes da API a cada deploy).
3. Configurar webhook do MP com a URL da API.
4. Deploy na Vercel; conferir o rewrite.
5. Secrets do GitHub; rodar o workflow **jobs** manualmente uma vez (`workflow_dispatch`).
6. Criar o primeiro usuário e promovê-lo a admin: `UPDATE usuarios SET papel_plataforma='admin_plataforma' WHERE email='...';`.

## Checklist de aceite (feito pelo dono)

- [ ] Cadastro + e-mail de verificação chega.
- [ ] Criar evento com imagem (upload no R2 abre em produção).
- [ ] Compra real de baixo valor: pagamento aprovado, e-mail com QR chega, ingresso abre.
- [ ] Reembolso da compra de teste pelo painel.
- [ ] Check-in do ingresso pelo celular (câmera exige HTTPS).
- [ ] Jobs rodando (aba Actions verde).

## Não testado ainda em ambiente real

Entrega do e-mail com QR inline pelo Resend, pagamentos/reembolsos reais do Mercado Pago, câmera e modo offline em celular, upload no R2.
