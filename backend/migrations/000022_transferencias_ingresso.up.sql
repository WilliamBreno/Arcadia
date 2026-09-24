CREATE TABLE transferencias_ingresso (
    id              BIGSERIAL PRIMARY KEY,
    item_id         BIGINT       NOT NULL REFERENCES itens_pedido (id),
    feito_por       BIGINT       NOT NULL REFERENCES usuarios (id),
    de_nome         VARCHAR(255) NOT NULL,
    de_email        VARCHAR(255) NOT NULL,
    para_nome       VARCHAR(255) NOT NULL,
    para_email      VARCHAR(255) NOT NULL,
    codigo_anterior VARCHAR(20)  NOT NULL,
    criado_em       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX transferencias_ingresso_item_id_idx ON transferencias_ingresso (item_id);
