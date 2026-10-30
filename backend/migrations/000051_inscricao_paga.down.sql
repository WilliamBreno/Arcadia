UPDATE pagamentos SET devolucao_motivo = 'apos_reserva' WHERE devolucao_motivo = 'inscricao_encerrada';
ALTER TABLE pagamentos DROP CONSTRAINT pagamentos_devolucao_motivo_check;
ALTER TABLE pagamentos ADD CONSTRAINT pagamentos_devolucao_motivo_check
    CHECK (devolucao_motivo IN ('apos_reserva', 'duplicado', 'valor_divergente'));
UPDATE fichas_participacao SET status = 'pendente' WHERE status = 'aguardando_pagamento';
ALTER TABLE fichas_participacao DROP CONSTRAINT fichas_participacao_status_check;
ALTER TABLE fichas_participacao ADD CONSTRAINT fichas_participacao_status_check
    CHECK (status IN ('rascunho', 'pendente', 'aprovado', 'rejeitado', 'lista_espera', 'desistiu'));
ALTER TABLE fichas_participacao DROP COLUMN prazo_pagamento_em;
DROP INDEX IF EXISTS itens_pedido_ficha_idx;
ALTER TABLE itens_pedido DROP COLUMN ficha_id;
DROP INDEX IF EXISTS tipos_ingresso_inscricao_talento_un;
ALTER TABLE tipos_ingresso DROP COLUMN inscricao_talento;
ALTER TABLE eventos DROP COLUMN inscricao_talento_centavos;
