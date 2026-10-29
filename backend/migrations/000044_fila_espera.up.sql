-- Item 7.3: lista de espera de ingresso esgotado (qualquer evento; a fila
-- do item 3.2 continua sendo a da aprovação manual).
-- Quando uma vaga volta, ela fica guardada só para a primeira pessoa da
-- fila, na quantidade que ela pediu, até prazo_em (2 h; 30 min perto do
-- evento). Ninguém mais compra essa vaga enquanto isso.
--   aguardando -> convidado -> reservou -> comprou
--                           -> expirou (prazo passou ou reserva não paga)
--   saiu: a pessoa deixou a fila.
CREATE TABLE fila_espera (
    id               BIGSERIAL PRIMARY KEY,
    evento_id        BIGINT      NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    tipo_ingresso_id BIGINT      NOT NULL REFERENCES tipos_ingresso (id) ON DELETE CASCADE,
    usuario_id       BIGINT      NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    quantidade       INT         NOT NULL CHECK (quantidade BETWEEN 1 AND 20),
    status           VARCHAR(12) NOT NULL DEFAULT 'aguardando'
        CHECK (status IN ('aguardando', 'convidado', 'reservou', 'comprou', 'expirou', 'saiu')),
    convidado_em     TIMESTAMPTZ,
    prazo_em         TIMESTAMPTZ,
    pedido_id        BIGINT REFERENCES pedidos (id),
    criado_em        TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Uma entrada ativa por pessoa e tipo de ingresso.
CREATE UNIQUE INDEX fila_espera_ativa_idx ON fila_espera (tipo_ingresso_id, usuario_id)
    WHERE status IN ('aguardando', 'convidado', 'reservou');
CREATE INDEX fila_espera_ordem_idx ON fila_espera (tipo_ingresso_id, criado_em) WHERE status = 'aguardando';
CREATE INDEX fila_espera_convites_idx ON fila_espera (prazo_em) WHERE status = 'convidado';
