DROP TABLE IF EXISTS comissoes_pagas;
ALTER TABLE itens_pedido DROP COLUMN IF EXISTS comissao_afiliado_centavos;
ALTER TABLE afiliados
    DROP COLUMN IF EXISTS painel_token_hash,
    DROP COLUMN IF EXISTS comissao_fixa_centavos,
    DROP COLUMN IF EXISTS comissao_bp,
    DROP COLUMN IF EXISTS comissao_tipo;
