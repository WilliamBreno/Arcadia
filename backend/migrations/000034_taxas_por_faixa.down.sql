DELETE FROM lancamentos WHERE tipo IN ('taxa_absorvida', 'estorno_taxa_absorvida');
ALTER TABLE lancamentos DROP CONSTRAINT IF EXISTS lancamentos_tipo_check;
ALTER TABLE lancamentos ADD CONSTRAINT lancamentos_tipo_check CHECK (tipo IN (
    'venda_preco', 'taxa_plataforma', 'garantia', 'taxa_processador',
    'reembolso_preco', 'reembolso_taxa', 'reembolso_garantia',
    'custo_processador_perdido', 'repasse'
));

ALTER TABLE repasses DROP COLUMN taxa_absorvida_centavos;
ALTER TABLE itens_pedido DROP COLUMN taxa_absorvida;
ALTER TABLE eventos DROP COLUMN taxa_paga_por;

DELETE FROM config_plataforma WHERE chave IN ('TAXA_FAIXAS', 'GARANTIA_PERCENTUAL_BP', 'GARANTIA_MIN_CENTAVOS', 'GARANTIA_MAX_CENTAVOS');
INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('TAXA_INGRESSO_CENTAVOS', '99', 'Taxa fixa da plataforma por ingresso vendido, em centavos.'),
    ('TAXA_CADASTRO_CENTAVOS', '49', 'Taxa fixa da plataforma por cadastro em evento gratuito, em centavos.'),
    ('GARANTIA_CENTAVOS', '199', 'Garantia de vaga por item, em centavos.')
ON CONFLICT (chave) DO NOTHING;
