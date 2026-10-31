CREATE TABLE grupo_gestores (
    grupo_id BIGINT NOT NULL REFERENCES grupos(id) ON DELETE CASCADE,
    usuario_id BIGINT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    PRIMARY KEY (grupo_id, usuario_id)
);
CREATE INDEX grupo_gestores_usuario_idx ON grupo_gestores(usuario_id);
CREATE TABLE grupo_gestao_convites (
    id BIGSERIAL PRIMARY KEY,
    grupo_id BIGINT NOT NULL REFERENCES grupos(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    papel VARCHAR(10) NOT NULL CHECK (papel IN ('lider', 'cogestor')),
    antigo_cogestor BOOLEAN NOT NULL DEFAULT true,
    criado_por BIGINT NOT NULL REFERENCES usuarios(id),
    token_hash CHAR(64) NOT NULL UNIQUE,
    expira_em TIMESTAMPTZ NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX grupo_gestao_convites_grupo_idx ON grupo_gestao_convites(grupo_id);
CREATE UNIQUE INDEX grupo_transferencia_unica_idx ON grupo_gestao_convites(grupo_id) WHERE papel = 'lider';
CREATE UNIQUE INDEX grupo_cogestor_convite_idx ON grupo_gestao_convites(grupo_id, email) WHERE papel = 'cogestor';
CREATE TABLE grupo_agenda (
    id BIGSERIAL PRIMARY KEY,
    grupo_id BIGINT NOT NULL REFERENCES grupos(id) ON DELETE CASCADE,
    tipo VARCHAR(12) NOT NULL CHECK (tipo IN ('apresentacao', 'bloqueio')),
    titulo VARCHAR(120) NOT NULL DEFAULT '',
    inicio DATE NOT NULL,
    fim DATE NOT NULL CHECK (fim >= inicio),
    publico BOOLEAN NOT NULL DEFAULT false,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX grupo_agenda_periodo_idx ON grupo_agenda(grupo_id, inicio, fim);
