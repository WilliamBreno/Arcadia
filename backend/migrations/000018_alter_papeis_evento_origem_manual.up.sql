-- Staff é adicionado diretamente pelo organizador (seção 3: "convite do
-- organizador", mas sem o fluxo público de token — é uma atribuição
-- direta por e-mail). Nem 'convite' nem 'inscricao' descrevem isso bem,
-- então a origem 'manual' cobre esse caso (e serve também pra qualquer
-- atribuição futura feita direto por um admin/organizador).
ALTER TABLE papeis_evento DROP CONSTRAINT papeis_evento_origem_check;
ALTER TABLE papeis_evento ADD CONSTRAINT papeis_evento_origem_check
    CHECK (origem IN ('convite', 'inscricao', 'manual'));
