-- Item 9.4: venda na porta pelo celular. Decisões do dono (30/10/2026):
-- Pix na tela (Mercado Pago, como online) ou dinheiro registrado; mesma taxa
-- (no Pix, somada ao preço; no dinheiro, o comprador paga só o preço e a
-- taxa sai do repasse, podendo deixar saldo devedor); nome e e-mail
-- opcionais, ingresso já nasce usado e sem cancelamento pelo comprador;
-- vendem o organizador e a equipe de check-in, no mesmo estoque, até o fim
-- do evento.
-- O pedido da porta fica num usuário de sistema (sem senha, e-mail .invalid,
-- ninguém entra com ele), para nada aparecer na conta de quem vendeu.
INSERT INTO usuarios (nome, email, receber_novidades, criado_em)
VALUES ('Bilheteria (venda na porta)', 'bilheteria@sistema.evve.invalid', false, now())
ON CONFLICT (email) DO NOTHING;

ALTER TABLE pedidos
    ADD COLUMN canal VARCHAR(10) NOT NULL DEFAULT 'online' CHECK (canal IN ('online', 'porta')),
    ADD COLUMN vendido_por BIGINT REFERENCES usuarios (id),
    ADD COLUMN metodo_porta VARCHAR(10) CHECK (metodo_porta IN ('pix', 'dinheiro'));
CREATE INDEX pedidos_porta_idx ON pedidos (evento_id, vendido_por) WHERE canal = 'porta';
