CREATE TABLE lancamentos (
    id             BIGSERIAL PRIMARY KEY,
    tipo           VARCHAR(30) NOT NULL CHECK (tipo IN (
        'venda_preco', 'taxa_plataforma', 'garantia', 'taxa_processador',
        'reembolso_preco', 'reembolso_taxa', 'reembolso_garantia',
        'custo_processador_perdido', 'repasse'
    )),
    valor_centavos BIGINT      NOT NULL,
    sinal          VARCHAR(1)  NOT NULL CHECK (sinal IN ('+', '-')),
    evento_id      BIGINT REFERENCES eventos (id),
    organizador_id BIGINT REFERENCES organizadores (id),
    pedido_id      BIGINT REFERENCES pedidos (id),
    item_id        BIGINT REFERENCES itens_pedido (id),
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lancamentos_evento_id_idx ON lancamentos (evento_id);
CREATE INDEX lancamentos_organizador_id_idx ON lancamentos (organizador_id);
CREATE INDEX lancamentos_pedido_id_idx ON lancamentos (pedido_id);
