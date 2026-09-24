CREATE TABLE organizador_membros (
    id             BIGSERIAL PRIMARY KEY,
    organizador_id BIGINT      NOT NULL REFERENCES organizadores (id) ON DELETE CASCADE,
    usuario_id     BIGINT      NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    papel          VARCHAR(20) NOT NULL DEFAULT 'gestor' CHECK (papel IN ('gestor')),
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organizador_id, usuario_id)
);

CREATE INDEX organizador_membros_usuario_idx ON organizador_membros (usuario_id);
