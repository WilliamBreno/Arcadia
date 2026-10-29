-- Item 7.2: entrar com código de 6 dígitos no e-mail, direto no checkout.
-- Guardamos só o HMAC do código (nunca o código). Vale 10 minutos e aceita
-- até 5 tentativas; pedir um código novo invalida os anteriores.
CREATE TABLE codigos_login (
    id          BIGSERIAL PRIMARY KEY,
    email       VARCHAR(255) NOT NULL,
    codigo_hash CHAR(64)     NOT NULL,
    tentativas  INT          NOT NULL DEFAULT 0,
    expira_em   TIMESTAMPTZ  NOT NULL,
    usado_em    TIMESTAMPTZ,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX codigos_login_email_idx ON codigos_login (email, criado_em DESC);

-- Item 7.2: recuperação de carrinho. Marca o pedido expirado já analisado
-- (com ou sem e-mail), para o job não olhar de novo.
ALTER TABLE pedidos ADD COLUMN recuperacao_em TIMESTAMPTZ;
CREATE INDEX pedidos_recuperacao_pendente_idx ON pedidos (expira_em)
    WHERE status = 'expirado' AND recuperacao_em IS NULL;

-- Um e-mail de carrinho por pessoa e evento, pela mesma tabela dos lembretes.
ALTER TABLE lembretes_enviados DROP CONSTRAINT IF EXISTS lembretes_enviados_tipo_check;
ALTER TABLE lembretes_enviados ADD CONSTRAINT lembretes_enviados_tipo_check CHECK (tipo IN ('7d', '1d', 'pos', 'carrinho'));
