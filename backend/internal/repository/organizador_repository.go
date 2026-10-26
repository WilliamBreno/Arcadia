package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type OrganizadorRepository struct {
	db *gorm.DB
}

func NovoOrganizadorRepository(db *gorm.DB) *OrganizadorRepository {
	return &OrganizadorRepository{db: db}
}

func (r *OrganizadorRepository) Criar(o *domain.Organizador) error {
	return r.db.Create(o).Error
}

func (r *OrganizadorRepository) Salvar(o *domain.Organizador) error {
	return r.db.Save(o).Error
}

func (r *OrganizadorRepository) BuscarPorUsuarioID(usuarioID int64) (*domain.Organizador, error) {
	var o domain.Organizador
	if err := r.db.First(&o, "usuario_id = ?", usuarioID).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizadorRepository) BuscarPorID(id int64) (*domain.Organizador, error) {
	var o domain.Organizador
	if err := r.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizadorRepository) BuscarPorSlug(slug string) (*domain.Organizador, error) {
	var o domain.Organizador
	if err := r.db.First(&o, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrganizadorRepository) SlugExiste(slug string) (bool, error) {
	var total int64
	if err := r.db.Model(&domain.Organizador{}).Where("slug = ?", slug).Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

// OrganizadorAdmin é a linha da lista de verificação do painel admin (item 5.4).
type OrganizadorAdmin struct {
	domain.Organizador
	UsuarioEmail  string
	Eventos       int64
	RepassesPagos int64
}

func (r *OrganizadorRepository) ListarAdmin(somentePendentes bool) ([]OrganizadorAdmin, error) {
	var linhas []OrganizadorAdmin
	q := r.db.Table("organizadores").
		Select(`organizadores.*, usuarios.email as usuario_email,
			(SELECT COUNT(*) FROM eventos WHERE eventos.organizador_id = organizadores.id) as eventos,
			(SELECT COUNT(*) FROM repasses WHERE repasses.organizador_id = organizadores.id AND repasses.status = 'pago') as repasses_pagos`).
		Joins("JOIN usuarios ON usuarios.id = organizadores.usuario_id").
		Order("organizadores.criado_em DESC")
	if somentePendentes {
		q = q.Where("organizadores.verificado_em IS NULL")
	}
	return linhas, q.Scan(&linhas).Error
}

// DefinirVerificacao marca ou desmarca o organizador como verificado.
func (r *OrganizadorRepository) DefinirVerificacao(id int64, adminID *int64, verificado bool) error {
	campos := map[string]any{"verificado_em": nil, "verificado_por": nil}
	if verificado {
		campos["verificado_em"] = time.Now()
		campos["verificado_por"] = adminID
	}
	return r.db.Model(&domain.Organizador{}).Where("id = ?", id).Updates(campos).Error
}
