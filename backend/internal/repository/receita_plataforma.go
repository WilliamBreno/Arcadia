package repository

import (
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type ReceitaPlataforma struct {
	Itens            int64
	TaxaCentavos     int64
	GarantiaCentavos int64
}

// ReceitaPlataforma é o relatório da seção 7.6: taxa + garantia dos itens
// pago|utilizado cujo evento já começou (filtro opcional por início do evento).
func (r *ItemPedidoRepository) ReceitaPlataforma(de, ate *time.Time) (ReceitaPlataforma, error) {
	var res ReceitaPlataforma
	q := r.db.Table("itens_pedido").
		Select("COUNT(*) as itens, COALESCE(SUM(itens_pedido.taxa_plataforma_centavos),0) as taxa_centavos, COALESCE(SUM(itens_pedido.garantia_centavos),0) as garantia_centavos").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Joins("JOIN eventos ON eventos.id = tipos_ingresso.evento_id").
		Where("itens_pedido.status IN ? AND eventos.inicio_em <= now()", []domain.StatusItemPedido{domain.StatusItemPago, domain.StatusItemUtilizado})
	if de != nil {
		q = q.Where("eventos.inicio_em >= ?", *de)
	}
	if ate != nil {
		q = q.Where("eventos.inicio_em < ?", *ate)
	}
	err := q.Scan(&res).Error
	return res, err
}
