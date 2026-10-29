DELETE FROM lembretes_enviados WHERE tipo = 'pos';
ALTER TABLE lembretes_enviados DROP CONSTRAINT IF EXISTS lembretes_enviados_tipo_check;
ALTER TABLE lembretes_enviados ADD CONSTRAINT lembretes_enviados_tipo_check CHECK (tipo IN ('7d', '1d'));
DROP TABLE avaliacoes_publico;
