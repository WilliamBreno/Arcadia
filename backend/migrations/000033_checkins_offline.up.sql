-- Entradas lidas offline e sincronizadas depois. entrada_id é gerado no
-- aparelho: reenviar o mesmo lote (rede caiu no meio) não duplica nada.
CREATE TABLE checkins_offline (
    id              BIGSERIAL PRIMARY KEY,
    evento_id       BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    entrada_id      VARCHAR(64)  NOT NULL,
    item_id         BIGINT       REFERENCES itens_pedido (id),
    codigo          VARCHAR(20)  NOT NULL,
    sessao_id       BIGINT       REFERENCES sessoes (id),
    staff_usuario_id BIGINT      NOT NULL REFERENCES usuarios (id),
    lido_em         TIMESTAMPTZ  NOT NULL,
    sincronizado_em TIMESTAMPTZ  NOT NULL DEFAULT now(),
    resultado       VARCHAR(20)  NOT NULL CHECK (resultado IN ('aceito', 'conflito', 'cancelado', 'nao_encontrado', 'sessao_invalida')),
    UNIQUE (evento_id, entrada_id)
);

CREATE INDEX checkins_offline_evento_resultado_idx ON checkins_offline (evento_id, resultado);
