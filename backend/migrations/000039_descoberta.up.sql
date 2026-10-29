-- Item 6.2: coleções curadas pelo admin ("Rolês geek do mês", "Grátis neste
-- fim de semana"...), exibidas na página inicial e em /c/:slug.
CREATE TABLE colecoes (
    id         BIGSERIAL PRIMARY KEY,
    slug       VARCHAR(120) NOT NULL UNIQUE,
    titulo     VARCHAR(120) NOT NULL,
    descricao  TEXT         NOT NULL DEFAULT '',
    ordem      INT          NOT NULL DEFAULT 0,
    ativa      BOOLEAN      NOT NULL DEFAULT true,
    criado_em  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE colecao_eventos (
    colecao_id BIGINT NOT NULL REFERENCES colecoes (id) ON DELETE CASCADE,
    evento_id  BIGINT NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    ordem      INT    NOT NULL DEFAULT 0,
    PRIMARY KEY (colecao_id, evento_id)
);

-- Consultas de "Em alta" (vendas recentes por evento) e de distância.
CREATE INDEX IF NOT EXISTS itens_pedido_criado_em_idx ON itens_pedido (criado_em);
CREATE INDEX IF NOT EXISTS eventos_publicos_inicio_idx ON eventos (inicio_em) WHERE status = 'publicado' AND visibilidade = 'publico';
