-- Item 8.4: inscrição paga de talento. O participante paga depois de
-- aprovado (48 h), pelo mesmo pedido/pagamento dos ingressos, com a mesma
-- taxa e a mesma garantia opcional.
ALTER TABLE eventos
    ADD COLUMN inscricao_talento_centavos BIGINT NOT NULL DEFAULT 0
        CHECK (inscricao_talento_centavos >= 0);

-- Um "ingresso interno" por evento, inativo: nunca aparece na vitrine nem
-- é vendido pelo checkout comum; só segura os itens de inscrição.
ALTER TABLE tipos_ingresso
    ADD COLUMN inscricao_talento BOOLEAN NOT NULL DEFAULT false;
CREATE UNIQUE INDEX tipos_ingresso_inscricao_talento_un
    ON tipos_ingresso (evento_id) WHERE inscricao_talento;

ALTER TABLE itens_pedido
    ADD COLUMN ficha_id BIGINT REFERENCES fichas_participacao (id);
CREATE INDEX itens_pedido_ficha_idx ON itens_pedido (ficha_id) WHERE ficha_id IS NOT NULL;

ALTER TABLE fichas_participacao
    ADD COLUMN prazo_pagamento_em TIMESTAMPTZ;
ALTER TABLE fichas_participacao DROP CONSTRAINT fichas_participacao_status_check;
ALTER TABLE fichas_participacao ADD CONSTRAINT fichas_participacao_status_check
    CHECK (status IN ('rascunho', 'pendente', 'aprovado', 'rejeitado', 'lista_espera', 'desistiu', 'aguardando_pagamento'));

-- Pagamento que chega depois que a inscrição deixou de esperar por ele
-- (prazo vencido, organizador removeu, pessoa desistiu) volta inteiro.
ALTER TABLE pagamentos DROP CONSTRAINT pagamentos_devolucao_motivo_check;
ALTER TABLE pagamentos ADD CONSTRAINT pagamentos_devolucao_motivo_check
    CHECK (devolucao_motivo IN ('apos_reserva', 'duplicado', 'valor_divergente', 'inscricao_encerrada'));
