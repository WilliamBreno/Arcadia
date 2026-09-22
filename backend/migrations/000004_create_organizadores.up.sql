CREATE TABLE organizadores (
    id             BIGSERIAL PRIMARY KEY,
    usuario_id     BIGINT       NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    nome_publico   VARCHAR(255) NOT NULL,
    slug           VARCHAR(255) NOT NULL,
    descricao      TEXT         NOT NULL DEFAULT '',
    logo_url       TEXT         NOT NULL DEFAULT '',
    tipo_pessoa    VARCHAR(2)   NOT NULL CHECK (tipo_pessoa IN ('pf', 'pj')),
    documento      VARCHAR(20)  NOT NULL,
    chave_pix      VARCHAR(255) NOT NULL DEFAULT '',
    tipo_chave_pix VARCHAR(20)  NOT NULL DEFAULT '',
    instagram      VARCHAR(255) NOT NULL DEFAULT '',
    site           VARCHAR(255) NOT NULL DEFAULT '',
    status         VARCHAR(20)  NOT NULL DEFAULT 'pendente'
        CHECK (status IN ('pendente', 'ativo', 'suspenso')),
    criado_em      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Um perfil de organizador por usuário (decisão registrada na seção 14).
CREATE UNIQUE INDEX organizadores_usuario_id_idx ON organizadores (usuario_id);
CREATE UNIQUE INDEX organizadores_slug_idx ON organizadores (slug);
