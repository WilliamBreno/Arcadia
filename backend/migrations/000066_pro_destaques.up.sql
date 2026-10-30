-- Item 10.4: Plano Pro do organizador e destaque pago na vitrine. O
-- dinheiro é da Evve (não passa por pedido, repasse ou ledger do evento).

INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('PRO_MENSAL_CENTAVOS', '4900', 'Plano Pro: 30 dias (item 10.4)'),
    ('PRO_ANUAL_CENTAVOS', '49000', 'Plano Pro: 12 meses (item 10.4)'),
    ('DESTAQUE_DIA_CENTAVOS', '1500', 'Destaque na vitrine: preço por dia (item 10.4)'),
    ('DESTAQUE_VAGAS_DIA', '3', 'Destaque na vitrine: eventos por cidade/categoria e dia (item 10.4)')
ON CONFLICT (chave) DO NOTHING;

-- Cobrança da plataforma (Pro ou destaque), paga por Pix ou cartão no
-- Mercado Pago com referência "PLAT-<id>".
CREATE TABLE cobrancas_plataforma (
    id             BIGSERIAL PRIMARY KEY,
    organizador_id BIGINT NOT NULL REFERENCES organizadores(id) ON DELETE CASCADE,
    usuario_id     BIGINT NOT NULL REFERENCES usuarios(id),
    tipo           VARCHAR(12) NOT NULL CHECK (tipo IN ('pro_mensal', 'pro_anual', 'destaque')),
    descricao      VARCHAR(200) NOT NULL,
    valor_centavos BIGINT NOT NULL CHECK (valor_centavos >= 0),
    status         VARCHAR(12) NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'paga', 'devolvida')),
    mp_payment_id  VARCHAR(40) UNIQUE,
    pix_mp_payment_id VARCHAR(40),
    pix_copia_cola TEXT,
    pix_expira_em  TIMESTAMPTZ,
    metodo         VARCHAR(10) NOT NULL DEFAULT '',
    taxa_processador_centavos BIGINT NOT NULL DEFAULT 0,
    reembolsado_centavos BIGINT NOT NULL DEFAULT 0,
    expira_em      TIMESTAMPTZ NOT NULL,
    pago_em        TIMESTAMPTZ,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX cobrancas_plataforma_org ON cobrancas_plataforma (organizador_id, criado_em DESC);

-- Períodos do Pro (somam-se: renovar antes de vencer emenda no fim).
CREATE TABLE periodos_pro (
    id             BIGSERIAL PRIMARY KEY,
    organizador_id BIGINT NOT NULL REFERENCES organizadores(id) ON DELETE CASCADE,
    inicio_em      TIMESTAMPTZ NOT NULL,
    fim_em         TIMESTAMPTZ NOT NULL,
    cobranca_id    BIGINT UNIQUE REFERENCES cobrancas_plataforma(id),
    aviso_vencimento_em TIMESTAMPTZ,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (fim_em > inicio_em)
);
CREATE INDEX periodos_pro_org ON periodos_pro (organizador_id, fim_em DESC);

-- Destaque: um evento num lugar (cidade ou categoria) num dia.
CREATE TABLE destaques (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    organizador_id BIGINT NOT NULL REFERENCES organizadores(id) ON DELETE CASCADE,
    alvo_tipo      VARCHAR(10) NOT NULL CHECK (alvo_tipo IN ('cidade', 'categoria')),
    alvo_valor     VARCHAR(120) NOT NULL,
    dia            DATE NOT NULL,
    valor_centavos BIGINT NOT NULL DEFAULT 0,
    incluido_pro   BOOLEAN NOT NULL DEFAULT false,
    status         VARCHAR(12) NOT NULL DEFAULT 'reservado' CHECK (status IN ('reservado', 'pago', 'cancelado', 'devolvido')),
    cobranca_id    BIGINT REFERENCES cobrancas_plataforma(id),
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX destaques_alvo_dia ON destaques (alvo_tipo, alvo_valor, dia) WHERE status IN ('reservado', 'pago');
CREATE UNIQUE INDEX destaques_evento_alvo_dia ON destaques (evento_id, alvo_tipo, alvo_valor, dia) WHERE status IN ('reservado', 'pago');
