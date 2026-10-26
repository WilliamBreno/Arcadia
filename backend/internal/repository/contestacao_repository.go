package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type ContestacaoRepository struct {
	db *gorm.DB
}

func NovoContestacaoRepository(db *gorm.DB) *ContestacaoRepository {
	return &ContestacaoRepository{db: db}
}

func (r *ContestacaoRepository) BuscarPorPagamento(tx *gorm.DB, pagamentoID int64) (*domain.Contestacao, error) {
	var c domain.Contestacao
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "pagamento_id = ?", pagamentoID).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ContestacaoRepository) Salvar(tx *gorm.DB, c *domain.Contestacao) error {
	c.AtualizadoEm = time.Now()
	return tx.Save(c).Error
}

// ContestacaoDetalhada é a linha do painel admin.
type ContestacaoDetalhada struct {
	domain.Contestacao
	EventoTitulo string
	UsuarioEmail string
	MPPaymentID  string
}

func (r *ContestacaoRepository) Listar(status string) ([]ContestacaoDetalhada, error) {
	var linhas []ContestacaoDetalhada
	q := r.db.Table("contestacoes").
		Select("contestacoes.*, eventos.titulo as evento_titulo, usuarios.email as usuario_email, pagamentos.mp_payment_id").
		Joins("JOIN eventos ON eventos.id = contestacoes.evento_id").
		Joins("JOIN usuarios ON usuarios.id = contestacoes.usuario_id").
		Joins("JOIN pagamentos ON pagamentos.id = contestacoes.pagamento_id").
		Order("contestacoes.criado_em DESC")
	if status != "" {
		q = q.Where("contestacoes.status = ?", status)
	}
	return linhas, q.Scan(&linhas).Error
}
