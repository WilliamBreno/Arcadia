package repository

import (
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// ItemPacote é um ingresso válido para o pacote offline.
type ItemPacote struct {
	Codigo         string
	QRToken        string
	TitularNome    string
	Status         string
	UtilizadoEm    *time.Time
	TipoIngressoID int64
	TipoNome       string
	MeiaEntrada    bool
}

// ListarParaPacote traz os itens pago/utilizado do evento (inclui cortesias).
func (r *ItemPedidoRepository) ListarParaPacote(eventoID int64) ([]ItemPacote, error) {
	var linhas []ItemPacote
	err := r.db.Raw(`SELECT ip.codigo, ip.qr_token, ip.titular_nome, ip.status, ip.utilizado_em,
		ip.tipo_ingresso_id, t.nome AS tipo_nome, t.meia_entrada
		FROM itens_pedido ip JOIN tipos_ingresso t ON t.id = ip.tipo_ingresso_id
		WHERE t.evento_id = ? AND ip.status IN ('pago', 'utilizado')
		ORDER BY ip.id`, eventoID).Scan(&linhas).Error
	return linhas, err
}

// MarcarUtilizadoAtomicoEm é o check-in atômico com o horário REAL da leitura
// (offline pode sincronizar horas depois).
func (r *ItemPedidoRepository) MarcarUtilizadoAtomicoEm(itemID int64, quando time.Time) (bool, error) {
	res := r.db.Model(&domain.ItemPedido{}).
		Where("id = ? AND status = ?", itemID, domain.StatusItemPago).
		Updates(map[string]any{"status": domain.StatusItemUtilizado, "utilizado_em": quando})
	return res.RowsAffected > 0, res.Error
}

type CheckinOffline struct {
	EntradaID      string
	ItemID         *int64
	Codigo         string
	SessaoID       *int64
	StaffUsuarioID int64
	LidoEm         time.Time
	Resultado      string
}

// BuscarOfflinePorEntrada devolve o resultado já gravado (idempotência).
func (r *ItemPedidoRepository) BuscarOfflinePorEntrada(eventoID int64, entradaID string) (string, bool) {
	var resultado string
	res := r.db.Raw(`SELECT resultado FROM checkins_offline WHERE evento_id = ? AND entrada_id = ?`, eventoID, entradaID).Scan(&resultado)
	return resultado, res.Error == nil && res.RowsAffected > 0 && resultado != ""
}

func (r *ItemPedidoRepository) GravarOffline(eventoID int64, c CheckinOffline) error {
	return r.db.Exec(`INSERT INTO checkins_offline (evento_id, entrada_id, item_id, codigo, sessao_id, staff_usuario_id, lido_em, resultado)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (evento_id, entrada_id) DO NOTHING`,
		eventoID, c.EntradaID, c.ItemID, c.Codigo, c.SessaoID, c.StaffUsuarioID, c.LidoEm, c.Resultado).Error
}

type ConflitoOffline struct {
	Codigo      string
	TitularNome string
	LidoEm      time.Time
	StaffNome   string
	SessaoID    *int64
}

// ListarConflitosOffline: entradas offline recusadas por já terem entrado antes.
func (r *ItemPedidoRepository) ListarConflitosOffline(eventoID int64) ([]ConflitoOffline, error) {
	var linhas []ConflitoOffline
	err := r.db.Raw(`SELECT c.codigo, COALESCE(ip.titular_nome, '') AS titular_nome, c.lido_em, u.nome AS staff_nome, c.sessao_id
		FROM checkins_offline c LEFT JOIN itens_pedido ip ON ip.id = c.item_id JOIN usuarios u ON u.id = c.staff_usuario_id
		WHERE c.evento_id = ? AND c.resultado = 'conflito' ORDER BY c.lido_em`, eventoID).Scan(&linhas).Error
	return linhas, err
}
