CREATE TABLE api_keys (
    id              BIGSERIAL PRIMARY KEY,
    organizador_id  BIGINT       NOT NULL REFERENCES organizadores (id) ON DELETE CASCADE,
    nome            VARCHAR(100) NOT NULL,
    prefixo         VARCHAR(12)  NOT NULL,
    hash            CHAR(64)     NOT NULL UNIQUE,
    ativo           BOOLEAN      NOT NULL DEFAULT true,
    criado_em       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    ultimo_uso_em   TIMESTAMPTZ
);

CREATE INDEX api_keys_organizador_idx ON api_keys (organizador_id);
