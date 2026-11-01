package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// validarOrdem exige a lista completa: uma tela desatualizada não altera
// parcialmente a ordem nem aceita itens pertencentes a outro perfil.
func validarOrdem(ids, atuais []int64) bool {
	if len(ids) != len(atuais) {
		return false
	}
	conjunto := make(map[int64]bool, len(atuais))
	for _, id := range atuais {
		conjunto[id] = true
	}
	for _, id := range ids {
		if !conjunto[id] {
			return false
		}
		delete(conjunto, id)
	}
	return len(conjunto) == 0
}

func (s *GruposService) Reordenar(usuarioID, grupoID int64, ids []int64, integrantes bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var g Grupo
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, grupoID).Error; err != nil {
			return ErrArtistaNaoEncontrado
		}
		local := *s
		local.db = tx
		if _, err := local.doGestor(usuarioID, grupoID); err != nil {
			return err
		}
		tabela, filtro, invalido := "grupo_midias", "grupo_id = ?", ErrMidiaInvalida
		if integrantes {
			tabela, filtro, invalido = "grupo_integrantes", "grupo_id = ? AND status IN ('convidado','ativo')", ErrIntegranteInvalido
		}
		var atuais []int64
		if err := tx.Table(tabela).Clauses(clause.Locking{Strength: "UPDATE"}).Where(filtro, grupoID).Pluck("id", &atuais).Error; err != nil {
			return err
		}
		if !validarOrdem(ids, atuais) {
			return fmt.Errorf("%w: a lista mudou; atualize a página e tente novamente", invalido)
		}
		for ordem, id := range ids {
			if err := tx.Table(tabela).Where("id = ? AND grupo_id = ?", id, grupoID).Update("ordem", ordem).Error; err != nil {
				return err
			}
		}
		return tx.Model(&g).Update("atualizado_em", time.Now()).Error
	})
}

// ArtistasParaSitemap usa as mesmas regras de idade da página pública.
// Grupos públicos podem ser indexados; solo exige adulto com aceite ativo.
func (s *GruposService) ArtistasParaSitemap() ([]string, error) {
	var grupos []Grupo
	if err := s.db.Where("publico").Order("id").Find(&grupos).Error; err != nil {
		return nil, err
	}
	// Consulta em lote para não fazer duas consultas por perfil solo.
	var integrantes []GrupoIntegrante
	if err := s.db.Raw(`SELECT gi.grupo_id, gi.status, gi.autorizado_em,
	 GREATEST(gi.data_nascimento, (
	  SELECT max(x.d) FROM (
	   SELECT pt.data_nascimento::date AS d FROM perfis_talento pt WHERE pt.usuario_id = gi.usuario_id
	   UNION ALL SELECT f.data_nascimento::date AS d FROM fichas_participacao f WHERE f.usuario_id = gi.usuario_id
	  ) x
	 )) AS data_nascimento
	 FROM grupo_integrantes gi JOIN grupos g ON g.id = gi.grupo_id
	 WHERE g.publico AND g.solo AND gi.usuario_id = g.lider_usuario_id AND gi.status = 'ativo'`).Scan(&integrantes).Error; err != nil {
		return nil, err
	}
	agora := time.Now()
	situacoes := make(map[int64]SituacaoIntegrante, len(integrantes))
	for _, i := range integrantes {
		situacoes[i.GrupoID] = i.situacao(agora)
	}
	slugs := []string{}
	for _, g := range grupos {
		if g.Solo {
			sit := situacoes[g.ID]
			if !sit.Visivel || sit.Menor || sit.Anonimo {
				continue
			}
		}
		slugs = append(slugs, g.Slug)
	}
	return slugs, nil
}
