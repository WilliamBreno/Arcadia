-- Item 5.4: verificação do organizador e reserva de risco no repasse.

-- Verificado = o admin conferiu o documento e que a chave Pix é do mesmo
-- titular. Mudar documento ou chave Pix tira a verificação.
ALTER TABLE organizadores ADD COLUMN verificado_em TIMESTAMPTZ;
ALTER TABLE organizadores ADD COLUMN verificado_por BIGINT REFERENCES usuarios (id);

-- Reserva de risco: parte do repasse de organizador novo, sobre as vendas
-- no cartão, segurada para cobrir contestações (chargeback).
--   nenhuma: não houve reserva; retida: segurada até reserva_liberar_em;
--   paga: o admin enviou o Pix da reserva; usada: o admin usou a reserva
--   para cobrir uma contestação perdida.
ALTER TABLE repasses ADD COLUMN reserva_centavos BIGINT NOT NULL DEFAULT 0 CHECK (reserva_centavos >= 0);
ALTER TABLE repasses ADD COLUMN reserva_liberar_em TIMESTAMPTZ;
ALTER TABLE repasses ADD COLUMN reserva_status VARCHAR(20) NOT NULL DEFAULT 'nenhuma'
    CHECK (reserva_status IN ('nenhuma', 'retida', 'paga', 'usada'));
ALTER TABLE repasses ADD COLUMN reserva_pago_em TIMESTAMPTZ;
ALTER TABLE repasses ADD COLUMN reserva_comprovante_url TEXT NOT NULL DEFAULT '';
ALTER TABLE repasses ADD COLUMN reserva_observacao TEXT NOT NULL DEFAULT '';

INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('RESERVA_RISCO_BP', '1000', 'Reserva de risco em pontos-base das vendas no cartão de organizador novo (1000 = 10%).'),
    ('RESERVA_RISCO_DIAS', '60', 'Dias corridos que a reserva de risco fica segurada depois da liberação do repasse.'),
    ('RESERVA_RISCO_EVENTOS_MIN', '3', 'Repasses pagos para o organizador verificado deixar de ser considerado novo.')
ON CONFLICT (chave) DO NOTHING;

ALTER TABLE lancamentos DROP CONSTRAINT IF EXISTS lancamentos_tipo_check;
ALTER TABLE lancamentos ADD CONSTRAINT lancamentos_tipo_check CHECK (tipo IN (
    'venda_preco', 'taxa_plataforma', 'garantia', 'taxa_processador',
    'reembolso_preco', 'reembolso_taxa', 'reembolso_garantia',
    'custo_processador_perdido', 'repasse',
    'taxa_absorvida', 'estorno_taxa_absorvida',
    'chargeback', 'reserva_usada'
));
