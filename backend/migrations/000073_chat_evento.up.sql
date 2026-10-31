CREATE TABLE chat_mensagens (
 id BIGSERIAL PRIMARY KEY,
 evento_id BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
 usuario_id BIGINT NOT NULL REFERENCES usuarios(id),
 chave VARCHAR(80) NOT NULL,
 texto VARCHAR(3000) NOT NULL DEFAULT '',
 foto BYTEA,
 fixada BOOLEAN NOT NULL DEFAULT false,
 criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(evento_id, usuario_id, chave),
 CHECK (texto <> '' OR foto IS NOT NULL)
);
CREATE INDEX chat_mensagens_evento_idx ON chat_mensagens(evento_id, id);
CREATE TABLE chat_leituras (
 evento_id BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
 usuario_id BIGINT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
 lido_ate BIGINT NOT NULL DEFAULT 0,
 notificado_ate BIGINT NOT NULL DEFAULT 0,
 email_ativo BOOLEAN NOT NULL DEFAULT true,
 notificado_em TIMESTAMPTZ,
 PRIMARY KEY(evento_id, usuario_id)
);
CREATE TABLE chat_atencoes (
 id BIGSERIAL PRIMARY KEY,
 evento_id BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
 jurado_id BIGINT NOT NULL REFERENCES usuarios(id),
 mensagem_id BIGINT NOT NULL REFERENCES chat_mensagens(id) ON DELETE CASCADE,
 atendido_por BIGINT REFERENCES usuarios(id),
 criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
 atendido_em TIMESTAMPTZ
);
CREATE UNIQUE INDEX chat_atencoes_pendente_idx ON chat_atencoes(evento_id, jurado_id) WHERE atendido_em IS NULL;
