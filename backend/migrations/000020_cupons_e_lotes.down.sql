ALTER TABLE itens_pedido DROP COLUMN IF EXISTS desconto_centavos, DROP COLUMN IF EXISTS cupom_id;
DROP TABLE IF EXISTS cupons;
ALTER TABLE tipos_ingresso DROP COLUMN IF EXISTS lote_grupo;
