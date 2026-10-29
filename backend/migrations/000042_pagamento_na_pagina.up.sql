-- Item 7.1: pagamento dentro da página do Evve.
-- Pix gerado direto na API do Mercado Pago: um QR por pedido, reaproveitado
-- enquanto vale. O QR vale 30 minutos (mínimo do Mercado Pago) e a reserva
-- do pedido passa a terminar junto com ele.
ALTER TABLE pedidos
    ADD COLUMN pix_mp_payment_id VARCHAR(50),
    ADD COLUMN pix_copia_cola    TEXT,
    ADD COLUMN pix_expira_em     TIMESTAMPTZ;

-- Pagamento que não virou ingresso e precisa voltar inteiro ao comprador:
-- chegou depois da reserva sem vaga para reemitir, ou pagou duas vezes o
-- mesmo pedido, ou o valor não bate. devolvido_em vazio = estorno pendente
-- (o job tenta de novo até o Mercado Pago confirmar).
ALTER TABLE pagamentos
    ADD COLUMN devolucao_motivo VARCHAR(30)
        CHECK (devolucao_motivo IN ('apos_reserva', 'duplicado', 'valor_divergente')),
    ADD COLUMN devolucao_mp_id  VARCHAR(50),
    ADD COLUMN devolvido_em     TIMESTAMPTZ;

CREATE INDEX pagamentos_devolucao_pendente_idx ON pagamentos (id)
    WHERE devolucao_motivo IS NOT NULL AND devolvido_em IS NULL;
