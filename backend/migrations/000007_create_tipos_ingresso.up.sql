CREATE TABLE tipos_ingresso (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    nome           VARCHAR(255) NOT NULL,
    descricao      TEXT         NOT NULL DEFAULT '',
    preco_centavos BIGINT       NOT NULL DEFAULT 0,
    quantidade     INT          NOT NULL,
    vendas_inicio  TIMESTAMPTZ,
    vendas_fim     TIMESTAMPTZ,
    min_por_pedido INT          NOT NULL DEFAULT 1,
    max_por_pedido INT          NOT NULL DEFAULT 10,
    ordem          INT          NOT NULL DEFAULT 0,
    ativo          BOOLEAN      NOT NULL DEFAULT true,
    criado_em      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX tipos_ingresso_evento_id_idx ON tipos_ingresso (evento_id);
