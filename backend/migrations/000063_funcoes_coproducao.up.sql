-- Item 9.8: funções da equipe do organizador e coprodução com divisão
-- de repasse.

-- Funções: gestor (como antes), financeiro (vê dinheiro, não edita),
-- leitura (só vê) e checkin (só portaria e venda na porta).
ALTER TABLE organizador_membros DROP CONSTRAINT IF EXISTS organizador_membros_papel_check;
ALTER TABLE organizador_membros ADD CONSTRAINT organizador_membros_papel_check
    CHECK (papel IN ('gestor', 'financeiro', 'leitura', 'checkin'));

-- Coprodução: o organizador principal (dono do evento) convida outro
-- organizador com um percentual do repasse líquido, em pontos-base
-- (10000 = 100%). percentual_bp é o que vale; proposto_bp é a mudança
-- esperando o aceite do coprodutor.
CREATE TABLE coproducoes (
    id             BIGSERIAL PRIMARY KEY,
    evento_id      BIGINT NOT NULL REFERENCES eventos(id) ON DELETE CASCADE,
    organizador_id BIGINT NOT NULL REFERENCES organizadores(id) ON DELETE CASCADE,
    percentual_bp  INT NOT NULL DEFAULT 0 CHECK (percentual_bp >= 0 AND percentual_bp < 10000),
    proposto_bp    INT CHECK (proposto_bp >= 0 AND proposto_bp < 10000),
    status         VARCHAR(12) NOT NULL DEFAULT 'convidado' CHECK (status IN ('convidado', 'ativa', 'recusada', 'encerrada')),
    convidado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    respondido_em  TIMESTAMPTZ,
    UNIQUE (evento_id, organizador_id)
);
CREATE INDEX coproducoes_organizador ON coproducoes (organizador_id);

-- Repasse: um por evento e organizador (o principal e cada coprodutor),
-- com o percentual usado na divisão.
DROP INDEX IF EXISTS repasses_evento_ativo_idx;
CREATE UNIQUE INDEX repasses_evento_org_ativo_idx ON repasses (evento_id, organizador_id) WHERE status <> 'cancelado';
ALTER TABLE repasses ADD COLUMN percentual_bp INT NOT NULL DEFAULT 10000;
