ALTER TABLE eventos ADD COLUMN aprovacao_manual BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE solicitacoes_plateia (
    id          BIGSERIAL PRIMARY KEY,
    evento_id   BIGINT      NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    usuario_id  BIGINT      NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    status      VARCHAR(20) NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'aprovada', 'rejeitada', 'lista_espera')),
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    decidido_em TIMESTAMPTZ,
    UNIQUE (evento_id, usuario_id)
);

CREATE INDEX solicitacoes_plateia_evento_status_idx ON solicitacoes_plateia (evento_id, status);
