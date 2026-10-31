-- Item 11.7: coleções temáticas (estilo "Coleções" da Sympla). A coleção
-- ganha ícone, período em que aparece, regra automática opcional e pode ser
-- aberta para o organizador marcar o próprio evento (tema).
ALTER TABLE colecoes
    ADD COLUMN icone              VARCHAR(30) NOT NULL DEFAULT 'estrela',
    ADD COLUMN exibir_de          DATE,
    ADD COLUMN exibir_ate         DATE,
    ADD COLUMN regra              VARCHAR(20) NOT NULL DEFAULT 'manual'
        CHECK (regra IN ('manual', 'gratuitos', 'esta_semana', 'online', 'categoria')),
    ADD COLUMN regra_valor        VARCHAR(60) NOT NULL DEFAULT '',
    ADD COLUMN aberta_organizador BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN na_home            BOOLEAN     NOT NULL DEFAULT true,
    ADD CONSTRAINT colecoes_periodo_ck CHECK (exibir_de IS NULL OR exibir_ate IS NULL OR exibir_de <= exibir_ate);

-- Quem colocou o evento na coleção: o admin (lista montada à mão) ou o
-- organizador (tema). O admin pode tirar um evento marcado pelo
-- organizador; a linha fica "bloqueada" para ele não marcar de novo.
ALTER TABLE colecao_eventos
    ADD COLUMN origem    VARCHAR(12) NOT NULL DEFAULT 'admin' CHECK (origem IN ('admin', 'organizador')),
    ADD COLUMN bloqueado BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN criado_em TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX colecao_eventos_evento_idx ON colecao_eventos (evento_id);

-- Temas da época e coleções automáticas iniciais (o admin edita as datas).
INSERT INTO colecoes (slug, titulo, descricao, ordem, icone, exibir_de, exibir_ate, regra, aberta_organizador, na_home) VALUES
    ('halloween', 'Halloween', 'Festas à fantasia, terror e noites temáticas.', 10, 'abobora', '2026-10-01', '2026-11-02', 'manual', true, true),
    ('natal', 'Natal', 'Cantatas, feiras e encontros de fim de ano.', 20, 'arvore', '2026-11-20', '2026-12-25', 'manual', true, true),
    ('reveillon-2027', 'Réveillon 2027', 'Para virar o ano em grande estilo.', 30, 'tacas', '2026-11-15', '2027-01-01', 'manual', true, true),
    ('ferias', 'Férias', 'Programação para curtir as férias.', 40, 'sol', '2026-12-10', '2027-02-28', 'manual', true, true),
    ('carnaval', 'Carnaval', 'Blocos, bailes e prévias.', 50, 'mascara', '2027-01-05', '2027-02-10', 'manual', true, true),
    ('festa-junina', 'Festa Junina', 'Arraiás, quadrilhas e forró.', 60, 'fogueira', '2027-05-15', '2027-07-31', 'manual', true, true),
    ('dia-das-criancas', 'Dia das Crianças', 'Programação para a criançada.', 70, 'pipa', '2027-09-15', '2027-10-12', 'manual', true, true),
    ('semana-geek', 'Semana Geek', 'Animes, games, cosplay e cultura pop.', 80, 'controle', NULL, NULL, 'manual', true, true),
    ('gratuitos', 'Gratuitos', 'Eventos com entrada gratuita.', 100, 'ingresso', NULL, NULL, 'gratuitos', false, false),
    ('esta-semana', 'Esta semana', 'O que acontece nos próximos 7 dias.', 110, 'calendario', NULL, NULL, 'esta_semana', false, false),
    ('online', 'Eventos online', 'Para assistir de onde estiver.', 120, 'tela', NULL, NULL, 'online', false, false)
-- Se a coleção já existe (por exemplo, a migração foi desfeita e refeita),
-- só recebe a configuração do tema se ainda estiver com os valores padrão.
ON CONFLICT (slug) DO UPDATE SET icone = EXCLUDED.icone, exibir_de = EXCLUDED.exibir_de, exibir_ate = EXCLUDED.exibir_ate,
    regra = EXCLUDED.regra, aberta_organizador = EXCLUDED.aberta_organizador, na_home = EXCLUDED.na_home
WHERE colecoes.icone = 'estrela' AND colecoes.regra = 'manual' AND colecoes.exibir_de IS NULL AND colecoes.exibir_ate IS NULL
    AND NOT colecoes.aberta_organizador;
