CREATE TABLE mural_posts (
 id BIGSERIAL PRIMARY KEY,
 evento_id BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
 usuario_id BIGINT NOT NULL REFERENCES usuarios(id),
 chave VARCHAR(80) NOT NULL,
 texto VARCHAR(2000) NOT NULL DEFAULT '',
 status VARCHAR(12) NOT NULL DEFAULT 'pendente' CHECK(status IN ('pendente','aprovado','rejeitado','removido')),
 destaque BOOLEAN NOT NULL DEFAULT false,
 ia_status VARCHAR(16) NOT NULL DEFAULT 'pendente' CHECK(ia_status IN ('pendente','analisando','concluida','indisponivel')),
 ia_alerta BOOLEAN NOT NULL DEFAULT false,
 ia_motivos JSONB NOT NULL DEFAULT '[]',
 ia_tokens_entrada BIGINT NOT NULL DEFAULT 0,
 ia_tokens_saida BIGINT NOT NULL DEFAULT 0,
 ia_tentativas INT NOT NULL DEFAULT 0,
 ia_iniciada_em TIMESTAMPTZ,
 moderador_id BIGINT REFERENCES usuarios(id),
 motivo VARCHAR(500) NOT NULL DEFAULT '',
 moderado_em TIMESTAMPTZ,
 criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(evento_id,usuario_id,chave)
);
CREATE INDEX mural_posts_evento_idx ON mural_posts(evento_id,status,id DESC);
CREATE TABLE mural_fotos (
 id BIGSERIAL PRIMARY KEY,
 post_id BIGINT NOT NULL REFERENCES mural_posts(id) ON DELETE CASCADE,
 jpeg BYTEA NOT NULL
);
CREATE INDEX mural_fotos_post_idx ON mural_fotos(post_id);
CREATE TABLE mural_curtidas (
 post_id BIGINT NOT NULL REFERENCES mural_posts(id) ON DELETE CASCADE,
 usuario_id BIGINT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
 PRIMARY KEY(post_id,usuario_id)
);
CREATE TABLE mural_denuncias (
 id BIGSERIAL PRIMARY KEY,
 post_id BIGINT NOT NULL REFERENCES mural_posts(id) ON DELETE CASCADE,
 usuario_id BIGINT NOT NULL REFERENCES usuarios(id),
 motivo VARCHAR(500) NOT NULL,
 criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
 resolvido_em TIMESTAMPTZ,
 UNIQUE(post_id,usuario_id)
);
