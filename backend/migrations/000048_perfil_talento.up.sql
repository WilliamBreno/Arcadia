-- Item 8.1: perfil público do talento (cosplayer, dançarino, cantor...).
-- Decisões do dono (30/10/2026):
--  - 18+ cria e publica quando quiser; de 12 a 17, só fica público depois que
--    o responsável autoriza pelo link enviado ao e-mail dele (e pode
--    desautorizar); menores de 12 não têm perfil público;
--  - perfil de menor mostra só nome artístico, bio, foto (se o responsável
--    autorizar), personagens e colocações; nunca nome real, idade, cidade,
--    redes ou contato; e não vai para o Google;
--  - nada é publicado sozinho: o talento cria o perfil; depois, os concursos
--    aprovados aparecem sozinhos e ele pode ocultar qualquer um.
CREATE TABLE perfis_talento (
    id               BIGSERIAL PRIMARY KEY,
    usuario_id       BIGINT       NOT NULL UNIQUE REFERENCES usuarios (id) ON DELETE CASCADE,
    slug             VARCHAR(80)  NOT NULL UNIQUE,
    nome_artistico   VARCHAR(60)  NOT NULL,
    bio              VARCHAR(500) NOT NULL DEFAULT '',
    foto_url         TEXT         NOT NULL DEFAULT '',
    cidade           VARCHAR(80)  NOT NULL DEFAULT '',
    uf               VARCHAR(2)   NOT NULL DEFAULT '',
    instagram        VARCHAR(40)  NOT NULL DEFAULT '',
    tiktok           VARCHAR(40)  NOT NULL DEFAULT '',
    youtube          VARCHAR(60)  NOT NULL DEFAULT '',
    -- Declarada pelo talento (sem documento). Vale a mais nova entre esta e
    -- as das fichas de concurso dele.
    data_nascimento  DATE         NOT NULL,
    publico          BOOLEAN      NOT NULL DEFAULT true,
    -- Responsável (menores de 18). O token do link fica só como hash.
    responsavel_nome  VARCHAR(120) NOT NULL DEFAULT '',
    responsavel_email VARCHAR(255) NOT NULL DEFAULT '',
    autorizacao_token_hash CHAR(64),
    autorizado_em     TIMESTAMPTZ,
    autoriza_foto     BOOLEAN      NOT NULL DEFAULT false,
    criado_em        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    atualizado_em    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX perfis_talento_token_idx ON perfis_talento (autorizacao_token_hash) WHERE autorizacao_token_hash IS NOT NULL;

-- O talento pode ocultar uma participação do perfil.
ALTER TABLE fichas_participacao ADD COLUMN oculta_no_perfil BOOLEAN NOT NULL DEFAULT false;
