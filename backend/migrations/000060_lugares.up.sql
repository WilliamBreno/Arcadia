-- Item 9.5: mapa de assentos e mesas.
-- Setor: fileiras (teatro, auditório) ou mesas (jantar, show). Aceita um
-- ou mais tipos de ingresso (inteira, meia, lotes), que disputam as
-- mesmas cadeiras.
CREATE TABLE setores_mapa (
    id         BIGSERIAL PRIMARY KEY,
    evento_id  BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    nome       TEXT NOT NULL,
    formato    TEXT NOT NULL CHECK (formato IN ('fileiras', 'mesas')),
    ordem      INT NOT NULL DEFAULT 0,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (evento_id, nome)
);

CREATE TABLE setor_tipos (
    setor_id         BIGINT NOT NULL REFERENCES setores_mapa(id) ON DELETE CASCADE,
    tipo_ingresso_id BIGINT NOT NULL REFERENCES tipos_ingresso(id) ON DELETE CASCADE,
    PRIMARY KEY (setor_id, tipo_ingresso_id)
);
CREATE INDEX setor_tipos_tipo ON setor_tipos (tipo_ingresso_id);

-- Lugar = uma cadeira. grupo é a fileira ("A") ou a mesa ("3"); numero é
-- o lugar na fileira ou a cadeira da mesa. mesa_inteira: as cadeiras da
-- mesa só saem juntas, no mesmo pedido.
CREATE TABLE lugares (
    id           BIGSERIAL PRIMARY KEY,
    setor_id     BIGINT NOT NULL REFERENCES setores_mapa(id) ON DELETE CASCADE,
    grupo        TEXT NOT NULL,
    numero       INT NOT NULL CHECK (numero > 0),
    ordem_grupo  INT NOT NULL DEFAULT 0,
    mesa_inteira BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (setor_id, grupo, numero)
);

ALTER TABLE itens_pedido
    ADD COLUMN lugar_id BIGINT REFERENCES lugares(id) ON DELETE SET NULL,
    ADD COLUMN lugar_rotulo TEXT NOT NULL DEFAULT '';

-- Dois ingressos ativos nunca ocupam a mesma cadeira (reserva em aberto
-- também segura; expirado, cancelado e reembolsado liberam).
CREATE UNIQUE INDEX itens_pedido_lugar_ativo ON itens_pedido (lugar_id)
    WHERE lugar_id IS NOT NULL AND status IN ('reservado', 'pago', 'utilizado');
