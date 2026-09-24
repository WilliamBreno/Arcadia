ALTER TABLE tipos_ingresso ADD COLUMN sessao_id BIGINT REFERENCES sessoes (id);
UPDATE tipos_ingresso t SET sessao_id = (SELECT MIN(sessao_id) FROM tipo_ingresso_sessoes s WHERE s.tipo_ingresso_id = t.id);
DROP TABLE IF EXISTS tipo_ingresso_sessoes;
