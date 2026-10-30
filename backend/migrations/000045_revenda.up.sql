-- Item 7.4: revenda oficial ligada à lista de espera.
-- Decisões do dono (29/10/2026): preço = valor de face; o vendedor não paga
-- taxa a mais (recebe de volta o preço do ingresso; a taxa de serviço e a
-- garantia da compra dele não voltam); só dá para revender com o tipo de
-- ingresso esgotado; fecha no início do evento.
--   a_venda -> reservada (alguém está pagando) -> vendida
--           -> retirada (o dono desistiu)  -> encerrada (evento começou/cancelou)
CREATE TABLE revendas (
    id                BIGSERIAL PRIMARY KEY,
    item_id           BIGINT      NOT NULL REFERENCES itens_pedido (id),
    evento_id         BIGINT      NOT NULL REFERENCES eventos (id),
    tipo_ingresso_id  BIGINT      NOT NULL REFERENCES tipos_ingresso (id),
    vendedor_id       BIGINT      NOT NULL REFERENCES usuarios (id),
    -- O que volta para o vendedor: o preço que ele pagou pelo ingresso.
    valor_devolver_centavos BIGINT NOT NULL CHECK (valor_devolver_centavos > 0),
    status            VARCHAR(10) NOT NULL DEFAULT 'a_venda'
        CHECK (status IN ('a_venda', 'reservada', 'vendida', 'retirada', 'encerrada')),
    item_comprador_id BIGINT REFERENCES itens_pedido (id),
    vendida_em        TIMESTAMPTZ,
    criado_em         TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Um anúncio ativo por ingresso; a busca é pelos mais antigos de cada tipo.
CREATE UNIQUE INDEX revendas_item_ativa_idx ON revendas (item_id) WHERE status IN ('a_venda', 'reservada');
CREATE INDEX revendas_tipo_a_venda_idx ON revendas (tipo_ingresso_id, criado_em) WHERE status = 'a_venda';
CREATE INDEX revendas_comprador_idx ON revendas (item_comprador_id) WHERE item_comprador_id IS NOT NULL;

-- O estorno ao vendedor usa a fila de reembolsos (com nova tentativa).
ALTER TABLE reembolsos DROP CONSTRAINT IF EXISTS reembolsos_tipo_check;
ALTER TABLE reembolsos ADD CONSTRAINT reembolsos_tipo_check
    CHECK (tipo IN ('garantia', 'arrependimento', 'evento_cancelado', 'manual', 'revenda'));
