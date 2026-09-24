package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// ContarAtivosPorTipos conta itens que ocupam vaga (reservado/pago/utilizado)
// por tipo de ingresso — base da virada de lote por quantidade. Passe tx
// para ler dentro de uma transação, ou nil.
func (r *ItemPedidoRepository) ContarAtivosPorTipos(db *gorm.DB, tipoIDs []int64) (map[int64]int64, error) {
	if db == nil {
		db = r.db
	}
	resultado := map[int64]int64{}
	if len(tipoIDs) == 0 {
		return resultado, nil
	}
	var linhas []struct {
		TipoIngressoID int64
		Total          int64
	}
	err := db.Model(&domain.ItemPedido{}).
		Select("tipo_ingresso_id, COUNT(*) as total").
		Where("tipo_ingresso_id IN ? AND status IN ?", tipoIDs, []domain.StatusItemPedido{
			domain.StatusItemReservado, domain.StatusItemPago, domain.StatusItemUtilizado,
		}).
		Group("tipo_ingresso_id").Scan(&linhas).Error
	if err != nil {
		return nil, err
	}
	for _, l := range linhas {
		resultado[l.TipoIngressoID] = l.Total
	}
	return resultado, nil
}

// ListarPorGrupoLote devolve os tipos do mesmo lote_grupo de um evento.
func (r *TipoIngressoRepository) ListarPorGrupoLote(db *gorm.DB, eventoID int64, grupo string) ([]domain.TipoIngresso, error) {
	if db == nil {
		db = r.db
	}
	var tipos []domain.TipoIngresso
	err := db.Where("evento_id = ? AND lote_grupo = ?", eventoID, grupo).Find(&tipos).Error
	return tipos, err
}
