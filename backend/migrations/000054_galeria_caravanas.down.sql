DROP TABLE IF EXISTS caravana_membros;
DROP TABLE IF EXISTS caravanas;
DROP TABLE IF EXISTS remocoes_foto;
DROP TABLE IF EXISTS marcacoes_foto;
DROP TABLE IF EXISTS fotos_evento;
UPDATE papeis_evento SET status = 'removido' WHERE papel = 'fotografo';
DELETE FROM papeis_evento WHERE papel = 'fotografo';
ALTER TABLE papeis_evento DROP CONSTRAINT papeis_evento_papel_check;
ALTER TABLE papeis_evento ADD CONSTRAINT papeis_evento_papel_check
    CHECK (papel IN ('organizador', 'jurado', 'participante', 'staff'));
