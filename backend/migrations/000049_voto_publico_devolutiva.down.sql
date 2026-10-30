DROP TABLE IF EXISTS votos_publico;
ALTER TABLE eventos
    DROP COLUMN IF EXISTS devolutiva_liberada_em,
    DROP COLUMN IF EXISTS votacao_fechada_em,
    DROP COLUMN IF EXISTS votacao_aberta_em;
