-- Item 9.2: comissão de divulgadores. Decisões do dono (30/10/2026): o
-- organizador escolhe, por divulgador, % do preço ou valor fixo por
-- ingresso; a Evve só calcula e mostra, o organizador paga por fora e
-- registra; conta ingresso pago e não cancelado (cancelou, a comissão sai),
-- e só fica "a pagar" depois do fim do evento; o divulgador vê as próprias
-- vendas por um link secreto, sem dados dos compradores.
ALTER TABLE afiliados
    ADD COLUMN comissao_tipo VARCHAR(12) NOT NULL DEFAULT 'nenhuma'
        CHECK (comissao_tipo IN ('nenhuma', 'percentual', 'fixo')),
    ADD COLUMN comissao_bp INT NOT NULL DEFAULT 0 CHECK (comissao_bp BETWEEN 0 AND 5000),
    ADD COLUMN comissao_fixa_centavos BIGINT NOT NULL DEFAULT 0 CHECK (comissao_fixa_centavos BETWEEN 0 AND 100000),
    ADD COLUMN painel_token_hash VARCHAR(64) UNIQUE;

-- Comissão gravada no item na hora da compra (mudar a regra vale para as
-- próximas vendas, nunca muda o que já foi vendido).
ALTER TABLE itens_pedido
    ADD COLUMN comissao_afiliado_centavos BIGINT NOT NULL DEFAULT 0 CHECK (comissao_afiliado_centavos >= 0);

CREATE TABLE comissoes_pagas (
    id             BIGSERIAL PRIMARY KEY,
    afiliado_id    BIGINT      NOT NULL REFERENCES afiliados (id) ON DELETE CASCADE,
    valor_centavos BIGINT      NOT NULL CHECK (valor_centavos > 0),
    observacao     TEXT        NOT NULL DEFAULT '',
    registrado_por BIGINT REFERENCES usuarios (id),
    pago_em        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX comissoes_pagas_afiliado_idx ON comissoes_pagas (afiliado_id);
