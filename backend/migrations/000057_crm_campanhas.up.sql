-- Item 9.3: CRM do organizador. Decisões do dono (30/10/2026): só recebe
-- quem aceitou (segue o organizador, 6.4, inclusive quem marca na compra
-- "quero receber novidades"); cota de 5.000 e-mails por mês e 2 campanhas
-- por dia por organizador; exportação CSV compatível com Mailchimp/Brevo;
-- WhatsApp oficial fica para depois (só o link wa.me para mandar o ingresso).
CREATE TABLE campanhas_email (
    id                  BIGSERIAL PRIMARY KEY,
    organizador_id      BIGINT       NOT NULL REFERENCES organizadores (id),
    criado_por          BIGINT       NOT NULL REFERENCES usuarios (id),
    assunto             VARCHAR(120) NOT NULL,
    corpo_texto         TEXT         NOT NULL,
    segmento            VARCHAR(20)  NOT NULL CHECK (segmento IN ('todos', 'compraram', 'foram')),
    segmento_evento_id  BIGINT REFERENCES eventos (id),
    destinatarios       INT          NOT NULL DEFAULT 0,
    enviada_em          TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX campanhas_email_org_idx ON campanhas_email (organizador_id, enviada_em);
