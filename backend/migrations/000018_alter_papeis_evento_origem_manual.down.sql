ALTER TABLE papeis_evento DROP CONSTRAINT papeis_evento_origem_check;
ALTER TABLE papeis_evento ADD CONSTRAINT papeis_evento_origem_check
    CHECK (origem IN ('convite', 'inscricao'));
