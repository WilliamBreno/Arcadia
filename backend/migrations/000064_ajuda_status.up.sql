-- Item 9.9: central de ajuda (mensagens ao suporte) e página de status.

CREATE TABLE contatos_suporte (
    id         BIGSERIAL PRIMARY KEY,
    usuario_id BIGINT REFERENCES usuarios(id) ON DELETE SET NULL,
    nome       VARCHAR(120) NOT NULL,
    email      VARCHAR(200) NOT NULL,
    assunto    VARCHAR(30)  NOT NULL,
    pedido     VARCHAR(40)  NOT NULL DEFAULT '',
    mensagem   VARCHAR(4000) NOT NULL,
    status     VARCHAR(12)  NOT NULL DEFAULT 'aberto' CHECK (status IN ('aberto', 'respondido')),
    criado_em  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX contatos_suporte_status ON contatos_suporte (status, criado_em DESC);

-- Avisos de incidente publicados pelo admin (aparecem em /status e numa
-- faixa no site enquanto estiverem abertos).
CREATE TABLE avisos_status (
    id          BIGSERIAL PRIMARY KEY,
    titulo      VARCHAR(120) NOT NULL,
    mensagem    VARCHAR(1000) NOT NULL DEFAULT '',
    nivel       VARCHAR(10)  NOT NULL CHECK (nivel IN ('aviso', 'falha', 'manutencao')),
    componente  VARCHAR(20)  NOT NULL DEFAULT 'geral',
    inicio_em   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    fim_em      TIMESTAMPTZ,
    criado_por  BIGINT REFERENCES usuarios(id) ON DELETE SET NULL
);
CREATE INDEX avisos_status_abertos ON avisos_status (inicio_em DESC) WHERE fim_em IS NULL;
