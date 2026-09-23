CREATE TABLE pedidos (
    id              BIGSERIAL PRIMARY KEY,
    usuario_id      BIGINT       NOT NULL REFERENCES usuarios (id),
    evento_id       BIGINT       NOT NULL REFERENCES eventos (id),
    status          VARCHAR(30)  NOT NULL DEFAULT 'aguardando_pagamento'
        CHECK (status IN ('aberto', 'aguardando_pagamento', 'pago', 'expirado', 'cancelado', 'reembolsado_parcial', 'reembolsado')),
    total_centavos  BIGINT       NOT NULL DEFAULT 0,
    expira_em       TIMESTAMPTZ,
    mp_preference_id VARCHAR(100),
    criado_em       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX pedidos_usuario_id_idx ON pedidos (usuario_id);
CREATE INDEX pedidos_evento_id_idx ON pedidos (evento_id);
CREATE INDEX pedidos_status_expira_em_idx ON pedidos (status, expira_em);
