DROP INDEX IF EXISTS pedidos_porta_idx;
ALTER TABLE pedidos DROP COLUMN IF EXISTS metodo_porta, DROP COLUMN IF EXISTS vendido_por, DROP COLUMN IF EXISTS canal;
-- O usuário de sistema fica (pode ter pedidos antigos ligados a ele).
