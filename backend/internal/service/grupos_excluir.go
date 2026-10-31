package service

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrExcluirArtista = errors.New("não foi possível excluir o perfil de artista")

// Somente o titular/líder pode excluir; históricos de terceiros são preservados.
func (s *GruposService) Excluir(usuarioID, grupoID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var g Grupo
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, grupoID).Error; err != nil {
			return ErrArtistaNaoEncontrado
		}
		if g.LiderUsuarioID != usuarioID {
			return ErrArtistaSemPermissao
		}
		for _, table := range []string{"artista_orcamentos", "atracoes", "fichas_participacao"} {
			var n int64
			if err := tx.Table(table).Where("grupo_id = ?", grupoID).Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return fmt.Errorf("%w: há solicitações de contratação, atrações ou inscrições vinculadas. Para preservar esse histórico, retire a visibilidade pública na seção Perfil", ErrExcluirArtista)
			}
		}
		return tx.Delete(&g).Error
	})
}
