package repository

import (
	"strings"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type AfiliadoRepository struct {
	db *gorm.DB
}

func NovoAfiliadoRepository(db *gorm.DB) *AfiliadoRepository { return &AfiliadoRepository{db: db} }

func (r *AfiliadoRepository) Criar(a *domain.Afiliado) error  { return r.db.Create(a).Error }
func (r *AfiliadoRepository) Salvar(a *domain.Afiliado) error { return r.db.Save(a).Error }

func (r *AfiliadoRepository) BuscarPorID(id int64) (*domain.Afiliado, error) {
	var a domain.Afiliado
	if err := r.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// BuscarAtivoPorCodigo é case-insensitive e restrito ao evento.
func (r *AfiliadoRepository) BuscarAtivoPorCodigo(tx *gorm.DB, eventoID int64, codigo string) (*domain.Afiliado, error) {
	if tx == nil {
		tx = r.db
	}
	var a domain.Afiliado
	err := tx.First(&a, "evento_id = ? AND codigo = ? AND ativo = true", eventoID, strings.ToUpper(strings.TrimSpace(codigo))).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

type AfiliadoComStats struct {
	domain.Afiliado
	Pedidos int64
	Itens   int64
	Receita int64
}

// ListarComStats conta só pedidos pagos atribuídos ao afiliado (sem cortesias).
func (r *AfiliadoRepository) ListarComStats(eventoID int64) ([]AfiliadoComStats, error) {
	var linhas []AfiliadoComStats
	err := r.db.Raw(`SELECT a.*,
		COUNT(DISTINCT p.id) FILTER (WHERE ip.id IS NOT NULL) AS pedidos,
		COUNT(ip.id) AS itens,
		COALESCE(SUM(ip.preco_centavos), 0) AS receita
		FROM afiliados a
		LEFT JOIN pedidos p ON p.afiliado_id = a.id AND p.status = 'pago'
		LEFT JOIN itens_pedido ip ON ip.pedido_id = p.id AND ip.status IN ('pago','utilizado') AND ip.cortesia = false
		WHERE a.evento_id = ?
		GROUP BY a.id ORDER BY a.id`, eventoID).Scan(&linhas).Error
	return linhas, err
}
