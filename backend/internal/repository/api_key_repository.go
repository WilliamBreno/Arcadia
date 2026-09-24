package repository

import (
	"time"

	"gorm.io/gorm"
)

type APIKey struct {
	ID            int64 `gorm:"primaryKey"`
	OrganizadorID int64
	Nome          string
	Prefixo       string
	Hash          string
	Ativo         bool
	CriadoEm      time.Time
	UltimoUsoEm   *time.Time
}

func (APIKey) TableName() string { return "api_keys" }

type APIKeyRepository struct {
	db *gorm.DB
}

func NovoAPIKeyRepository(db *gorm.DB) *APIKeyRepository { return &APIKeyRepository{db: db} }

func (r *APIKeyRepository) Criar(k *APIKey) error { return r.db.Create(k).Error }

func (r *APIKeyRepository) ListarPorOrganizador(organizadorID int64) ([]APIKey, error) {
	var lista []APIKey
	err := r.db.Where("organizador_id = ?", organizadorID).Order("id").Find(&lista).Error
	return lista, err
}

// BuscarAtivaPorHash localiza a chave pelo hash SHA-256 (a chave em texto
// nunca é guardada) e registra o uso.
func (r *APIKeyRepository) BuscarAtivaPorHash(hash string) (*APIKey, error) {
	var k APIKey
	if err := r.db.First(&k, "hash = ? AND ativo = true", hash).Error; err != nil {
		return nil, err
	}
	agora := time.Now()
	r.db.Model(&APIKey{}).Where("id = ?", k.ID).UpdateColumn("ultimo_uso_em", agora)
	return &k, nil
}

func (r *APIKeyRepository) Revogar(organizadorID, id int64) (bool, error) {
	res := r.db.Model(&APIKey{}).Where("id = ? AND organizador_id = ? AND ativo = true", id, organizadorID).Update("ativo", false)
	return res.RowsAffected > 0, res.Error
}
