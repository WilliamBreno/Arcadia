CREATE TABLE repasses (
    id                        BIGSERIAL PRIMARY KEY,
    evento_id                 BIGINT      NOT NULL REFERENCES eventos (id),
    organizador_id            BIGINT      NOT NULL REFERENCES organizadores (id),
    valor_bruto_centavos      BIGINT      NOT NULL,
    taxa_processador_centavos BIGINT      NOT NULL,
    valor_liquido_centavos    BIGINT      NOT NULL,
    status                    VARCHAR(20) NOT NULL DEFAULT 'pendente'
        CHECK (status IN ('calculado', 'pendente', 'pago', 'cancelado')),
    liberar_em                TIMESTAMPTZ NOT NULL,
    pago_em                   TIMESTAMPTZ,
    comprovante_url           TEXT        NOT NULL DEFAULT '',
    observacao                TEXT        NOT NULL DEFAULT '',
    criado_em                 TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX repasses_evento_ativo_idx ON repasses (evento_id) WHERE status <> 'cancelado';
CREATE INDEX repasses_organizador_id_idx ON repasses (organizador_id);
CREATE INDEX repasses_status_idx ON repasses (status);
