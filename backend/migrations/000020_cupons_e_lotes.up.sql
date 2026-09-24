-- Lotes: tipos de ingresso com o mesmo lote_grupo formam uma sequência
-- (ordenada por ordem, id); só o primeiro lote disponível é vendável.
ALTER TABLE tipos_ingresso ADD COLUMN lote_grupo VARCHAR(100) NOT NULL DEFAULT '';

CREATE TABLE cupons (
    id          BIGSERIAL PRIMARY KEY,
    evento_id   BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    codigo      VARCHAR(50)  NOT NULL,
    tipo        VARCHAR(20)  NOT NULL CHECK (tipo IN ('percentual', 'valor')),
    valor       BIGINT       NOT NULL CHECK (valor > 0),
    max_usos    INT,
    usos        INT          NOT NULL DEFAULT 0,
    valido_de   TIMESTAMPTZ,
    valido_ate  TIMESTAMPTZ,
    ativo       BOOLEAN      NOT NULL DEFAULT true,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CHECK (tipo <> 'percentual' OR valor <= 100)
);

CREATE UNIQUE INDEX cupons_evento_codigo_idx ON cupons (evento_id, codigo);

ALTER TABLE itens_pedido
    ADD COLUMN cupom_id BIGINT REFERENCES cupons (id),
    ADD COLUMN desconto_centavos BIGINT NOT NULL DEFAULT 0;
