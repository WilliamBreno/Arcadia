package repository

import "time"

type RelatorioEvento struct {
	EventoID   int64
	Titulo     string
	Status     string
	Vendidos   int64
	Cortesias  int64
	Receita    int64
	Checkins   int64
	Cancelados int64
}

type VendasDia struct {
	Dia        string
	Quantidade int64
	Receita    int64
}

// RelatorioPorEvento agrega, por evento do organizador, vendas (sem
// cortesias), receita bruta (preço), check-ins e cancelamentos.
func (r *ItemPedidoRepository) RelatorioPorEvento(organizadorID int64) ([]RelatorioEvento, error) {
	var linhas []RelatorioEvento
	err := r.db.Raw(`SELECT e.id AS evento_id, e.titulo, e.status,
		COUNT(ip.id) FILTER (WHERE ip.status IN ('pago','utilizado') AND NOT ip.cortesia) AS vendidos,
		COUNT(ip.id) FILTER (WHERE ip.status IN ('pago','utilizado') AND ip.cortesia) AS cortesias,
		COALESCE(SUM(ip.preco_centavos) FILTER (WHERE ip.status IN ('pago','utilizado') AND NOT ip.cortesia), 0) AS receita,
		COUNT(ip.id) FILTER (WHERE ip.status = 'utilizado') AS checkins,
		COUNT(ip.id) FILTER (WHERE ip.status IN ('cancelado','reembolsado') AND NOT ip.cortesia) AS cancelados
		FROM eventos e
		LEFT JOIN tipos_ingresso t ON t.evento_id = e.id
		LEFT JOIN itens_pedido ip ON ip.tipo_ingresso_id = t.id
		WHERE e.organizador_id = ?
		GROUP BY e.id ORDER BY e.inicio_em DESC NULLS LAST, e.id DESC`, organizadorID).Scan(&linhas).Error
	return linhas, err
}

// VendasPorDia é a série temporal de vendas pagas (sem cortesias) do
// organizador, com filtro opcional de período.
func (r *ItemPedidoRepository) VendasPorDia(organizadorID int64, de, ate *time.Time) ([]VendasDia, error) {
	var linhas []VendasDia
	q := r.db.Table("itens_pedido").
		Select("to_char(itens_pedido.criado_em, 'YYYY-MM-DD') AS dia, COUNT(*) AS quantidade, COALESCE(SUM(itens_pedido.preco_centavos), 0) AS receita").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Joins("JOIN eventos ON eventos.id = tipos_ingresso.evento_id").
		Where("eventos.organizador_id = ? AND itens_pedido.status IN ('pago','utilizado') AND itens_pedido.cortesia = false", organizadorID)
	if de != nil {
		q = q.Where("itens_pedido.criado_em >= ?", *de)
	}
	if ate != nil {
		q = q.Where("itens_pedido.criado_em < ?", *ate)
	}
	err := q.Group("dia").Order("dia").Scan(&linhas).Error
	return linhas, err
}
