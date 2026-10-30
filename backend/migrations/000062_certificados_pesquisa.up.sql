-- Item 9.7: certificados automáticos e pesquisa pós-evento (NPS).

-- Configuração por evento (desligado por padrão).
CREATE TABLE config_certificado (
    evento_id         BIGINT PRIMARY KEY REFERENCES eventos(id) ON DELETE CASCADE,
    ativo             BOOLEAN NOT NULL DEFAULT false,
    publico           BOOLEAN NOT NULL DEFAULT true,
    participantes     BOOLEAN NOT NULL DEFAULT false,
    jurados           BOOLEAN NOT NULL DEFAULT false,
    equipe            BOOLEAN NOT NULL DEFAULT false,
    carga_horaria_min INT NOT NULL DEFAULT 0 CHECK (carga_horaria_min >= 0 AND carga_horaria_min <= 60000),
    texto             VARCHAR(400) NOT NULL DEFAULT '',
    assinatura_nome   VARCHAR(120) NOT NULL DEFAULT '',
    assinatura_cargo  VARCHAR(120) NOT NULL DEFAULT '',
    atualizado_em     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Um certificado por ingresso (público) ou por papel (participante,
-- jurado, equipe). O código é público: quem tem o código valida.
CREATE TABLE certificados (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    funcao         VARCHAR(14) NOT NULL CHECK (funcao IN ('publico', 'participante', 'jurado', 'equipe')),
    item_id        BIGINT UNIQUE REFERENCES itens_pedido(id) ON DELETE SET NULL,
    ficha_id       BIGINT UNIQUE REFERENCES fichas_participacao(id) ON DELETE SET NULL,
    papel_id       BIGINT UNIQUE REFERENCES papeis_evento(id) ON DELETE SET NULL,
    usuario_id     BIGINT REFERENCES usuarios(id) ON DELETE SET NULL,
    email          VARCHAR(200) NOT NULL DEFAULT '',
    nome           VARCHAR(160) NOT NULL,
    nome_corrigido BOOLEAN NOT NULL DEFAULT false,
    detalhe        VARCHAR(160) NOT NULL DEFAULT '',
    codigo         VARCHAR(16) NOT NULL UNIQUE,
    emitido_em     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX certificados_evento ON certificados (evento_id);
CREATE INDEX certificados_email ON certificados (lower(email));
CREATE INDEX certificados_usuario ON certificados (usuario_id);

-- Pesquisa: anônima para o organizador. respondente = hash do e-mail
-- (só para não responder duas vezes).
CREATE TABLE respostas_pesquisa (
    id          BIGSERIAL PRIMARY KEY,
    evento_id   BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    respondente VARCHAR(64) NOT NULL,
    nota        SMALLINT NOT NULL CHECK (nota BETWEEN 0 AND 10),
    comentario  VARCHAR(1000) NOT NULL DEFAULT '',
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (evento_id, respondente)
);

-- Pós-evento já processado (certificados emitidos e e-mails enviados).
CREATE TABLE pos_evento_enviado (
    evento_id  BIGINT PRIMARY KEY REFERENCES eventos(id) ON DELETE CASCADE,
    enviado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
    emails     INT NOT NULL DEFAULT 0
);
