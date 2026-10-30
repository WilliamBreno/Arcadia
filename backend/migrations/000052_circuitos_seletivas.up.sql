-- Item 8.5: circuitos com ranking por temporada e seletivas.
-- Decisões do dono (30/10/2026): qualquer organizador cria um circuito e
-- aceita os eventos de outros; pontos fixos do 1º ao 10º (25-18-15-12-10-
-- 8-6-4-2-1) contados quando o resultado é liberado; menor aparece só pelo
-- primeiro nome; classificado na seletiva entra aprovado na final (e paga
-- a inscrição, se houver).
CREATE TABLE circuitos (
    id               BIGSERIAL PRIMARY KEY,
    organizador_id   BIGINT       NOT NULL REFERENCES organizadores (id),
    nome             VARCHAR(120) NOT NULL,
    slug             VARCHAR(140) NOT NULL UNIQUE,
    descricao        TEXT         NOT NULL DEFAULT '',
    temporada_inicio DATE         NOT NULL,
    temporada_fim    DATE         NOT NULL CHECK (temporada_fim >= temporada_inicio),
    criado_em        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE circuito_eventos (
    id          BIGSERIAL PRIMARY KEY,
    circuito_id BIGINT      NOT NULL REFERENCES circuitos (id) ON DELETE CASCADE,
    evento_id   BIGINT      NOT NULL REFERENCES eventos (id),
    status      VARCHAR(10) NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'aceito', 'recusado')),
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    decidido_em TIMESTAMPTZ
);
-- Um evento está em no máximo um circuito (pedido aberto ou aceito).
CREATE UNIQUE INDEX circuito_eventos_evento_un ON circuito_eventos (evento_id) WHERE status <> 'recusado';
CREATE INDEX circuito_eventos_circuito_idx ON circuito_eventos (circuito_id, status);

-- Resultado do evento gravado quando o organizador libera (base dos
-- pontos do circuito e das seletivas). Liberar de novo regrava.
CREATE TABLE resultados_evento (
    id           BIGSERIAL PRIMARY KEY,
    evento_id    BIGINT       NOT NULL REFERENCES eventos (id),
    ficha_id     BIGINT       NOT NULL REFERENCES fichas_participacao (id),
    usuario_id   BIGINT       NOT NULL REFERENCES usuarios (id),
    tipo         VARCHAR(20)  NOT NULL,
    posicao      INT          NOT NULL CHECK (posicao > 0),
    nome_publico VARCHAR(120) NOT NULL,
    menor        BOOLEAN      NOT NULL DEFAULT false,
    criado_em    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (evento_id, ficha_id)
);
CREATE INDEX resultados_evento_usuario_idx ON resultados_evento (usuario_id);

-- Seletiva: o evento "seletiva" classifica os N primeiros de cada
-- categoria para o evento "final". Se os dois são do mesmo organizador,
-- já nasce aceita; senão o dono da seletiva aceita.
CREATE TABLE seletivas (
    id                 BIGSERIAL PRIMARY KEY,
    final_evento_id    BIGINT      NOT NULL REFERENCES eventos (id),
    seletiva_evento_id BIGINT      NOT NULL REFERENCES eventos (id),
    vagas              INT         NOT NULL CHECK (vagas BETWEEN 1 AND 10),
    status             VARCHAR(10) NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'aceita', 'recusada')),
    criado_em          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (final_evento_id <> seletiva_evento_id),
    UNIQUE (final_evento_id, seletiva_evento_id)
);
CREATE INDEX seletivas_seletiva_idx ON seletivas (seletiva_evento_id, status);

-- Quem chegou à final pela seletiva (selo para o organizador e o e-mail).
ALTER TABLE fichas_participacao
    ADD COLUMN classificada_por_evento_id BIGINT REFERENCES eventos (id);
