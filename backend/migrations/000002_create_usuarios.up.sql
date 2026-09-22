CREATE TABLE usuarios (
    id                            BIGSERIAL PRIMARY KEY,
    nome                          VARCHAR(255) NOT NULL,
    email                         VARCHAR(255) NOT NULL,
    senha_hash                    VARCHAR(255),
    google_id                     VARCHAR(255),
    telefone                      VARCHAR(30)  NOT NULL DEFAULT '',
    avatar_url                    TEXT         NOT NULL DEFAULT '',
    email_verificado_em           TIMESTAMPTZ,
    email_verificacao_token_hash  VARCHAR(64),
    email_verificacao_expira_em   TIMESTAMPTZ,
    senha_reset_token_hash        VARCHAR(64),
    senha_reset_expira_em         TIMESTAMPTZ,
    papel_plataforma              VARCHAR(20)  NOT NULL DEFAULT 'usuario'
        CHECK (papel_plataforma IN ('usuario', 'admin_plataforma')),
    criado_em                     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX usuarios_email_idx ON usuarios (email);
CREATE UNIQUE INDEX usuarios_google_id_idx ON usuarios (google_id) WHERE google_id IS NOT NULL;
CREATE INDEX usuarios_email_verificacao_token_idx ON usuarios (email_verificacao_token_hash) WHERE email_verificacao_token_hash IS NOT NULL;
CREATE INDEX usuarios_senha_reset_token_idx ON usuarios (senha_reset_token_hash) WHERE senha_reset_token_hash IS NOT NULL;
