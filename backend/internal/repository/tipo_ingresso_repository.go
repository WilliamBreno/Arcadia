package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

var ErrTipoIngressoVinculado = errors.New("este ingresso possui registros vinculados e não pode ser excluído")

func (r *TipoIngressoRepository) Excluir(t *domain.TipoIngresso, validar func([]domain.TipoIngresso) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var evento domain.Evento
		if e := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).First(&evento, t.EventoID).Error; e != nil {
			return e
		}
		var atual domain.TipoIngresso
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("evento_id = ?", t.EventoID).First(&atual, t.ID).Error; e != nil {
			return e
		}
		for _, ref := range []struct{ tabela, motivo string }{
			{"itens_pedido", "já tem pedidos ou ingressos emitidos, inclusive reservas ou registros antigos. O histórico precisa ser preservado"},
			{"pacote_itens", "faz parte de um pacote. Remova-o do pacote antes de excluir"},
			{"grupos_compra", "tem compras em grupo vinculadas. Esses registros precisam ser preservados"},
			{"fila_espera", "tem uma lista de espera vinculada. Esses registros precisam ser preservados"},
		} {
			var n int64
			if e := tx.Table(ref.tabela).Where("tipo_ingresso_id = ?", t.ID).Count(&n).Error; e != nil {
				return e
			}
			if n > 0 {
				return fmt.Errorf("%w: %s. Para interromper novas vendas, desative este ingresso.", ErrTipoIngressoVinculado, ref.motivo)
			}
		}
		var restantes []domain.TipoIngresso
		if e := tx.Where("evento_id = ? AND id <> ? AND NOT inscricao_talento AND NOT espaco_expositor AND NOT adicional", t.EventoID, t.ID).Find(&restantes).Error; e != nil {
			return e
		}
		if e := validar(restantes); e != nil {
			return e
		}
		e := tx.Delete(&atual).Error
		var pg interface{ SQLState() string }
		if errors.Is(e, gorm.ErrForeignKeyViolated) || (errors.As(e, &pg) && pg.SQLState() == "23503") {
			return ErrTipoIngressoVinculado
		}
		return e
	})
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
	// O ingresso interno da inscrição paga (item 8.4) não é do organizador editar.
	if err := r.db.Where("evento_id = ? AND NOT inscricao_talento AND NOT espaco_expositor AND NOT adicional", eventoID).Order("ordem, id").Find(&tipos).Error; err != nil {
		return nil, err
	}
	return tipos, r.PreencherSessoes(tipos)
}

func (r *TipoIngressoRepository) ContarAtivosPorEvento(eventoID int64) (int64, error) {
	var total int64
	err := r.db.Model(&domain.TipoIngresso{}).
		Where("evento_id = ? AND ativo = true AND NOT inscricao_talento AND NOT espaco_expositor AND NOT adicional", eventoID).
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
	if err := r.db.Where("evento_id = ? AND ativo = true AND NOT inscricao_talento AND NOT espaco_expositor AND NOT adicional", eventoID).Order("ordem, id").Find(&tipos).Error; err != nil {
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
		PrecoCentavos *int64
		TotalTipos    int64
	}
	err := r.db.Table("eventos e").
		Select("e.id AS evento_id, COUNT(ti.id) AS total_tipos, MIN(ti.preco_centavos) FILTER (WHERE ti.ativo) AS preco_centavos").
		Joins("LEFT JOIN tipos_ingresso ti ON ti.evento_id = e.id AND NOT ti.inscricao_talento AND NOT ti.espaco_expositor AND NOT ti.adicional").
		Where("e.id IN ?", eventoIDs).
		Group("e.id").
		Scan(&linhas).Error
	if err != nil {
		return nil, err
	}

	resultado := make(map[int64]int64, len(linhas))
	for _, l := range linhas {
		if l.PrecoCentavos != nil {
			resultado[l.EventoID] = *l.PrecoCentavos
		} else if l.TotalTipos == 0 {
			resultado[l.EventoID] = 0
		}
	}
	return resultado, nil
}

// Sem tipos cadastrados é entrada livre; tipos inativos/esgotados não significam entrada livre.
func (r *TipoIngressoRepository) EntradaLivre(eventoID int64) (bool, error) {
	var n int64
	e := r.db.Model(&domain.TipoIngresso{}).Where("evento_id = ? AND NOT inscricao_talento AND NOT espaco_expositor AND NOT adicional", eventoID).Count(&n).Error
	return n == 0, e
}

const exprEntradaLivre = "NOT EXISTS (SELECT 1 FROM tipos_ingresso ti WHERE ti.evento_id = eventos.id AND NOT ti.inscricao_talento AND NOT ti.espaco_expositor AND NOT ti.adicional)"

// PreencherSessoes carrega, em lote, as sessões de cada tipo (vazio = todas).
func (r *TipoIngressoRepository) PreencherSessoes(tipos []domain.TipoIngresso) error {
	return r.PreencherSessoesCom(r.db, tipos)
}

// PreencherSessoesCom faz o mesmo usando a conexão dada — dentro de uma
// transação, passe o tx: pedir outra conexão do pool no meio da transação
// trava a API quando o pool esgota (achado no teste de carga, item 5.5).
func (r *TipoIngressoRepository) PreencherSessoesCom(db *gorm.DB, tipos []domain.TipoIngresso) error {
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
	if err := db.Table("tipo_ingresso_sessoes").Where("tipo_ingresso_id IN ?", ids).Order("sessao_id").Scan(&linhas).Error; err != nil {
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
