CREATE TABLE midias_ficha (
    id        BIGSERIAL PRIMARY KEY,
    ficha_id  BIGINT      NOT NULL REFERENCES fichas_participacao (id) ON DELETE CASCADE,
    tipo      VARCHAR(30) NOT NULL CHECK (tipo IN ('foto_referencia', 'foto_cosplay', 'audio')),
    url       TEXT        NOT NULL,
    ordem     INT         NOT NULL DEFAULT 0,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX midias_ficha_ficha_id_idx ON midias_ficha (ficha_id);
