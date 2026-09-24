package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type RepasseRepository struct {
	db *gorm.DB
}

func NovoRepasseRepository(db *gorm.DB) *RepasseRepository {
	return &RepasseRepository{db: db}
}

func (r *RepasseRepository) Criar(tx *gorm.DB, rep *domain.Repasse) error {
	return tx.Create(rep).Error
}

func (r *RepasseRepository) BuscarPorID(id int64) (*domain.Repasse, error) {
	var rep domain.Repasse
	if err := r.db.First(&rep, id).Error; err != nil {
		return nil, err
	}
	return &rep, nil
}

func (r *RepasseRepository) BuscarAtivoPorEvento(eventoID int64) (*domain.Repasse, error) {
	var rep domain.Repasse
	if err := r.db.Where("evento_id = ? AND status <> ?", eventoID, domain.StatusRepasseCancelado).First(&rep).Error; err != nil {
		return nil, err
	}
	return &rep, nil
}

// MarcarPagoAtomico só vence se o repasse ainda estava pendente —
// chamar duas vezes não paga (nem lança) duas vezes.
func (r *RepasseRepository) MarcarPagoAtomico(tx *gorm.DB, id int64, comprovante, observacao string) (bool, error) {
	res := tx.Model(&domain.Repasse{}).
		Where("id = ? AND status = ?", id, domain.StatusRepassePendente).
		Updates(map[string]any{
			"status": domain.StatusRepassePago, "pago_em": time.Now(),
			"comprovante_url": comprovante, "observacao": observacao,
		})
	return res.RowsAffected > 0, res.Error
}

type RepasseDetalhado struct {
	domain.Repasse
	EventoTitulo    string
	OrganizadorNome string
	ChavePix        string
	TipoChavePix    string
}

func (r *RepasseRepository) base() *gorm.DB {
	return r.db.Table("repasses").
		Select("repasses.*, eventos.titulo as evento_titulo, organizadores.nome_publico as organizador_nome, organizadores.chave_pix as chave_pix, organizadores.tipo_chave_pix as tipo_chave_pix").
		Joins("JOIN eventos ON eventos.id = repasses.evento_id").
		Joins("JOIN organizadores ON organizadores.id = repasses.organizador_id")
}

func (r *RepasseRepository) ListarPorOrganizador(organizadorID int64) ([]RepasseDetalhado, error) {
	var linhas []RepasseDetalhado
	err := r.base().Where("repasses.organizador_id = ?", organizadorID).Order("repasses.liberar_em DESC").Scan(&linhas).Error
	return linhas, err
}

func (r *RepasseRepository) ListarPorStatus(status string) ([]RepasseDetalhado, error) {
	var linhas []RepasseDetalhado
	q := r.base()
	if status != "" {
		q = q.Where("repasses.status = ?", status)
	}
	err := q.Order("repasses.liberar_em").Scan(&linhas).Error
	return linhas, err
}
