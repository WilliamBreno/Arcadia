-- Item 8.7: galeria oficial do evento e caravanas. Decisões do dono
-- (30/10/2026): organizador e fotógrafos convidados sobem fotos (até 500
-- por evento, 10 MB cada, com crédito); só a própria pessoa adulta se marca
-- e a foto marcada vai para o perfil de talento; qualquer um pede "não quero
-- aparecer" e a foto some na hora até o organizador revisar. Caravana: cada
-- membro compra o próprio ingresso; link fechado; o responsável vê nome e
-- telefone (o membro aceita ao entrar); o organizador vê só os números.
ALTER TABLE papeis_evento DROP CONSTRAINT papeis_evento_papel_check;
ALTER TABLE papeis_evento ADD CONSTRAINT papeis_evento_papel_check
    CHECK (papel IN ('organizador', 'jurado', 'participante', 'staff', 'fotografo'));

CREATE TABLE fotos_evento (
    id          BIGSERIAL PRIMARY KEY,
    evento_id   BIGINT       NOT NULL REFERENCES eventos (id),
    enviada_por BIGINT       NOT NULL REFERENCES usuarios (id),
    url         VARCHAR(500) NOT NULL,
    oculta      BOOLEAN      NOT NULL DEFAULT false,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX fotos_evento_evento_idx ON fotos_evento (evento_id, id);

CREATE TABLE marcacoes_foto (
    id         BIGSERIAL PRIMARY KEY,
    foto_id    BIGINT      NOT NULL REFERENCES fotos_evento (id) ON DELETE CASCADE,
    usuario_id BIGINT      NOT NULL REFERENCES usuarios (id),
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (foto_id, usuario_id)
);
CREATE INDEX marcacoes_foto_usuario_idx ON marcacoes_foto (usuario_id);

CREATE TABLE remocoes_foto (
    id         BIGSERIAL PRIMARY KEY,
    foto_id    BIGINT      NOT NULL REFERENCES fotos_evento (id) ON DELETE CASCADE,
    usuario_id BIGINT REFERENCES usuarios (id),
    motivo     TEXT        NOT NULL DEFAULT '',
    status     VARCHAR(10) NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'mantida', 'removida')),
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX remocoes_foto_pendentes_idx ON remocoes_foto (foto_id) WHERE status = 'pendente';

CREATE TABLE caravanas (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT       NOT NULL REFERENCES eventos (id),
    responsavel_id BIGINT       NOT NULL REFERENCES usuarios (id),
    codigo         VARCHAR(16)  NOT NULL UNIQUE,
    nome           VARCHAR(80)  NOT NULL,
    cidade         VARCHAR(80)  NOT NULL,
    uf             VARCHAR(2)   NOT NULL DEFAULT '',
    ponto_embarque VARCHAR(200) NOT NULL,
    saida_em       TIMESTAMPTZ,
    vagas          INT CHECK (vagas IS NULL OR vagas BETWEEN 1 AND 500),
    telefone       VARCHAR(30)  NOT NULL DEFAULT '',
    ativa          BOOLEAN      NOT NULL DEFAULT true,
    criado_em      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX caravanas_evento_idx ON caravanas (evento_id);

CREATE TABLE caravana_membros (
    id          BIGSERIAL PRIMARY KEY,
    caravana_id BIGINT       NOT NULL REFERENCES caravanas (id) ON DELETE CASCADE,
    usuario_id  BIGINT       NOT NULL REFERENCES usuarios (id),
    nome        VARCHAR(120) NOT NULL,
    telefone    VARCHAR(30)  NOT NULL DEFAULT '',
    embarcou_em TIMESTAMPTZ,
    criado_em   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (caravana_id, usuario_id)
);
