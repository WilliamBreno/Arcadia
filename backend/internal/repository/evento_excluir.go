package repository

import (
	"errors"
	"fmt"
	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrExcluirRascunho = errors.New("não foi possível excluir o rascunho")

func (r *EventoRepository) ExcluirRascunho(organizadorID, id int64) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var e domain.Evento
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organizador_id = ?", id, organizadorID).First(&e).Error; err != nil {
			return err
		}
		if e.Status != domain.StatusEventoRascunho {
			return fmt.Errorf("%w: somente eventos em rascunho podem ser excluídos", ErrExcluirRascunho)
		}
		for _, table := range []string{"pedidos", "fichas_participacao", "atracoes"} {
			var n int64
			if err := tx.Table(table).Where("evento_id = ?", id).Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return fmt.Errorf("%w: existem pedidos, inscrições ou atrações vinculados. Esses registros precisam ser preservados", ErrExcluirRascunho)
			}
		}
		return tx.Delete(&e).Error
	})
	var pg interface{ SQLState() string }
	if errors.Is(err, gorm.ErrForeignKeyViolated) || (errors.As(err, &pg) && pg.SQLState() == "23503") {
		return fmt.Errorf("%w: existem registros vinculados que precisam ser preservados", ErrExcluirRascunho)
	}
	return err
}
