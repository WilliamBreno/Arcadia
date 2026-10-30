DROP INDEX IF EXISTS itens_pedido_expositor_idx;
ALTER TABLE itens_pedido DROP COLUMN IF EXISTS expositor_id;
DROP INDEX IF EXISTS tipos_ingresso_espaco_expositor_un;
ALTER TABLE tipos_ingresso DROP COLUMN IF EXISTS espaco_expositor;
DROP TABLE IF EXISTS credenciais_expositor;
DROP TABLE IF EXISTS expositores;
DROP TABLE IF EXISTS espacos;
DROP TABLE IF EXISTS tipos_espaco;
