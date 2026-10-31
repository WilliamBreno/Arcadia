-- Item 11.6: limite opcional de jurados e participantes por evento, por ordem
-- de chegada. NULL = sem limite.
CREATE TABLE vagas_evento (
    evento_id     BIGINT PRIMARY KEY REFERENCES eventos(id) ON DELETE CASCADE,
    jurados       INT CHECK (jurados > 0),
    participantes INT CHECK (participantes > 0),
    cosplay       INT CHECK (cosplay > 0),
    danca         INT CHECK (danca > 0),
    canto         INT CHECK (canto > 0),
    atuacao       INT CHECK (atuacao > 0),
    lista_espera  BOOLEAN NOT NULL DEFAULT false,
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
