DROP INDEX IF EXISTS repasses_envio_referencia_un;
ALTER TABLE repasses
    DROP COLUMN IF EXISTS envio_aprovado_por,
    DROP COLUMN IF EXISTS envio_proxima_em,
    DROP COLUMN IF EXISTS mp_payout_id,
    DROP COLUMN IF EXISTS envio_referencia,
    DROP COLUMN IF EXISTS envio_tentativa,
    DROP COLUMN IF EXISTS envio_motivo,
    DROP COLUMN IF EXISTS envio_status;
DELETE FROM config_plataforma WHERE chave IN ('REPASSE_DIAS_UTEIS_VERIFICADO', 'REPASSE_AUTO_LIMITE_CENTAVOS');
