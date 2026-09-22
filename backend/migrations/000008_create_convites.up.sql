CREATE TABLE convites (
    id          BIGSERIAL PRIMARY KEY,
    evento_id   BIGINT      NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    tipo        VARCHAR(30) NOT NULL CHECK (tipo IN ('jurado', 'participante_especial')),
    token_hash  VARCHAR(64) NOT NULL,
    max_usos    INT,
    usos        INT         NOT NULL DEFAULT 0,
    expira_em   TIMESTAMPTZ,
    revogado_em TIMESTAMPTZ,
    criado_por  BIGINT      NOT NULL REFERENCES usuarios (id),
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX convites_token_hash_idx ON convites (token_hash);
CREATE INDEX convites_evento_id_idx ON convites (evento_id);
