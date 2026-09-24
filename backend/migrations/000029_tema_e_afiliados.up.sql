ALTER TABLE eventos ADD COLUMN cor_tema VARCHAR(7) NOT NULL DEFAULT '';

CREATE TABLE afiliados (
    id        BIGSERIAL PRIMARY KEY,
    evento_id BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    nome      VARCHAR(100) NOT NULL,
    codigo    VARCHAR(20)  NOT NULL UNIQUE,
    ativo     BOOLEAN      NOT NULL DEFAULT true,
    criado_em TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX afiliados_evento_idx ON afiliados (evento_id);

ALTER TABLE pedidos ADD COLUMN afiliado_id BIGINT REFERENCES afiliados (id);
