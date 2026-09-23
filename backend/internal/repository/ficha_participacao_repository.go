package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type FichaParticipacaoRepository struct {
	db *gorm.DB
}

func NovoFichaParticipacaoRepository(db *gorm.DB) *FichaParticipacaoRepository {
	return &FichaParticipacaoRepository{db: db}
}

func (r *FichaParticipacaoRepository) Criar(f *domain.FichaParticipacao) error {
	return r.db.Create(f).Error
}

func (r *FichaParticipacaoRepository) Salvar(f *domain.FichaParticipacao) error {
	return r.db.Save(f).Error
}

func (r *FichaParticipacaoRepository) BuscarPorID(id int64) (*domain.FichaParticipacao, error) {
	var f domain.FichaParticipacao
	if err := r.db.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FichaParticipacaoRepository) Buscar(eventoID, usuarioID int64, papel domain.Papel) (*domain.FichaParticipacao, error) {
	var f domain.FichaParticipacao
	err := r.db.First(&f, "evento_id = ? AND usuario_id = ? AND papel = ?", eventoID, usuarioID, papel).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FichaParticipacaoRepository) ListarPorEvento(eventoID int64, statusFiltro *domain.StatusFicha) ([]domain.FichaParticipacao, error) {
	query := r.db.Where("evento_id = ?", eventoID)
	if statusFiltro != nil {
		query = query.Where("status = ?", *statusFiltro)
	}
	var fichas []domain.FichaParticipacao
	err := query.Order("ordem_apresentacao, criado_em").Find(&fichas).Error
	return fichas, err
}

// ListarPorEventoEPapel filtra também por papel — usado na área do
// jurado, que só pode ver fichas de participantes, nunca de outros
// jurados (seção 3 do plano).
func (r *FichaParticipacaoRepository) ListarPorEventoEPapel(eventoID int64, papel domain.Papel, statusFiltro *domain.StatusFicha) ([]domain.FichaParticipacao, error) {
	query := r.db.Where("evento_id = ? AND papel = ?", eventoID, papel)
	if statusFiltro != nil {
		query = query.Where("status = ?", *statusFiltro)
	}
	var fichas []domain.FichaParticipacao
	err := query.Order("ordem_apresentacao, criado_em").Find(&fichas).Error
	return fichas, err
}
