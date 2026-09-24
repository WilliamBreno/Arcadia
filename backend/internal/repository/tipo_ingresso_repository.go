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
	lista := []domain.TipoIngresso{t}
	if err := r.PreencherSessoes(lista); err != nil {
		return nil, err
	}
	return &lista[0], nil
}

func (r *TipoIngressoRepository) ListarPorEvento(eventoID int64) ([]domain.TipoIngresso, error) {
	var tipos []domain.TipoIngresso
	if err := r.db.Where("evento_id = ?", eventoID).Order("ordem, id").Find(&tipos).Error; err != nil {
		return nil, err
	}
	return tipos, r.PreencherSessoes(tipos)
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
	if err := r.db.Where("evento_id = ? AND ativo = true", eventoID).Order("ordem, id").Find(&tipos).Error; err != nil {
		return nil, err
	}
	return tipos, r.PreencherSessoes(tipos)
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

// PreencherSessoes carrega, em lote, as sessões de cada tipo (vazio = todas).
func (r *TipoIngressoRepository) PreencherSessoes(tipos []domain.TipoIngresso) error {
	if len(tipos) == 0 {
		return nil
	}
	ids := make([]int64, len(tipos))
	for i := range tipos {
		ids[i] = tipos[i].ID
	}
	var linhas []struct {
		TipoIngressoID int64
		SessaoID       int64
	}
	if err := r.db.Table("tipo_ingresso_sessoes").Where("tipo_ingresso_id IN ?", ids).Order("sessao_id").Scan(&linhas).Error; err != nil {
		return err
	}
	porTipo := map[int64][]int64{}
	for _, l := range linhas {
		porTipo[l.TipoIngressoID] = append(porTipo[l.TipoIngressoID], l.SessaoID)
	}
	for i := range tipos {
		tipos[i].SessaoIDs = porTipo[tipos[i].ID]
	}
	return nil
}

// DefinirSessoes substitui o conjunto de sessões do tipo (vazio = todas).
func (r *TipoIngressoRepository) DefinirSessoes(tipoID int64, sessaoIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM tipo_ingresso_sessoes WHERE tipo_ingresso_id = ?`, tipoID).Error; err != nil {
			return err
		}
		for _, id := range sessaoIDs {
			if err := tx.Exec(`INSERT INTO tipo_ingresso_sessoes (tipo_ingresso_id, sessao_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, tipoID, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
