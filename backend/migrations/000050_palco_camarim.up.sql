-- Item 8.3: modo palco/telão e camarim (check-in de participantes).
-- Decisões do dono (30/10/2026): telão aberto por link secreto (trocável);
-- no telão, a foto de referência do personagem aparece sempre e a foto do
-- participante só se ele tiver 18+; camarim com QR próprio do participante
-- e busca por nome.
ALTER TABLE fichas_participacao
    ADD COLUMN status_palco VARCHAR(12) NOT NULL DEFAULT 'aguardando'
        CHECK (status_palco IN ('aguardando', 'presente', 'no_palco', 'apresentou', 'ausente')),
    ADD COLUMN presente_em   TIMESTAMPTZ,
    ADD COLUMN apresentou_em TIMESTAMPTZ;

ALTER TABLE eventos
    ADD COLUMN palco_ficha_atual_id BIGINT REFERENCES fichas_participacao (id) ON DELETE SET NULL,
    ADD COLUMN telao_modo           VARCHAR(12) NOT NULL DEFAULT 'intervalo'
        CHECK (telao_modo IN ('intervalo', 'apresentacao', 'resultado')),
    -- Só o hash do link secreto do telão.
    ADD COLUMN telao_token_hash     CHAR(64),
    ADD COLUMN palco_atualizado_em  TIMESTAMPTZ;
CREATE UNIQUE INDEX eventos_telao_token_idx ON eventos (telao_token_hash) WHERE telao_token_hash IS NOT NULL;
