CREATE TABLE pagamentos (
    id                       BIGSERIAL PRIMARY KEY,
    pedido_id                BIGINT       NOT NULL REFERENCES pedidos (id),
    mp_payment_id            VARCHAR(50)  NOT NULL,
    metodo                   VARCHAR(20)  NOT NULL DEFAULT ''
        CHECK (metodo IN ('', 'pix', 'cartao')),
    status                   VARCHAR(30)  NOT NULL DEFAULT '',
    valor_centavos           BIGINT       NOT NULL DEFAULT 0,
    taxa_processador_centavos BIGINT      NOT NULL DEFAULT 0,
    payload_json              JSONB       NOT NULL DEFAULT '{}',
    criado_em                TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Idempotência do webhook (seção 7.3): um mp_payment_id só é processado uma vez.
CREATE UNIQUE INDEX pagamentos_mp_payment_id_idx ON pagamentos (mp_payment_id);
CREATE INDEX pagamentos_pedido_id_idx ON pagamentos (pedido_id);
