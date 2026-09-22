CREATE TABLE locais (
    id             BIGSERIAL PRIMARY KEY,
    organizador_id BIGINT           NOT NULL REFERENCES organizadores (id) ON DELETE CASCADE,
    nome           VARCHAR(255)     NOT NULL,
    logradouro     VARCHAR(255)     NOT NULL DEFAULT '',
    numero         VARCHAR(20)      NOT NULL DEFAULT '',
    bairro         VARCHAR(255)     NOT NULL DEFAULT '',
    cidade         VARCHAR(255)     NOT NULL,
    uf             CHAR(2)          NOT NULL,
    cep            VARCHAR(9)       NOT NULL DEFAULT '',
    latitude       DOUBLE PRECISION,
    longitude      DOUBLE PRECISION,
    capacidade     INT,
    observacoes    TEXT             NOT NULL DEFAULT '',
    criado_em      TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE INDEX locais_organizador_id_idx ON locais (organizador_id);
