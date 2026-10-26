DELETE FROM lancamentos WHERE tipo = 'chargeback';
ALTER TABLE lancamentos DROP CONSTRAINT IF EXISTS lancamentos_tipo_check;
ALTER TABLE lancamentos ADD CONSTRAINT lancamentos_tipo_check CHECK (tipo IN (
    'venda_preco', 'taxa_plataforma', 'garantia', 'taxa_processador',
    'reembolso_preco', 'reembolso_taxa', 'reembolso_garantia',
    'custo_processador_perdido', 'repasse',
    'taxa_absorvida', 'estorno_taxa_absorvida'
));
DROP TABLE contestacoes;
ALTER TABLE usuarios DROP COLUMN cartao_bloqueado_em;
ALTER TABLE eventos DROP COLUMN max_ingressos_por_conta;
