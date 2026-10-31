DROP INDEX IF EXISTS fichas_participacao_grupo_idx;
ALTER TABLE fichas_participacao DROP COLUMN IF EXISTS grupo_id;
DROP TABLE IF EXISTS atracoes;
DROP TABLE IF EXISTS grupo_midias;
DROP TABLE IF EXISTS grupo_integrantes;
DROP TABLE IF EXISTS grupos;
