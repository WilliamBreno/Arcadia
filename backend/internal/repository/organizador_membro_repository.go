package repository

import (
	"time"

	"gorm.io/gorm"
)

type OrganizadorMembro struct {
	ID            int64 `gorm:"primaryKey"`
	OrganizadorID int64
	UsuarioID     int64
	Papel         string
	CriadoEm      time.Time
}

func (OrganizadorMembro) TableName() string { return "organizador_membros" }

type MembroDetalhado struct {
	OrganizadorMembro
	Nome  string
	Email string
}

type OrganizadorMembroRepository struct {
	db *gorm.DB
}

func NovoOrganizadorMembroRepository(db *gorm.DB) *OrganizadorMembroRepository {
	return &OrganizadorMembroRepository{db: db}
}

func (r *OrganizadorMembroRepository) Adicionar(m *OrganizadorMembro) error {
	return r.db.Create(m).Error
}

func (r *OrganizadorMembroRepository) Existe(organizadorID, usuarioID int64) bool {
	var total int64
	r.db.Model(&OrganizadorMembro{}).Where("organizador_id = ? AND usuario_id = ?", organizadorID, usuarioID).Count(&total)
	return total > 0
}

func (r *OrganizadorMembroRepository) Remover(organizadorID, usuarioID int64) (bool, error) {
	res := r.db.Where("organizador_id = ? AND usuario_id = ?", organizadorID, usuarioID).Delete(&OrganizadorMembro{})
	return res.RowsAffected > 0, res.Error
}

// PrimeiroOrganizadorDoMembro devolve o organizador (id) do qual o usuário é
// membro — o mais antigo, se houver vários.
func (r *OrganizadorMembroRepository) PrimeiroOrganizadorDoMembro(usuarioID int64) (int64, bool) {
	var m OrganizadorMembro
	if err := r.db.Where("usuario_id = ?", usuarioID).Order("id").First(&m).Error; err != nil {
		return 0, false
	}
	return m.OrganizadorID, true
}

func (r *OrganizadorMembroRepository) Listar(organizadorID int64) ([]MembroDetalhado, error) {
	var lista []MembroDetalhado
	err := r.db.Table("organizador_membros").
		Select("organizador_membros.*, usuarios.nome as nome, usuarios.email as email").
		Joins("JOIN usuarios ON usuarios.id = organizador_membros.usuario_id").
		Where("organizador_membros.organizador_id = ?", organizadorID).
		Order("organizador_membros.id").Scan(&lista).Error
	return lista, err
}
