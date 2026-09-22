package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type TipoIngressoRepository struct {
	db *gorm.DB
}

func NovoTipoIngressoRepository(db *gorm.DB) *TipoIngressoRepository {
	return &TipoIngressoRepository{db: db}
}

func (r *TipoIngressoRepository) Criar(t *domain.TipoIngresso) error {
	return r.db.Create(t).Error
}

func (r *TipoIngressoRepository) Salvar(t *domain.TipoIngresso) error {
	return r.db.Save(t).Error
}

func (r *TipoIngressoRepository) Excluir(t *domain.TipoIngresso) error {
	return r.db.Delete(t).Error
}

func (r *TipoIngressoRepository) BuscarPorID(id int64) (*domain.TipoIngresso, error) {
	var t domain.TipoIngresso
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TipoIngressoRepository) ListarPorEvento(eventoID int64) ([]domain.TipoIngresso, error) {
	var tipos []domain.TipoIngresso
	if err := r.db.Where("evento_id = ?", eventoID).Order("ordem, id").Find(&tipos).Error; err != nil {
		return nil, err
	}
	return tipos, nil
}

func (r *TipoIngressoRepository) ContarAtivosPorEvento(eventoID int64) (int64, error) {
	var total int64
	err := r.db.Model(&domain.TipoIngresso{}).
		Where("evento_id = ? AND ativo = true", eventoID).
		Count(&total).Error
	return total, err
}

func (r *TipoIngressoRepository) ExisteComPrecoMaiorQueZero(eventoID int64) (bool, error) {
	var total int64
	err := r.db.Model(&domain.TipoIngresso{}).
		Where("evento_id = ? AND ativo = true AND preco_centavos > 0", eventoID).
		Count(&total).Error
	return total > 0, err
}

// ListarAtivosPublicoPorEvento retorna os tipos ativos de um evento, para
// exibir preço/quantidade na página pública do evento.
func (r *TipoIngressoRepository) ListarAtivosPublicoPorEvento(eventoID int64) ([]domain.TipoIngresso, error) {
	var tipos []domain.TipoIngresso
	err := r.db.Where("evento_id = ? AND ativo = true", eventoID).Order("ordem, id").Find(&tipos).Error
	return tipos, err
}

// PrecoMinimoPorEvento retorna, para cada evento_id em eventoIDs, o menor
// preco_centavos entre os tipos ativos — usado no "a partir de R$ X" dos
// cards de evento (evita N+1 ao montar a listagem pública).
func (r *TipoIngressoRepository) PrecoMinimoPorEvento(eventoIDs []int64) (map[int64]int64, error) {
	if len(eventoIDs) == 0 {
		return map[int64]int64{}, nil
	}

	var linhas []struct {
		EventoID      int64
		PrecoCentavos int64
	}
	err := r.db.Model(&domain.TipoIngresso{}).
		Select("evento_id, MIN(preco_centavos) as preco_centavos").
		Where("evento_id IN ? AND ativo = true", eventoIDs).
		Group("evento_id").
		Scan(&linhas).Error
	if err != nil {
		return nil, err
	}

	resultado := make(map[int64]int64, len(linhas))
	for _, l := range linhas {
		resultado[l.EventoID] = l.PrecoCentavos
	}
	return resultado, nil
}
