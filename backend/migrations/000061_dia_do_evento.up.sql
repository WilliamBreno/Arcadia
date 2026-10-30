-- Item 9.6: painel ao vivo, ingressos importados e credenciais impressas.

-- Portões que a equipe escolhe na tela de check-in.
CREATE TABLE portoes_evento (
    id        BIGSERIAL PRIMARY KEY,
    evento_id BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    nome      VARCHAR(40) NOT NULL,
    UNIQUE (evento_id, nome)
);

-- Registro de cada entrada liberada (online, offline sincronizada,
-- importado ou credencial), com o portão. Alimenta o painel ao vivo.
CREATE TABLE entradas_evento (
    id         BIGSERIAL PRIMARY KEY,
    evento_id  BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    origem     VARCHAR(12) NOT NULL CHECK (origem IN ('ingresso', 'importado', 'credencial', 'expositor')),
    ref_id     BIGINT NOT NULL,
    portao     VARCHAR(40) NOT NULL DEFAULT '',
    usuario_id BIGINT REFERENCES usuarios(id) ON DELETE SET NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX entradas_evento_tempo ON entradas_evento (evento_id, criado_em);

-- Ingressos vendidos em outra plataforma: só valem no check-in (sem
-- dinheiro, taxa, reembolso ou e-mail do Evve).
CREATE TABLE ingressos_importados (
    id           BIGSERIAL PRIMARY KEY,
    evento_id    BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    codigo       VARCHAR(200) NOT NULL,
    nome         VARCHAR(160) NOT NULL DEFAULT '',
    email        VARCHAR(200) NOT NULL DEFAULT '',
    tipo         VARCHAR(120) NOT NULL DEFAULT '',
    origem       VARCHAR(60)  NOT NULL DEFAULT '',
    importado_em TIMESTAMPTZ  NOT NULL DEFAULT now(),
    utilizado_em TIMESTAMPTZ,
    UNIQUE (evento_id, codigo)
);

-- Credencial impressa (equipe, jurados, imprensa, convidados). Equipe e
-- jurados vêm dos papéis do evento; imprensa e convidados, à mão.
CREATE TABLE credenciais_evento (
    id                  BIGSERIAL PRIMARY KEY,
    evento_id           BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    nome                VARCHAR(120) NOT NULL,
    funcao              VARCHAR(20) NOT NULL CHECK (funcao IN ('equipe', 'jurado', 'imprensa', 'convidado')),
    papel_id            BIGINT UNIQUE REFERENCES papeis_evento(id) ON DELETE SET NULL,
    ativa               BOOLEAN NOT NULL DEFAULT true,
    primeira_entrada_em TIMESTAMPTZ,
    ultima_entrada_em   TIMESTAMPTZ,
    criado_em           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX credenciais_evento_idx ON credenciais_evento (evento_id);
