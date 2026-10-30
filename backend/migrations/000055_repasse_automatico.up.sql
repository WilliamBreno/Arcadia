-- Item 9.1: repasse automático por Pix (Mercado Pago Payouts). Decisões do
-- dono (30/10/2026): organizador verificado recebe 1 dia útil após o fim do
-- evento, os demais 3; o sistema envia sozinho até R$ 20 mil, sem
-- reembolso falhado, sem contestação em disputa e com chave Pix CPF/CNPJ
-- igual ao documento; o resto fica para o admin aprovar; a tarifa do envio
-- (se houver) é custo da Evve. Repasses que já existiam ficam 'manual' (o
-- admin pode ter pago por fora); só os novos, gerados com o envio ligado, vão
-- para a fila ('na_fila').
INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('REPASSE_DIAS_UTEIS_VERIFICADO', '1', 'Dias úteis após o fim do evento para repassar a organizador verificado (item 9.1)'),
    ('REPASSE_AUTO_LIMITE_CENTAVOS', '2000000', 'Acima disso o repasse automático espera o admin aprovar (item 9.1)')
ON CONFLICT (chave) DO NOTHING;

ALTER TABLE repasses
    ADD COLUMN envio_status VARCHAR(20) NOT NULL DEFAULT 'manual'
        CHECK (envio_status IN ('manual', 'na_fila', 'aguardando_aprovacao', 'enviando', 'processando', 'enviado', 'falhou')),
    ADD COLUMN envio_motivo TEXT NOT NULL DEFAULT '',
    ADD COLUMN envio_tentativa INT NOT NULL DEFAULT 0,
    ADD COLUMN envio_referencia VARCHAR(50),
    ADD COLUMN mp_payout_id VARCHAR(60),
    ADD COLUMN envio_proxima_em TIMESTAMPTZ,
    ADD COLUMN envio_aprovado_por BIGINT REFERENCES usuarios (id);
CREATE UNIQUE INDEX repasses_envio_referencia_un ON repasses (envio_referencia) WHERE envio_referencia IS NOT NULL;
