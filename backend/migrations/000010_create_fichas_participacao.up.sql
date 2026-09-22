CREATE TABLE fichas_participacao (
    id                          BIGSERIAL PRIMARY KEY,
    evento_id                   BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    usuario_id                  BIGINT       NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    papel                       VARCHAR(20)  NOT NULL CHECK (papel IN ('jurado', 'participante')),
    origem                      VARCHAR(20)  NOT NULL CHECK (origem IN ('convite', 'inscricao')),
    nome                        VARCHAR(255) NOT NULL,
    nome_artistico              VARCHAR(255) NOT NULL DEFAULT '',
    instagram                   VARCHAR(255) NOT NULL DEFAULT '',
    data_nascimento             DATE,
    foto_url                    TEXT         NOT NULL DEFAULT '',
    telefone                    VARCHAR(30)  NOT NULL DEFAULT '',
    status                      VARCHAR(20)  NOT NULL DEFAULT 'rascunho'
        CHECK (status IN ('rascunho', 'pendente', 'aprovado', 'rejeitado', 'lista_espera', 'desistiu')),
    motivo_rejeicao             TEXT         NOT NULL DEFAULT '',
    ordem_apresentacao          INT,
    tipo_apresentacao           VARCHAR(20)
        CHECK (tipo_apresentacao IS NULL OR tipo_apresentacao IN ('cosplay', 'danca', 'canto', 'atuacao')),
    dados                       JSONB        NOT NULL DEFAULT '{}',
    responsavel_nome            VARCHAR(255) NOT NULL DEFAULT '',
    responsavel_contato         VARCHAR(255) NOT NULL DEFAULT '',
    autorizacao_responsavel_url TEXT         NOT NULL DEFAULT '',
    criado_em                   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX fichas_participacao_unico_idx ON fichas_participacao (evento_id, usuario_id, papel);
