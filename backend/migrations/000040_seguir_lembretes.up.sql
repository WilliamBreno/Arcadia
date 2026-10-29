-- Item 6.4: seguir organizador/categoria, aviso de evento novo e lembretes.

-- tipo organizador: alvo = id do organizador. tipo categoria: alvo = slug
-- da categoria, ou "slug@cidade" para só uma cidade (ex.: danca@maceio-al).
CREATE TABLE seguindo (
    id         BIGSERIAL PRIMARY KEY,
    usuario_id BIGINT       NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
    tipo       VARCHAR(20)  NOT NULL CHECK (tipo IN ('organizador', 'categoria')),
    alvo       VARCHAR(160) NOT NULL,
    rotulo     VARCHAR(200) NOT NULL DEFAULT '',
    criado_em  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (usuario_id, tipo, alvo)
);
CREATE INDEX seguindo_alvo_idx ON seguindo (tipo, alvo);

-- A pessoa pode desligar todos os avisos de novidades (link no e-mail).
ALTER TABLE usuarios ADD COLUMN receber_novidades BOOLEAN NOT NULL DEFAULT true;

-- Cada evento avisa os seguidores uma vez só.
CREATE TABLE avisos_publicacao (
    evento_id  BIGINT      PRIMARY KEY REFERENCES eventos (id) ON DELETE CASCADE,
    enviados   INT         NOT NULL DEFAULT 0,
    enviado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lembrete enviado (7 dias e 1 dia antes), uma vez por e-mail e evento.
CREATE TABLE lembretes_enviados (
    evento_id  BIGINT       NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    email      VARCHAR(320) NOT NULL,
    tipo       VARCHAR(10)  NOT NULL CHECK (tipo IN ('7d', '1d')),
    enviado_em TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (evento_id, email, tipo)
);
