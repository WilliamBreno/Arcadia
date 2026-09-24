-- Um tipo de ingresso passa a poder valer para VÁRIAS sessões (dias):
-- sem linhas = todas as sessões; com linhas = só as listadas.
CREATE TABLE tipo_ingresso_sessoes (
    tipo_ingresso_id BIGINT NOT NULL REFERENCES tipos_ingresso (id) ON DELETE CASCADE,
    sessao_id        BIGINT NOT NULL REFERENCES sessoes (id),
    PRIMARY KEY (tipo_ingresso_id, sessao_id)
);

INSERT INTO tipo_ingresso_sessoes (tipo_ingresso_id, sessao_id)
SELECT id, sessao_id FROM tipos_ingresso WHERE sessao_id IS NOT NULL;

ALTER TABLE tipos_ingresso DROP COLUMN sessao_id;
