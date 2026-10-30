-- Item 7.6: perguntas personalizadas no checkout (tamanho de camiseta, como
-- soube do evento...). Decisões do dono (29/10/2026): respondidas uma vez por
-- ingresso; o organizador vê um aviso sobre dado sensível e explica para
-- que pede; as respostas são apagadas 90 dias depois do fim do evento.
CREATE TABLE perguntas_evento (
    id               BIGSERIAL PRIMARY KEY,
    evento_id        BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    texto            VARCHAR(200) NOT NULL,
    tipo             VARCHAR(10)  NOT NULL CHECK (tipo IN ('texto', 'opcao')),
    opcoes           JSONB        NOT NULL DEFAULT '[]',
    obrigatoria      BOOLEAN      NOT NULL DEFAULT false,
    -- Para que o organizador pede (mostrado ao comprador).
    finalidade       VARCHAR(200) NOT NULL,
    -- Vazio = vale para todos os tipos de ingresso.
    tipo_ingresso_ids JSONB       NOT NULL DEFAULT '[]',
    ordem            INT          NOT NULL DEFAULT 0,
    ativa            BOOLEAN      NOT NULL DEFAULT true,
    criado_em        TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX perguntas_evento_idx ON perguntas_evento (evento_id) WHERE ativa;

CREATE TABLE respostas_item (
    item_id     BIGINT       NOT NULL REFERENCES itens_pedido (id) ON DELETE CASCADE,
    pergunta_id BIGINT       NOT NULL REFERENCES perguntas_evento (id) ON DELETE CASCADE,
    resposta    VARCHAR(300) NOT NULL,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (item_id, pergunta_id)
);
CREATE INDEX respostas_item_pergunta_idx ON respostas_item (pergunta_id);
