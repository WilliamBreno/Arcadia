package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type AvaliacaoRepository struct {
	db *gorm.DB
}

func NovoAvaliacaoRepository(db *gorm.DB) *AvaliacaoRepository {
	return &AvaliacaoRepository{db: db}
}

func (r *AvaliacaoRepository) DB() *gorm.DB { return r.db }

func (r *AvaliacaoRepository) CriteriosPorEvento(eventoID int64) ([]domain.CriterioAvaliacao, error) {
	var lista []domain.CriterioAvaliacao
	err := r.db.Where("evento_id = ?", eventoID).Order("ordem, id").Find(&lista).Error
	return lista, err
}

func (r *AvaliacaoRepository) CriarCriterio(c *domain.CriterioAvaliacao) error {
	return r.db.Create(c).Error
}

func (r *AvaliacaoRepository) SalvarCriterio(c *domain.CriterioAvaliacao) error {
	return r.db.Save(c).Error
}

func (r *AvaliacaoRepository) BuscarCriterio(id int64) (*domain.CriterioAvaliacao, error) {
	var c domain.CriterioAvaliacao
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *AvaliacaoRepository) ExcluirCriterio(c *domain.CriterioAvaliacao) error {
	return r.db.Delete(c).Error
}

func (r *AvaliacaoRepository) ContarAvaliacoesDoCriterio(criterioID int64) (int64, error) {
	var total int64
	err := r.db.Model(&domain.Avaliacao{}).Where("criterio_id = ?", criterioID).Count(&total).Error
	return total, err
}

func (r *AvaliacaoRepository) DoJuradoNaFicha(fichaID, juradoID int64) ([]domain.Avaliacao, error) {
	var lista []domain.Avaliacao
	err := r.db.Where("ficha_id = ? AND jurado_usuario_id = ?", fichaID, juradoID).Find(&lista).Error
	return lista, err
}

// SalvarNotas faz upsert por (ficha, jurado, critério) — só chamado
// enquanto a avaliação não está finalizada.
func (r *AvaliacaoRepository) SalvarNotas(tx *gorm.DB, notas []domain.Avaliacao) error {
	if len(notas) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ficha_id"}, {Name: "jurado_usuario_id"}, {Name: "criterio_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"nota", "comentario"}),
	}).Create(&notas).Error
}

func (r *AvaliacaoRepository) Finalizar(tx *gorm.DB, fichaID, juradoID int64) error {
	return tx.Model(&domain.Avaliacao{}).
		Where("ficha_id = ? AND jurado_usuario_id = ?", fichaID, juradoID).
		Update("finalizada", true).Error
}

// FinalizadasDoEvento traz as avaliações finalizadas das fichas do evento.
func (r *AvaliacaoRepository) FinalizadasDoEvento(eventoID int64) ([]domain.Avaliacao, error) {
	var lista []domain.Avaliacao
	err := r.db.Table("avaliacoes").
		Select("avaliacoes.*").
		Joins("JOIN fichas_participacao ON fichas_participacao.id = avaliacoes.ficha_id").
		Where("fichas_participacao.evento_id = ? AND avaliacoes.finalizada = true", eventoID).
		Scan(&lista).Error
	return lista, err
}
