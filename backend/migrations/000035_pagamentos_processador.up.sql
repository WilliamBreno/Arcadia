-- Item 5.2: cada pagamento guarda o processador que o recebeu, para o
-- estorno ir ao lugar certo quando houver mais de um.
ALTER TABLE pagamentos ADD COLUMN processador VARCHAR(30) NOT NULL DEFAULT 'mercadopago';

-- Referência para estimar a economia do Pix quando o evento ainda não
-- teve venda no cartão (taxa do cartão no MP com liberação em 30 dias).
INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('TAXA_REFERENCIA_CARTAO_BP', '398', 'Taxa de referência do cartão em pontos-base (398 = 3,98%), só para estimar a economia com Pix no painel do organizador.')
ON CONFLICT (chave) DO NOTHING;
