CREATE TABLE cronograma_itens (
    id          BIGSERIAL PRIMARY KEY,
    evento_id   BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    titulo      VARCHAR(255) NOT NULL,
    descricao   TEXT         NOT NULL DEFAULT '',
    local       VARCHAR(255) NOT NULL DEFAULT '',
    inicio_em   TIMESTAMPTZ  NOT NULL,
    fim_em      TIMESTAMPTZ,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CHECK (fim_em IS NULL OR fim_em >= inicio_em)
);

CREATE INDEX cronograma_itens_evento_idx ON cronograma_itens (evento_id, inicio_em);
