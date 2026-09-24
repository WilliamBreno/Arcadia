package repository

import (
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func (r *ItemPedidoRepository) ListarCortesiasPorEvento(eventoID int64) ([]ItemVenda, error) {
	var linhas []ItemVenda
	err := r.db.Table("itens_pedido").
		Select("itens_pedido.*, tipos_ingresso.nome as tipo_ingresso_nome").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("tipos_ingresso.evento_id = ? AND itens_pedido.cortesia = true", eventoID).
		Order("itens_pedido.criado_em DESC").
		Scan(&linhas).Error
	return linhas, err
}

// RevogarCortesia é atômico: só cancela se ainda estiver `pago` e for
// cortesia do evento informado.
func (r *ItemPedidoRepository) RevogarCortesia(eventoID, itemID int64) (bool, error) {
	res := r.db.Exec(`UPDATE itens_pedido SET status = ?, cancelado_em = ?, motivo_cancelamento = 'cortesia revogada'
		WHERE id = ? AND cortesia = true AND status = ?
		AND tipo_ingresso_id IN (SELECT id FROM tipos_ingresso WHERE evento_id = ?)`,
		domain.StatusItemCancelado, time.Now(), itemID, domain.StatusItemPago, eventoID)
	return res.RowsAffected > 0, res.Error
}

// ListarCompradoresParaExportacao traz todos os itens do evento que já
// tiveram pagamento ou emissão (pago, utilizado, cancelado, reembolsado).
func (r *ItemPedidoRepository) ListarParaExportacao(eventoID int64) ([]ItemVenda, error) {
	var linhas []ItemVenda
	err := r.db.Table("itens_pedido").
		Select("itens_pedido.*, tipos_ingresso.nome as tipo_ingresso_nome").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("tipos_ingresso.evento_id = ? AND itens_pedido.status IN ?", eventoID, []domain.StatusItemPedido{
			domain.StatusItemPago, domain.StatusItemUtilizado, domain.StatusItemCancelado, domain.StatusItemReembolsado,
		}).
		Order("itens_pedido.criado_em").
		Scan(&linhas).Error
	return linhas, err
}
