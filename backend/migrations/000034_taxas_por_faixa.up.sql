-- Item 5.1: taxa fixa por faixa de preço, garantia percentual com mínimo
-- e teto, e opção de o organizador absorver a taxa (seção 2.1).
INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('TAXA_FAIXAS', '3000:99,8000:199,20000:399,*:699', 'Taxa da plataforma por faixa de preço do item (limite:taxa em centavos; * = acima). Preço zero não paga taxa.'),
    ('GARANTIA_PERCENTUAL_BP', '500', 'Garantia de vaga em pontos-base do preço do item (500 = 5%).'),
    ('GARANTIA_MIN_CENTAVOS', '199', 'Valor mínimo da garantia de vaga, em centavos.'),
    ('GARANTIA_MAX_CENTAVOS', '1490', 'Valor máximo (teto) da garantia de vaga, em centavos.')
ON CONFLICT (chave) DO NOTHING;

DELETE FROM config_plataforma WHERE chave IN ('TAXA_INGRESSO_CENTAVOS', 'TAXA_CADASTRO_CENTAVOS', 'GARANTIA_CENTAVOS');

ALTER TABLE eventos ADD COLUMN taxa_paga_por VARCHAR(20) NOT NULL DEFAULT 'comprador'
    CHECK (taxa_paga_por IN ('comprador', 'organizador'));

ALTER TABLE itens_pedido ADD COLUMN taxa_absorvida BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE repasses ADD COLUMN taxa_absorvida_centavos BIGINT NOT NULL DEFAULT 0;

ALTER TABLE lancamentos DROP CONSTRAINT IF EXISTS lancamentos_tipo_check;
ALTER TABLE lancamentos ADD CONSTRAINT lancamentos_tipo_check CHECK (tipo IN (
    'venda_preco', 'taxa_plataforma', 'garantia', 'taxa_processador',
    'reembolso_preco', 'reembolso_taxa', 'reembolso_garantia',
    'custo_processador_perdido', 'repasse',
    'taxa_absorvida', 'estorno_taxa_absorvida'
));
