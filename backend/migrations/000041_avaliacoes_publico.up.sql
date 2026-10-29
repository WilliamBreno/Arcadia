-- Item 6.5: avaliação do evento (e, somada, do organizador) por quem fez
-- check-in. Não confundir com "avaliacoes" (notas dos jurados, item 3.3).
CREATE TABLE avaliacoes_publico (
    id            BIGSERIAL PRIMARY KEY,
    evento_id     BIGINT      NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    usuario_id    BIGINT      NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    nota          SMALLINT    NOT NULL CHECK (nota BETWEEN 1 AND 5),
    comentario    TEXT        NOT NULL DEFAULT '' CHECK (length(comentario) <= 500),
    ocultada      BOOLEAN     NOT NULL DEFAULT false, -- moderação do admin
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (evento_id, usuario_id)
);
CREATE INDEX avaliacoes_publico_evento_idx ON avaliacoes_publico (evento_id) WHERE NOT ocultada;

-- Convite "Como foi?" depois do evento usa a mesma tabela dos lembretes.
ALTER TABLE lembretes_enviados DROP CONSTRAINT IF EXISTS lembretes_enviados_tipo_check;
ALTER TABLE lembretes_enviados ADD CONSTRAINT lembretes_enviados_tipo_check CHECK (tipo IN ('7d', '1d', 'pos'));
