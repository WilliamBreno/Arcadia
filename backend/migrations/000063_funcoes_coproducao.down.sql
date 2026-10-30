-- Atenção: só desfaz se não houver evento com mais de um repasse ativo
-- (coprodução). Com coprodução já paga, reverter perderia a divisão.
ALTER TABLE repasses DROP COLUMN IF EXISTS percentual_bp;
DROP INDEX IF EXISTS repasses_evento_org_ativo_idx;
CREATE UNIQUE INDEX repasses_evento_ativo_idx ON repasses (evento_id) WHERE status <> 'cancelado';
DROP TABLE IF EXISTS coproducoes;
UPDATE organizador_membros SET papel = 'gestor' WHERE papel <> 'gestor';
ALTER TABLE organizador_membros DROP CONSTRAINT IF EXISTS organizador_membros_papel_check;
ALTER TABLE organizador_membros ADD CONSTRAINT organizador_membros_papel_check CHECK (papel IN ('gestor'));
