-- Item 8.2: voto popular validado pelo check-in e devolutiva dos jurados.
-- Decisões do dono (30/10/2026):
--  - o voto gera um "Prêmio do público" separado; a nota dos jurados não muda;
--  - 1 voto por ingresso que passou no check-in, em cada categoria; dá para
--    trocar o voto enquanto a votação está aberta;
--  - o organizador abre e fecha a votação; a contagem só aparece quando ele
--    libera o resultado (e a votação está fechada);
--  - devolutiva: média e comentário por critério, jurado anônimo, liberada
--    pelo organizador; cada participante vê só a sua.
ALTER TABLE eventos
    ADD COLUMN votacao_aberta_em      TIMESTAMPTZ,
    ADD COLUMN votacao_fechada_em     TIMESTAMPTZ,
    ADD COLUMN devolutiva_liberada_em TIMESTAMPTZ;

CREATE TABLE votos_publico (
    item_id          BIGINT      NOT NULL REFERENCES itens_pedido (id) ON DELETE CASCADE,
    evento_id        BIGINT      NOT NULL REFERENCES eventos (id) ON DELETE CASCADE,
    tipo_apresentacao VARCHAR(20) NOT NULL DEFAULT '',
    ficha_id         BIGINT      NOT NULL REFERENCES fichas_participacao (id) ON DELETE CASCADE,
    criado_em        TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (item_id, tipo_apresentacao)
);
CREATE INDEX votos_publico_evento_idx ON votos_publico (evento_id, tipo_apresentacao, ficha_id);
