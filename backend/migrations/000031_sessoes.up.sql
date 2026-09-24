-- Eventos com várias sessões (datas). Estoque continua por tipo de ingresso
-- (compartilhado entre sessões); tipo com sessao_id só vale para aquela sessão.
CREATE TABLE sessoes (
    id           BIGSERIAL PRIMARY KEY,
    evento_id    BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    titulo       VARCHAR(150) NOT NULL DEFAULT '',
    inicio_em    TIMESTAMPTZ  NOT NULL,
    fim_em       TIMESTAMPTZ,
    status       VARCHAR(20)  NOT NULL DEFAULT 'ativa' CHECK (status IN ('ativa', 'cancelada')),
    cancelada_em TIMESTAMPTZ,
    motivo_cancelamento TEXT  NOT NULL DEFAULT '',
    criado_em    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CHECK (fim_em IS NULL OR fim_em >= inicio_em)
);

CREATE INDEX sessoes_evento_idx ON sessoes (evento_id, inicio_em);

ALTER TABLE tipos_ingresso ADD COLUMN sessao_id BIGINT REFERENCES sessoes (id);

-- Ingresso do evento todo entra uma vez por sessão.
CREATE TABLE checkins_sessao (
    id         BIGSERIAL PRIMARY KEY,
    item_id    BIGINT      NOT NULL REFERENCES itens_pedido (id),
    sessao_id  BIGINT      NOT NULL REFERENCES sessoes (id),
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_id, sessao_id)
);
