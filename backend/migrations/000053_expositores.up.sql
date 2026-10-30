-- Item 8.6: artist alley e expositores. Decisões do dono (30/10/2026):
-- candidatura com curadoria (aprovado escolhe-se a mesa e paga em 48 h);
-- mesma taxa do ingresso, paga pelo expositor; reembolso igual ao do
-- ingresso (garantia opcional, arrependimento de 7 dias; remoção ou evento
-- cancelado = tudo de volta); mesas numeradas por setor e credenciais QR.
CREATE TABLE tipos_espaco (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT       NOT NULL REFERENCES eventos (id),
    nome           VARCHAR(80)  NOT NULL,
    descricao      TEXT         NOT NULL DEFAULT '',
    preco_centavos BIGINT       NOT NULL CHECK (preco_centavos = 0 OR preco_centavos >= 100),
    credenciais    INT          NOT NULL DEFAULT 2 CHECK (credenciais BETWEEN 1 AND 10),
    ativo          BOOLEAN      NOT NULL DEFAULT true,
    criado_em      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX tipos_espaco_evento_idx ON tipos_espaco (evento_id);

CREATE TABLE espacos (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT      NOT NULL REFERENCES eventos (id),
    tipo_espaco_id BIGINT      NOT NULL REFERENCES tipos_espaco (id),
    setor          VARCHAR(20) NOT NULL,
    codigo         VARCHAR(20) NOT NULL,
    ordem          INT         NOT NULL DEFAULT 0,
    UNIQUE (evento_id, codigo)
);

CREATE TABLE expositores (
    id                 BIGSERIAL PRIMARY KEY,
    evento_id          BIGINT       NOT NULL REFERENCES eventos (id),
    usuario_id         BIGINT       NOT NULL REFERENCES usuarios (id),
    tipo_espaco_id     BIGINT       NOT NULL REFERENCES tipos_espaco (id),
    espaco_id          BIGINT REFERENCES espacos (id),
    marca              VARCHAR(80)  NOT NULL,
    categoria          VARCHAR(40)  NOT NULL DEFAULT '',
    descricao          TEXT         NOT NULL DEFAULT '',
    portfolio_url      VARCHAR(300) NOT NULL DEFAULT '',
    instagram          VARCHAR(60)  NOT NULL DEFAULT '',
    responsavel_nome   VARCHAR(120) NOT NULL,
    telefone           VARCHAR(30)  NOT NULL DEFAULT '',
    status             VARCHAR(20)  NOT NULL DEFAULT 'pendente'
        CHECK (status IN ('pendente', 'aguardando_pagamento', 'aprovado', 'rejeitado', 'desistiu')),
    motivo_rejeicao    TEXT         NOT NULL DEFAULT '',
    prazo_pagamento_em TIMESTAMPTZ,
    criado_em          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (evento_id, usuario_id)
);
-- Uma mesa só pode estar com um expositor ativo.
CREATE UNIQUE INDEX expositores_espaco_un ON expositores (espaco_id)
    WHERE espaco_id IS NOT NULL AND status IN ('aguardando_pagamento', 'aprovado');

-- Credenciais (expositor + ajudantes). O QR é EVVEX:<id>:<HMAC>, nada guardado.
CREATE TABLE credenciais_expositor (
    id           BIGSERIAL PRIMARY KEY,
    expositor_id BIGINT       NOT NULL REFERENCES expositores (id) ON DELETE CASCADE,
    nome         VARCHAR(120) NOT NULL DEFAULT '',
    primeira_entrada_em TIMESTAMPTZ,
    ultima_entrada_em   TIMESTAMPTZ,
    ativa        BOOLEAN      NOT NULL DEFAULT true,
    criado_em    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX credenciais_expositor_idx ON credenciais_expositor (expositor_id);

-- O pagamento reaproveita o pedido (como a inscrição de talento, 8.4).
ALTER TABLE tipos_ingresso ADD COLUMN espaco_expositor BOOLEAN NOT NULL DEFAULT false;
CREATE UNIQUE INDEX tipos_ingresso_espaco_expositor_un ON tipos_ingresso (evento_id) WHERE espaco_expositor;
ALTER TABLE itens_pedido ADD COLUMN expositor_id BIGINT REFERENCES expositores (id);
CREATE INDEX itens_pedido_expositor_idx ON itens_pedido (expositor_id) WHERE expositor_id IS NOT NULL;
