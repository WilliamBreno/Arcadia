-- As coleções criadas pela migração ficam (podem ter sido editadas ou já
-- existir antes com o mesmo slug); sem as marcações dos organizadores elas
-- ficam vazias e não aparecem. Tirar as marcações antes de apagar as colunas
-- evita que eventos bloqueados pelo admin voltem a aparecer.
DELETE FROM colecao_eventos WHERE origem = 'organizador';
DROP INDEX IF EXISTS colecao_eventos_evento_idx;
ALTER TABLE colecao_eventos DROP COLUMN IF EXISTS criado_em, DROP COLUMN IF EXISTS bloqueado, DROP COLUMN IF EXISTS origem;
ALTER TABLE colecoes DROP CONSTRAINT IF EXISTS colecoes_periodo_ck,
    DROP COLUMN IF EXISTS na_home, DROP COLUMN IF EXISTS aberta_organizador, DROP COLUMN IF EXISTS regra_valor,
    DROP COLUMN IF EXISTS regra, DROP COLUMN IF EXISTS exibir_ate, DROP COLUMN IF EXISTS exibir_de, DROP COLUMN IF EXISTS icone;
