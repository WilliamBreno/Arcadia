-- Item 11.1: perfil público de grupo (dança, cosplay, canto, teatro) e de
-- artista solo. Artista solo é um grupo de uma pessoa só (solo = true), para
-- dividir portfólio, agenda e contratação com os grupos.
CREATE TABLE grupos (
    id               BIGSERIAL PRIMARY KEY,
    slug             VARCHAR(90)  NOT NULL UNIQUE,
    nome             VARCHAR(80)  NOT NULL,
    tipo             VARCHAR(12)  NOT NULL CHECK (tipo IN ('danca', 'cosplay', 'canto', 'teatro', 'outro')),
    solo             BOOLEAN      NOT NULL DEFAULT false,
    bio              VARCHAR(1500) NOT NULL DEFAULT '',
    cidade           VARCHAR(80)  NOT NULL DEFAULT '',
    uf               VARCHAR(2)   NOT NULL DEFAULT '',
    instagram        VARCHAR(40)  NOT NULL DEFAULT '',
    tiktok           VARCHAR(40)  NOT NULL DEFAULT '',
    youtube          VARCHAR(60)  NOT NULL DEFAULT '',
    logo_url         TEXT         NOT NULL DEFAULT '',
    capa_url         TEXT         NOT NULL DEFAULT '',
    lider_usuario_id BIGINT       NOT NULL REFERENCES usuarios (id),
    publico          BOOLEAN      NOT NULL DEFAULT true,
    -- Selo do Instagram: código na bio, conferido pelo admin.
    instagram_codigo        VARCHAR(16) NOT NULL DEFAULT '',
    instagram_pedido_em     TIMESTAMPTZ,
    instagram_verificado_em TIMESTAMPTZ,
    criado_em        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    atualizado_em    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX grupos_lider_idx ON grupos (lider_usuario_id);
CREATE INDEX grupos_vitrine_idx ON grupos (tipo, uf) WHERE publico;

-- Integrante: o líder convida pelo e-mail; só aparece depois que a pessoa
-- aceita com a conta Evve daquele e-mail. Menor segue as regras do item 8.1.
CREATE TABLE grupo_integrantes (
    id                 BIGSERIAL PRIMARY KEY,
    grupo_id           BIGINT       NOT NULL REFERENCES grupos (id) ON DELETE CASCADE,
    usuario_id         BIGINT       REFERENCES usuarios (id) ON DELETE CASCADE,
    email              VARCHAR(255) NOT NULL,
    convite_token_hash CHAR(64),
    nome_artistico     VARCHAR(60)  NOT NULL,
    funcao             VARCHAR(60)  NOT NULL DEFAULT '',
    desde_ano          INT          CHECK (desde_ano BETWEEN 1950 AND 2100),
    foto_url           TEXT         NOT NULL DEFAULT '',
    instagram          VARCHAR(40)  NOT NULL DEFAULT '',
    data_nascimento    DATE,
    responsavel_nome   VARCHAR(120) NOT NULL DEFAULT '',
    responsavel_email  VARCHAR(255) NOT NULL DEFAULT '',
    autorizacao_token_hash CHAR(64),
    autorizado_em      TIMESTAMPTZ,
    autoriza_foto      BOOLEAN      NOT NULL DEFAULT false,
    status             VARCHAR(10)  NOT NULL DEFAULT 'convidado' CHECK (status IN ('convidado', 'ativo', 'recusou', 'saiu', 'removido')),
    ordem              INT          NOT NULL DEFAULT 0,
    criado_em          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    aceito_em          TIMESTAMPTZ,
    UNIQUE (grupo_id, email)
);
CREATE UNIQUE INDEX grupo_integrantes_convite_idx ON grupo_integrantes (convite_token_hash) WHERE convite_token_hash IS NOT NULL;
CREATE UNIQUE INDEX grupo_integrantes_autorizacao_idx ON grupo_integrantes (autorizacao_token_hash) WHERE autorizacao_token_hash IS NOT NULL;
CREATE INDEX grupo_integrantes_usuario_idx ON grupo_integrantes (usuario_id);

-- Portfólio: fotos enviadas e vídeos por link (YouTube, Instagram, TikTok).
CREATE TABLE grupo_midias (
    id        BIGSERIAL PRIMARY KEY,
    grupo_id  BIGINT      NOT NULL REFERENCES grupos (id) ON DELETE CASCADE,
    tipo      VARCHAR(6)  NOT NULL CHECK (tipo IN ('foto', 'video')),
    url       TEXT        NOT NULL,
    legenda   VARCHAR(140) NOT NULL DEFAULT '',
    evento_id BIGINT      REFERENCES eventos (id) ON DELETE SET NULL,
    ordem     INT         NOT NULL DEFAULT 0,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX grupo_midias_grupo_idx ON grupo_midias (grupo_id, ordem);

-- Atrações: o organizador convida o grupo para o evento; o líder confirma.
CREATE TABLE atracoes (
    id            BIGSERIAL PRIMARY KEY,
    evento_id     BIGINT      NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    grupo_id      BIGINT      NOT NULL REFERENCES grupos (id) ON DELETE CASCADE,
    status        VARCHAR(10) NOT NULL DEFAULT 'convidado' CHECK (status IN ('convidado', 'confirmado', 'recusado')),
    convidado_por BIGINT      REFERENCES usuarios (id) ON DELETE SET NULL,
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
    decidido_em   TIMESTAMPTZ,
    UNIQUE (evento_id, grupo_id)
);
CREATE INDEX atracoes_grupo_idx ON atracoes (grupo_id, status);

-- Inscrição em concurso pelo grupo: o resultado vai para o perfil do grupo.
ALTER TABLE fichas_participacao ADD COLUMN grupo_id BIGINT REFERENCES grupos (id) ON DELETE SET NULL;
CREATE INDEX fichas_participacao_grupo_idx ON fichas_participacao (grupo_id) WHERE grupo_id IS NOT NULL;
