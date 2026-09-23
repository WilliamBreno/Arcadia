CREATE TABLE itens_pedido (
    id                     BIGSERIAL PRIMARY KEY,
    pedido_id              BIGINT       NOT NULL REFERENCES pedidos (id) ON DELETE CASCADE,
    tipo_ingresso_id       BIGINT       NOT NULL REFERENCES tipos_ingresso (id),
    titular_nome           VARCHAR(255) NOT NULL DEFAULT '',
    titular_email          VARCHAR(255) NOT NULL DEFAULT '',
    preco_centavos         BIGINT       NOT NULL,
    taxa_plataforma_centavos BIGINT     NOT NULL,
    garantia_contratada    BOOLEAN      NOT NULL DEFAULT false,
    garantia_centavos      BIGINT       NOT NULL DEFAULT 0,
    total_centavos         BIGINT       NOT NULL,
    status                 VARCHAR(20)  NOT NULL DEFAULT 'reservado'
        CHECK (status IN ('reservado', 'pago', 'utilizado', 'cancelado', 'reembolsado', 'expirado')),
    codigo                 VARCHAR(20)  NOT NULL,
    qr_token               VARCHAR(64)  NOT NULL,
    utilizado_em           TIMESTAMPTZ,
    cancelado_em           TIMESTAMPTZ,
    motivo_cancelamento    TEXT         NOT NULL DEFAULT '',
    criado_em              TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX itens_pedido_codigo_idx ON itens_pedido (codigo);
CREATE INDEX itens_pedido_pedido_id_idx ON itens_pedido (pedido_id);
CREATE INDEX itens_pedido_tipo_ingresso_status_idx ON itens_pedido (tipo_ingresso_id, status);
