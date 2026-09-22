CREATE TABLE eventos (
    id                          BIGSERIAL PRIMARY KEY,
    organizador_id              BIGINT       NOT NULL REFERENCES organizadores (id) ON DELETE CASCADE,
    local_id                    BIGINT REFERENCES locais (id) ON DELETE SET NULL,
    titulo                      VARCHAR(255) NOT NULL,
    slug                        VARCHAR(255) NOT NULL,
    descricao                   TEXT         NOT NULL DEFAULT '',
    categoria                   VARCHAR(100) NOT NULL DEFAULT '',
    capa_url                    TEXT         NOT NULL DEFAULT '',
    inicio_em                   TIMESTAMPTZ,
    fim_em                      TIMESTAMPTZ,
    timezone                    VARCHAR(50)  NOT NULL DEFAULT 'America/Maceio',
    classificacao_etaria        VARCHAR(10)  NOT NULL DEFAULT '',
    visibilidade                VARCHAR(20)  NOT NULL DEFAULT 'publico'
        CHECK (visibilidade IN ('publico', 'nao_listado', 'privado')),
    tipo_acesso                 VARCHAR(20)  NOT NULL DEFAULT 'ingresso'
        CHECK (tipo_acesso IN ('ingresso', 'cadastro')),
    status                      VARCHAR(20)  NOT NULL DEFAULT 'rascunho'
        CHECK (status IN ('rascunho', 'publicado', 'encerrado', 'cancelado')),
    modo_participantes          VARCHAR(20)  NOT NULL DEFAULT 'nenhum'
        CHECK (modo_participantes IN ('nenhum', 'convite', 'inscricao_aberta', 'ambos')),
    inscricao_talentos_inicio   TIMESTAMPTZ,
    inscricao_talentos_fim      TIMESTAMPTZ,
    capacidade_total            INT,
    garantia_habilitada         BOOLEAN      NOT NULL DEFAULT true,
    politica_cancelamento_texto TEXT         NOT NULL DEFAULT '',
    max_itens_por_pedido        INT          NOT NULL DEFAULT 10,
    publicado_em                TIMESTAMPTZ,
    cancelado_em                TIMESTAMPTZ,
    motivo_cancelamento         TEXT         NOT NULL DEFAULT '',
    criado_em                   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX eventos_slug_idx ON eventos (slug);
CREATE INDEX eventos_organizador_id_idx ON eventos (organizador_id);
