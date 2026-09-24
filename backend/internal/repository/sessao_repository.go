package repository

import (
	"gorm.io/gorm"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type SessaoRepository struct {
	db *gorm.DB
}

func NovoSessaoRepository(db *gorm.DB) *SessaoRepository { return &SessaoRepository{db: db} }

func (r *SessaoRepository) Criar(s *domain.Sessao) error  { return r.db.Create(s).Error }
func (r *SessaoRepository) Salvar(s *domain.Sessao) error { return r.db.Save(s).Error }

func (r *SessaoRepository) BuscarPorID(id int64) (*domain.Sessao, error) {
	var s domain.Sessao
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessaoRepository) ListarPorEvento(eventoID int64) ([]domain.Sessao, error) {
	var lista []domain.Sessao
	err := r.db.Where("evento_id = ?", eventoID).Order("inicio_em, id").Find(&lista).Error
	return lista, err
}

// RegistrarCheckin grava a entrada do item na sessão; false = já tinha
// entrado nessa sessão (UNIQUE item+sessão barra corrida entre leitores).
func (r *SessaoRepository) RegistrarCheckin(itemID, sessaoID int64) (bool, error) {
	res := r.db.Exec(`INSERT INTO checkins_sessao (item_id, sessao_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, itemID, sessaoID)
	return res.RowsAffected > 0, res.Error
}

// ResumoDaSessao: ingressos válidos para a sessão (evento todo + os restritos
// a ela) e quantos já entraram nela.
func (r *SessaoRepository) ResumoDaSessao(eventoID, sessaoID int64) (total, entraram int64, err error) {
	err = r.db.Raw(`SELECT COUNT(*) FROM itens_pedido ip JOIN tipos_ingresso t ON t.id = ip.tipo_ingresso_id
		WHERE t.evento_id = ? AND ip.status IN ('pago','utilizado')
		AND (NOT EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts WHERE ts.tipo_ingresso_id = t.id)
		     OR EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts WHERE ts.tipo_ingresso_id = t.id AND ts.sessao_id = ?))`, eventoID, sessaoID).Scan(&total).Error
	if err != nil {
		return
	}
	err = r.db.Raw(`SELECT COUNT(*) FROM checkins_sessao WHERE sessao_id = ?`, sessaoID).Scan(&entraram).Error
	return
}

// ListarPagosSemSessaoAtiva: itens pagos de tipos que incluem a sessão
// cancelada e que não têm mais nenhuma sessão ativa (nada a que ir → reembolso).
func (r *SessaoRepository) ListarPagosSemSessaoAtiva(eventoID, sessaoID int64) ([]domain.ItemPedido, error) {
	var itens []domain.ItemPedido
	err := r.db.Raw(`SELECT ip.* FROM itens_pedido ip JOIN tipos_ingresso t ON t.id = ip.tipo_ingresso_id
		WHERE t.evento_id = ? AND ip.status = 'pago' AND ip.cortesia = false
		AND EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts WHERE ts.tipo_ingresso_id = t.id AND ts.sessao_id = ?)
		AND NOT EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts JOIN sessoes s ON s.id = ts.sessao_id
			WHERE ts.tipo_ingresso_id = t.id AND s.status = 'ativa')`, eventoID, sessaoID).Scan(&itens).Error
	return itens, err
}

// DesativarTiposSemSessaoAtiva para de vender tipos restritos cujas sessões
// foram todas canceladas.
func (r *SessaoRepository) DesativarTiposSemSessaoAtiva(eventoID int64) error {
	return r.db.Exec(`UPDATE tipos_ingresso t SET ativo = false WHERE t.evento_id = ?
		AND EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts WHERE ts.tipo_ingresso_id = t.id)
		AND NOT EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts JOIN sessoes s ON s.id = ts.sessao_id
			WHERE ts.tipo_ingresso_id = t.id AND s.status = 'ativa')`, eventoID).Error
}

// TitularesAfetados: quem continua com ingresso válido em outras sessões mas
// perdeu esta (evento todo ou conjunto de dias que a inclui) — só avisar.
func (r *SessaoRepository) TitularesAfetados(eventoID, sessaoID int64) ([]string, error) {
	var emails []string
	err := r.db.Raw(`SELECT DISTINCT ip.titular_email FROM itens_pedido ip JOIN tipos_ingresso t ON t.id = ip.tipo_ingresso_id
		WHERE t.evento_id = ? AND ip.status = 'pago' AND ip.titular_email <> ''
		AND (NOT EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts WHERE ts.tipo_ingresso_id = t.id)
		     OR (EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts WHERE ts.tipo_ingresso_id = t.id AND ts.sessao_id = ?)
		         AND EXISTS (SELECT 1 FROM tipo_ingresso_sessoes ts JOIN sessoes s ON s.id = ts.sessao_id
		                     WHERE ts.tipo_ingresso_id = t.id AND s.status = 'ativa')))`, eventoID, sessaoID).Scan(&emails).Error
	return emails, err
}

// EmUso: há tipo de ingresso ou entrada (check-in) ligados à sessão.
func (r *SessaoRepository) EmUso(sessaoID int64) (bool, error) {
	var total int64
	err := r.db.Raw(`SELECT (SELECT COUNT(*) FROM tipo_ingresso_sessoes WHERE sessao_id = ?) + (SELECT COUNT(*) FROM checkins_sessao WHERE sessao_id = ?)`, sessaoID, sessaoID).Scan(&total).Error
	return total > 0, err
}

func (r *SessaoRepository) Excluir(s *domain.Sessao) error { return r.db.Delete(s).Error }

func (r *SessaoRepository) HoraCheckin(itemID, sessaoID int64) (time.Time, error) {
	var t time.Time
	err := r.db.Raw(`SELECT criado_em FROM checkins_sessao WHERE item_id = ? AND sessao_id = ?`, itemID, sessaoID).Scan(&t).Error
	return t, err
}
