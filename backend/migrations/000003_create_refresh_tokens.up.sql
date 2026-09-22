CREATE TABLE refresh_tokens (
    id          BIGSERIAL PRIMARY KEY,
    usuario_id  BIGINT       NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    token_hash  VARCHAR(64)  NOT NULL,
    expira_em   TIMESTAMPTZ  NOT NULL,
    revogado_em TIMESTAMPTZ,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX refresh_tokens_token_hash_idx ON refresh_tokens (token_hash);
CREATE INDEX refresh_tokens_usuario_id_idx ON refresh_tokens (usuario_id);
