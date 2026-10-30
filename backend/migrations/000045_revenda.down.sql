ALTER TABLE reembolsos DROP CONSTRAINT IF EXISTS reembolsos_tipo_check;
ALTER TABLE reembolsos ADD CONSTRAINT reembolsos_tipo_check
    CHECK (tipo IN ('garantia', 'arrependimento', 'evento_cancelado', 'manual'));
DROP TABLE IF EXISTS revendas;
