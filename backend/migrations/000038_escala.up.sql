-- Item 5.5: preparo para escala.

-- Limite de tentativas compartilhado entre instâncias da API (antes ficava
-- na memória de cada processo). Janela fixa: (chave, início da janela).
CREATE UNLOGGED TABLE limites_taxa (
    chave     VARCHAR(200) NOT NULL,
    janela    BIGINT       NOT NULL,
    contagem  INT          NOT NULL DEFAULT 0,
    PRIMARY KEY (chave, janela)
);

-- Fila de e-mails: envio com novas tentativas em vez de perder o e-mail
-- quando o provedor falha.
CREATE TABLE emails_fila (
    id                   BIGSERIAL PRIMARY KEY,
    destinatario         VARCHAR(320) NOT NULL,
    assunto              TEXT         NOT NULL,
    corpo_html           TEXT         NOT NULL,
    imagens              JSONB        NOT NULL DEFAULT '[]',
    status               VARCHAR(20)  NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'enviado', 'falhou')),
    tentativas           INT          NOT NULL DEFAULT 0,
    ultimo_erro          TEXT         NOT NULL DEFAULT '',
    proxima_tentativa_em TIMESTAMPTZ  NOT NULL DEFAULT now(),
    criado_em            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    enviado_em           TIMESTAMPTZ
);
CREATE INDEX emails_fila_pendentes_idx ON emails_fila (proxima_tentativa_em) WHERE status = 'pendente';

-- Reembolsos que falham são tentados de novo com intervalo crescente e
-- param depois de um limite (antes: sem limite de tentativas).
ALTER TABLE reembolsos ADD COLUMN tentativas INT NOT NULL DEFAULT 0;
ALTER TABLE reembolsos ADD COLUMN proxima_tentativa_em TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE reembolsos ADD COLUMN ultimo_erro TEXT NOT NULL DEFAULT '';
