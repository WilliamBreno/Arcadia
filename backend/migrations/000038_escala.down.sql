ALTER TABLE reembolsos DROP COLUMN ultimo_erro;
ALTER TABLE reembolsos DROP COLUMN proxima_tentativa_em;
ALTER TABLE reembolsos DROP COLUMN tentativas;
DROP TABLE emails_fila;
DROP TABLE limites_taxa;
