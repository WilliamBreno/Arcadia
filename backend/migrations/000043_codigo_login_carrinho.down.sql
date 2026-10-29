DELETE FROM lembretes_enviados WHERE tipo = 'carrinho';
ALTER TABLE lembretes_enviados DROP CONSTRAINT IF EXISTS lembretes_enviados_tipo_check;
ALTER TABLE lembretes_enviados ADD CONSTRAINT lembretes_enviados_tipo_check CHECK (tipo IN ('7d', '1d', 'pos'));
DROP INDEX IF EXISTS pedidos_recuperacao_pendente_idx;
ALTER TABLE pedidos DROP COLUMN IF EXISTS recuperacao_em;
DROP TABLE IF EXISTS codigos_login;
