DROP INDEX IF EXISTS itens_pedido_lugar_ativo;
ALTER TABLE itens_pedido DROP COLUMN IF EXISTS lugar_rotulo, DROP COLUMN IF EXISTS lugar_id;
DROP TABLE IF EXISTS lugares;
DROP TABLE IF EXISTS setor_tipos;
DROP TABLE IF EXISTS setores_mapa;
