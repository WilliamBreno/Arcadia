CREATE TABLE config_plataforma (
    chave         VARCHAR(100) PRIMARY KEY,
    valor         VARCHAR(255) NOT NULL,
    descricao     TEXT,
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed: valores padrão vindos da seção 2.1 e 7.6 do plano-eventos.md.
-- Tabela usa chave textual como PK (sem bigserial), então não há
-- sequence para ajustar com setval().
INSERT INTO config_plataforma (chave, valor, descricao) VALUES
    ('TAXA_INGRESSO_CENTAVOS', '99',  'Taxa fixa da plataforma por ingresso vendido, em centavos.'),
    ('TAXA_CADASTRO_CENTAVOS', '49',  'Taxa fixa da plataforma por cadastro em evento gratuito, em centavos.'),
    ('GARANTIA_CENTAVOS',      '199', 'Preço da garantia de vaga (cancelamento flexível) por item, em centavos.'),
    ('REPASSE_DIAS_UTEIS',     '3',   'Dias úteis após o fim do evento para liberar o repasse ao organizador.'),
    ('RESERVA_MINUTOS',        '15',  'Minutos que um item fica reservado no checkout antes de expirar.')
ON CONFLICT (chave) DO NOTHING;
