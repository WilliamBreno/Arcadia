CREATE TABLE criterios_avaliacao (
    id                BIGSERIAL PRIMARY KEY,
    evento_id         BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    nome              VARCHAR(100) NOT NULL,
    peso              NUMERIC(8,2) NOT NULL CHECK (peso > 0),
    nota_min          NUMERIC(8,2) NOT NULL DEFAULT 0,
    nota_max          NUMERIC(8,2) NOT NULL DEFAULT 10,
    passo             NUMERIC(8,2) NOT NULL DEFAULT 1 CHECK (passo > 0),
    tipo_apresentacao VARCHAR(20),
    ordem             INT          NOT NULL DEFAULT 0,
    criado_em         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CHECK (nota_max > nota_min)
);

CREATE INDEX criterios_avaliacao_evento_idx ON criterios_avaliacao (evento_id);

CREATE TABLE avaliacoes (
    id                BIGSERIAL PRIMARY KEY,
    ficha_id          BIGINT       NOT NULL REFERENCES fichas_participacao (id) ON DELETE CASCADE,
    jurado_usuario_id BIGINT       NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    criterio_id       BIGINT       NOT NULL REFERENCES criterios_avaliacao (id) ON DELETE CASCADE,
    nota              NUMERIC(8,2) NOT NULL,
    comentario        TEXT         NOT NULL DEFAULT '',
    finalizada        BOOLEAN      NOT NULL DEFAULT false,
    criado_em         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (ficha_id, jurado_usuario_id, criterio_id)
);

CREATE INDEX avaliacoes_ficha_idx ON avaliacoes (ficha_id);

ALTER TABLE eventos ADD COLUMN resultado_liberado_em TIMESTAMPTZ;
