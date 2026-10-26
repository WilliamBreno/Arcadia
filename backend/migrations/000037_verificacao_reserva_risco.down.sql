DELETE FROM lancamentos WHERE tipo = 'reserva_usada';
ALTER TABLE lancamentos DROP CONSTRAINT IF EXISTS lancamentos_tipo_check;
ALTER TABLE lancamentos ADD CONSTRAINT lancamentos_tipo_check CHECK (tipo IN (
    'venda_preco', 'taxa_plataforma', 'garantia', 'taxa_processador',
    'reembolso_preco', 'reembolso_taxa', 'reembolso_garantia',
    'custo_processador_perdido', 'repasse',
    'taxa_absorvida', 'estorno_taxa_absorvida',
    'chargeback'
));
DELETE FROM config_plataforma WHERE chave IN ('RESERVA_RISCO_BP', 'RESERVA_RISCO_DIAS', 'RESERVA_RISCO_EVENTOS_MIN');
ALTER TABLE repasses DROP COLUMN reserva_observacao;
ALTER TABLE repasses DROP COLUMN reserva_comprovante_url;
ALTER TABLE repasses DROP COLUMN reserva_pago_em;
ALTER TABLE repasses DROP COLUMN reserva_status;
ALTER TABLE repasses DROP COLUMN reserva_liberar_em;
ALTER TABLE repasses DROP COLUMN reserva_centavos;
ALTER TABLE organizadores DROP COLUMN verificado_por;
ALTER TABLE organizadores DROP COLUMN verificado_em;
