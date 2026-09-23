CREATE TABLE regras_regionais (
    id                     BIGSERIAL PRIMARY KEY,
    uf                     CHAR(2)      NOT NULL,
    municipio              VARCHAR(255),
    permite_taxa           BOOLEAN      NOT NULL DEFAULT true,
    taxa_maxima_percentual NUMERIC(5, 2),
    exige_canal_sem_taxa   BOOLEAN      NOT NULL DEFAULT false,
    excecao_publico_ate    INT,
    observacao             TEXT         NOT NULL DEFAULT '',
    fonte                  TEXT         NOT NULL DEFAULT '',
    vigente_desde          DATE,
    criado_em              TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Um estado pode ter regra geral (municipio nulo) e cidades com regra
-- própria (Fortaleza/CE) — a busca tenta a específica antes da geral.
CREATE UNIQUE INDEX regras_regionais_uf_municipio_idx ON regras_regionais (uf, COALESCE(municipio, ''));

-- Seed: levantamento inicial da seção 7.8 do plano — A VALIDAR COM
-- ADVOGADO antes de confiar 100% nisso em produção.
INSERT INTO regras_regionais (uf, municipio, permite_taxa, taxa_maxima_percentual, exige_canal_sem_taxa, excecao_publico_ate, observacao, fonte) VALUES
    ('AC', NULL, false, NULL, false, NULL, 'Proíbe taxa de conveniência em venda online de ingressos.', 'Levantamento inicial — plano seção 7.8'),
    ('RR', NULL, false, NULL, false, NULL, 'Proíbe taxa de conveniência em venda online de ingressos.', 'Levantamento inicial — plano seção 7.8'),
    ('ES', NULL, false, NULL, true, 200, 'Proíbe taxa de conveniência, exceto se houver canal alternativo sem taxa; eventos até 200 pessoas dispensados.', 'Levantamento inicial — plano seção 7.8'),
    ('AL', NULL, true, 10.00, false, NULL, 'Taxa de conveniência limitada a 10% do valor do ingresso.', 'Levantamento inicial — plano seção 7.8'),
    ('CE', 'Fortaleza', true, NULL, true, NULL, 'Exige ao menos um canal de venda sem taxa de conveniência.', 'Levantamento inicial — plano seção 7.8')
ON CONFLICT DO NOTHING;
