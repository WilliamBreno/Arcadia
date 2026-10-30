DROP INDEX IF EXISTS eventos_telao_token_idx;
ALTER TABLE eventos
    DROP COLUMN IF EXISTS palco_atualizado_em,
    DROP COLUMN IF EXISTS telao_token_hash,
    DROP COLUMN IF EXISTS telao_modo,
    DROP COLUMN IF EXISTS palco_ficha_atual_id;
ALTER TABLE fichas_participacao
    DROP COLUMN IF EXISTS apresentou_em,
    DROP COLUMN IF EXISTS presente_em,
    DROP COLUMN IF EXISTS status_palco;
