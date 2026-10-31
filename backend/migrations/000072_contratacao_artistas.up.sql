CREATE TABLE municipios_artistas (
 id INT PRIMARY KEY, nome VARCHAR(100) NOT NULL, uf VARCHAR(2) NOT NULL,
 latitude DOUBLE PRECISION, longitude DOUBLE PRECISION
);
CREATE INDEX municipios_artistas_uf_idx ON municipios_artistas(uf, nome);
CREATE TABLE artista_ofertas (
 grupo_id BIGINT PRIMARY KEY REFERENCES grupos(id) ON DELETE CASCADE,
 habilitada BOOLEAN NOT NULL DEFAULT false,
 preco_tipo VARCHAR(12) NOT NULL CHECK (preco_tipo IN ('fixo','a_partir','consulta')),
 valor_centavos BIGINT CHECK (valor_centavos BETWEEN 0 AND 1000000000000),
 passagem_inclusa BOOLEAN NOT NULL DEFAULT false,
 hospedagem_inclusa BOOLEAN NOT NULL DEFAULT false,
 alimentacao_inclusa BOOLEAN NOT NULL DEFAULT false,
 integrantes INT NOT NULL CHECK (integrantes BETWEEN 1 AND 1000),
 duracao_minutos INT NOT NULL CHECK (duracao_minutos BETWEEN 1 AND 1440),
 origem_id INT REFERENCES municipios_artistas(id),
 observacoes VARCHAR(1500) NOT NULL DEFAULT '',
 CHECK (preco_tipo = 'consulta' OR valor_centavos IS NOT NULL)
);
CREATE TABLE artista_viagem_config (
 id INT PRIMARY KEY CHECK (id = 1), habilitada BOOLEAN NOT NULL DEFAULT false,
 onibus_centavos_km BIGINT, aviao_centavos_km BIGINT,
 distancia_min_km INT, aviao_desde_km INT, ida_volta BOOLEAN NOT NULL DEFAULT false,
 atualizado_por BIGINT REFERENCES usuarios(id), atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO artista_viagem_config(id) VALUES (1);
CREATE TABLE artista_orcamentos (
 id BIGSERIAL PRIMARY KEY,
 grupo_id BIGINT NOT NULL REFERENCES grupos(id),
 cliente_id BIGINT NOT NULL REFERENCES usuarios(id),
 inicio DATE NOT NULL, fim DATE NOT NULL CHECK (fim >= inicio),
 destino_id INT NOT NULL REFERENCES municipios_artistas(id),
 tipo_evento VARCHAR(100) NOT NULL,
 duracao_minutos INT NOT NULL CHECK (duracao_minutos BETWEEN 1 AND 1440),
 observacoes VARCHAR(3000) NOT NULL DEFAULT '',
 status VARCHAR(20) NOT NULL DEFAULT 'novo' CHECK (status IN ('novo','em_conversa','proposta_enviada','aceito','recusado','realizado','cancelado')),
 proposta_centavos BIGINT CHECK (proposta_centavos BETWEEN 0 AND 1000000000000),
 proposta_texto VARCHAR(3000) NOT NULL DEFAULT '',
 versao INT NOT NULL DEFAULT 0,
 motivo VARCHAR(1000) NOT NULL DEFAULT '',
 criado_em TIMESTAMPTZ NOT NULL DEFAULT now(), atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
 agenda_id BIGINT UNIQUE REFERENCES grupo_agenda(id) ON DELETE SET NULL
);
CREATE INDEX artista_orcamentos_cliente_idx ON artista_orcamentos(cliente_id, atualizado_em DESC);
CREATE INDEX artista_orcamentos_grupo_idx ON artista_orcamentos(grupo_id, atualizado_em DESC);
CREATE TABLE artista_mensagens (
 id BIGSERIAL PRIMARY KEY, orcamento_id BIGINT NOT NULL REFERENCES artista_orcamentos(id) ON DELETE CASCADE,
 usuario_id BIGINT NOT NULL REFERENCES usuarios(id), texto VARCHAR(3000) NOT NULL,
 criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX artista_mensagens_pedido_idx ON artista_mensagens(orcamento_id, id);
ALTER TABLE grupo_agenda ADD COLUMN orcamento_id BIGINT UNIQUE REFERENCES artista_orcamentos(id);
