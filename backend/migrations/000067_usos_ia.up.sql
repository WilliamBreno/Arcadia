-- Item 10.5: gerações de rascunho com IA (limite diário por organizador).
CREATE TABLE usos_ia (
    id             BIGSERIAL PRIMARY KEY,
    organizador_id BIGINT NOT NULL REFERENCES organizadores(id) ON DELETE CASCADE,
    usuario_id     BIGINT REFERENCES usuarios(id) ON DELETE SET NULL,
    tokens_entrada INT NOT NULL DEFAULT 0,
    tokens_saida   INT NOT NULL DEFAULT 0,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX usos_ia_org_dia ON usos_ia (organizador_id, criado_em);
