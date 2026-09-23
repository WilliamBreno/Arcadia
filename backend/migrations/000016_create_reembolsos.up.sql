CREATE TABLE reembolsos (
    id            BIGSERIAL PRIMARY KEY,
    pagamento_id  BIGINT      NOT NULL REFERENCES pagamentos (id),
    item_id       BIGINT      NOT NULL REFERENCES itens_pedido (id),
    valor_centavos BIGINT     NOT NULL,
    tipo          VARCHAR(30) NOT NULL CHECK (tipo IN ('garantia', 'arrependimento', 'evento_cancelado', 'manual')),
    status        VARCHAR(20) NOT NULL DEFAULT 'pendente'
        CHECK (status IN ('pendente', 'concluido', 'falhou')),
    mp_refund_id  VARCHAR(50) NOT NULL DEFAULT '',
    motivo        TEXT        NOT NULL DEFAULT '',
    solicitado_por BIGINT     NOT NULL REFERENCES usuarios (id),
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reembolsos_pagamento_id_idx ON reembolsos (pagamento_id);
CREATE INDEX reembolsos_item_id_idx ON reembolsos (item_id);

-- Seed: nova config para a seção 7.4 do plano (padrão devolve a taxa no
-- arrependimento do CDC — a validar com advogado, ver seção 14).
INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('CDC_REEMBOLSA_TAXA', 'true', 'Se o reembolso por direito de arrependimento (CDC art. 49) devolve também a taxa da plataforma.')
ON CONFLICT (chave) DO NOTHING;
