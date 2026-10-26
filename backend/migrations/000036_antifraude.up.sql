-- Item 5.3: antifraude e contestação (chargeback).

-- Limite de ingressos por conta em cada evento (0 = sem limite).
ALTER TABLE eventos ADD COLUMN max_ingressos_por_conta INT NOT NULL DEFAULT 10
    CHECK (max_ingressos_por_conta >= 0);

-- Quem contestou uma compra no cartão não paga mais com cartão (Pix segue liberado).
ALTER TABLE usuarios ADD COLUMN cartao_bloqueado_em TIMESTAMPTZ;

CREATE TABLE contestacoes (
    id                   BIGSERIAL PRIMARY KEY,
    pagamento_id         BIGINT      NOT NULL UNIQUE REFERENCES pagamentos (id),
    pedido_id            BIGINT      NOT NULL REFERENCES pedidos (id),
    evento_id            BIGINT      NOT NULL REFERENCES eventos (id),
    usuario_id           BIGINT      NOT NULL REFERENCES usuarios (id),
    valor_centavos       BIGINT      NOT NULL,
    -- em_disputa: comprador abriu a contestação; perdida: o dinheiro foi
    -- devolvido a ele pelo banco; resolvida: a disputa terminou a nosso favor.
    status               VARCHAR(20) NOT NULL CHECK (status IN ('em_disputa', 'perdida', 'resolvida')),
    status_processador   VARCHAR(40) NOT NULL DEFAULT '',
    criado_em            TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX contestacoes_status_idx ON contestacoes (status);

ALTER TABLE lancamentos DROP CONSTRAINT IF EXISTS lancamentos_tipo_check;
ALTER TABLE lancamentos ADD CONSTRAINT lancamentos_tipo_check CHECK (tipo IN (
    'venda_preco', 'taxa_plataforma', 'garantia', 'taxa_processador',
    'reembolso_preco', 'reembolso_taxa', 'reembolso_garantia',
    'custo_processador_perdido', 'repasse',
    'taxa_absorvida', 'estorno_taxa_absorvida',
    'chargeback'
));
