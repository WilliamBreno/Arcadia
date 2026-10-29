DROP INDEX IF EXISTS pagamentos_devolucao_pendente_idx;
ALTER TABLE pagamentos
    DROP COLUMN IF EXISTS devolvido_em,
    DROP COLUMN IF EXISTS devolucao_mp_id,
    DROP COLUMN IF EXISTS devolucao_motivo;
ALTER TABLE pedidos
    DROP COLUMN IF EXISTS pix_expira_em,
    DROP COLUMN IF EXISTS pix_copia_cola,
    DROP COLUMN IF EXISTS pix_mp_payment_id;
