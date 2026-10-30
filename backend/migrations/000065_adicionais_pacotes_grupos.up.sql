-- Item 10.3: adicionais (camiseta, estacionamento, kit...), pacotes com
-- preço fechado e compra em grupo com cada um pagando a sua parte.

-- Adicional = tipo de ingresso marcado (preço, estoque, taxa, reembolso e
-- repasse iguais aos do ingresso). Não entra na portaria: é entregue.
ALTER TABLE tipos_ingresso ADD COLUMN adicional BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE pacotes (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    nome           VARCHAR(80) NOT NULL,
    descricao      VARCHAR(300) NOT NULL DEFAULT '',
    preco_centavos BIGINT NOT NULL CHECK (preco_centavos > 0),
    quantidade     INT NOT NULL CHECK (quantidade > 0),
    max_por_pedido INT NOT NULL DEFAULT 4 CHECK (max_por_pedido BETWEEN 1 AND 20),
    ativo          BOOLEAN NOT NULL DEFAULT true,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX pacotes_evento ON pacotes (evento_id);

CREATE TABLE pacote_itens (
    pacote_id        BIGINT NOT NULL REFERENCES pacotes(id) ON DELETE CASCADE,
    tipo_ingresso_id BIGINT NOT NULL REFERENCES tipos_ingresso(id) ON DELETE CASCADE,
    quantidade       INT NOT NULL CHECK (quantidade BETWEEN 1 AND 20),
    PRIMARY KEY (pacote_id, tipo_ingresso_id)
);

-- Compra em grupo: quem cria segura N ingressos de um tipo até expira_em;
-- cada pessoa paga a sua parte pelo link (token).
CREATE TABLE grupos_compra (
    id               BIGSERIAL PRIMARY KEY,
    evento_id        BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    tipo_ingresso_id BIGINT NOT NULL REFERENCES tipos_ingresso(id) ON DELETE CASCADE,
    criador_id       BIGINT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    quantidade       INT NOT NULL CHECK (quantidade BETWEEN 2 AND 10),
    token            VARCHAR(40) NOT NULL UNIQUE,
    expira_em        TIMESTAMPTZ NOT NULL,
    status           VARCHAR(10) NOT NULL DEFAULT 'aberto' CHECK (status IN ('aberto', 'cancelado')),
    criado_em        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX grupos_compra_tipo_aberto ON grupos_compra (tipo_ingresso_id, expira_em) WHERE status = 'aberto';
CREATE INDEX grupos_compra_criador ON grupos_compra (criador_id);

ALTER TABLE itens_pedido
    ADD COLUMN adicional BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN pacote_id BIGINT REFERENCES pacotes(id) ON DELETE SET NULL,
    ADD COLUMN grupo_id BIGINT REFERENCES grupos_compra(id) ON DELETE SET NULL;
CREATE INDEX itens_pedido_pacote ON itens_pedido (pacote_id) WHERE pacote_id IS NOT NULL;
CREATE INDEX itens_pedido_grupo ON itens_pedido (grupo_id) WHERE grupo_id IS NOT NULL;
