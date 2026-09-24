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
		WHERE t.evento_id = ? AND ip.status IN ('pago','utilizado') AND (t.sessao_id IS NULL OR t.sessao_id = ?)`, eventoID, sessaoID).Scan(&total).Error
	if err != nil {
		return
	}
	err = r.db.Raw(`SELECT COUNT(*) FROM checkins_sessao WHERE sessao_id = ?`, sessaoID).Scan(&entraram).Error
	return
}

// ListarPagosDaSessao são os itens pagos de tipos restritos à sessão.
func (r *SessaoRepository) ListarPagosDaSessao(sessaoID int64) ([]domain.ItemPedido, error) {
	var itens []domain.ItemPedido
	err := r.db.Raw(`SELECT ip.* FROM itens_pedido ip JOIN tipos_ingresso t ON t.id = ip.tipo_ingresso_id
		WHERE t.sessao_id = ? AND ip.status = 'pago' AND ip.cortesia = false`, sessaoID).Scan(&itens).Error
	return itens, err
}

// DesativarTiposDaSessao para de vender os tipos restritos a ela.
func (r *SessaoRepository) DesativarTiposDaSessao(sessaoID int64) error {
	return r.db.Exec(`UPDATE tipos_ingresso SET ativo = false WHERE sessao_id = ?`, sessaoID).Error
}

// TitularesDoEventoTodo devolve e-mails de quem tem ingresso do evento todo.
func (r *SessaoRepository) TitularesDoEventoTodo(eventoID int64) ([]string, error) {
	var emails []string
	err := r.db.Raw(`SELECT DISTINCT ip.titular_email FROM itens_pedido ip JOIN tipos_ingresso t ON t.id = ip.tipo_ingresso_id
		WHERE t.evento_id = ? AND t.sessao_id IS NULL AND ip.status = 'pago' AND ip.titular_email <> ''`, eventoID).Scan(&emails).Error
	return emails, err
}

// EmUso: há tipo de ingresso ou entrada (check-in) ligados à sessão.
func (r *SessaoRepository) EmUso(sessaoID int64) (bool, error) {
	var total int64
	err := r.db.Raw(`SELECT (SELECT COUNT(*) FROM tipos_ingresso WHERE sessao_id = ?) + (SELECT COUNT(*) FROM checkins_sessao WHERE sessao_id = ?)`, sessaoID, sessaoID).Scan(&total).Error
	return total > 0, err
}

func (r *SessaoRepository) Excluir(s *domain.Sessao) error { return r.db.Delete(s).Error }

func (r *SessaoRepository) HoraCheckin(itemID, sessaoID int64) (time.Time, error) {
	var t time.Time
	err := r.db.Raw(`SELECT criado_em FROM checkins_sessao WHERE item_id = ? AND sessao_id = ?`, itemID, sessaoID).Scan(&t).Error
	return t, err
}
