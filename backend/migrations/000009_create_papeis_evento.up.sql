CREATE TABLE papeis_evento (
    id         BIGSERIAL PRIMARY KEY,
    evento_id  BIGINT      NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    usuario_id BIGINT      NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    papel      VARCHAR(20) NOT NULL CHECK (papel IN ('organizador', 'jurado', 'participante', 'staff')),
    origem     VARCHAR(20) NOT NULL CHECK (origem IN ('convite', 'inscricao')),
    status     VARCHAR(20) NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'confirmado', 'removido')),
    convite_id BIGINT REFERENCES convites (id) ON DELETE SET NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Único (evento, usuario, papel) — seção 5 do plano.
CREATE UNIQUE INDEX papeis_evento_unico_idx ON papeis_evento (evento_id, usuario_id, papel);
CREATE INDEX papeis_evento_usuario_id_idx ON papeis_evento (usuario_id);
