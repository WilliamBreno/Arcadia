package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type FiltrosEventoPublico struct {
	Busca            string
	Cidade           string
	UF               string
	Categoria        string
	OrganizadorID    *int64
	DataDe           *time.Time
	DataAte          *time.Time
	SomenteGratuitos bool
	Pagina           int
	PorPagina        int
	Ordenar          string // "data" (padrão) | "recentes"
}

// ListarPublicos lista eventos publicados e com visibilidade pública —
// usado pela home e pela busca (item 1.4). "não_listado" nunca aparece
// aqui, só é acessível pelo link direto (BuscarPublicoPorSlug).
func (r *EventoRepository) ListarPublicos(f FiltrosEventoPublico) ([]domain.Evento, int64, error) {
	base := r.db.Model(&domain.Evento{}).
		Joins("LEFT JOIN locais ON locais.id = eventos.local_id").
		Where("eventos.status = ?", domain.StatusEventoPublicado).
		Where("eventos.visibilidade = ?", domain.VisibilidadePublico)

	if f.Busca != "" {
		like := "%" + f.Busca + "%"
		base = base.Where("eventos.titulo ILIKE ? OR eventos.descricao ILIKE ?", like, like)
	}
	if f.Cidade != "" {
		base = base.Where("locais.cidade ILIKE ?", f.Cidade)
	}
	if f.UF != "" {
		base = base.Where("locais.uf = ?", strings.ToUpper(f.UF))
	}
	if f.Categoria != "" {
		base = base.Where("eventos.categoria = ?", f.Categoria)
	}
	if f.OrganizadorID != nil {
		base = base.Where("eventos.organizador_id = ?", *f.OrganizadorID)
	}
	if f.DataDe != nil {
		base = base.Where("eventos.inicio_em >= ?", f.DataDe)
	}
	if f.DataAte != nil {
		base = base.Where("eventos.inicio_em <= ?", f.DataAte)
	}
	if f.SomenteGratuitos {
		base = base.Where("EXISTS (SELECT 1 FROM tipos_ingresso ti WHERE ti.evento_id = eventos.id AND ti.ativo AND ti.preco_centavos = 0)")
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Distinct("eventos.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pagina := f.Pagina
	if pagina < 1 {
		pagina = 1
	}
	porPagina := f.PorPagina
	if porPagina < 1 || porPagina > 100 {
		porPagina = 20
	}

	ordenacao := "eventos.inicio_em ASC"
	if f.Ordenar == "recentes" {
		ordenacao = "eventos.criado_em DESC"
	}

	var eventos []domain.Evento
	err := base.Session(&gorm.Session{}).
		Select("eventos.*").
		Order(ordenacao).
		Limit(porPagina).
		Offset((pagina - 1) * porPagina).
		Find(&eventos).Error
	return eventos, total, err
}

// BuscarPublicoPorSlug aceita público e não_listado (quem tem o link),
// mas nunca privado nem rascunho/cancelado.
func (r *EventoRepository) BuscarPublicoPorSlug(slug string) (*domain.Evento, error) {
	var e domain.Evento
	err := r.db.
		Where("slug = ?", slug).
		Where("status = ?", domain.StatusEventoPublicado).
		Where("visibilidade IN ?", []domain.Visibilidade{domain.VisibilidadePublico, domain.VisibilidadeNaoListado}).
		First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventoRepository) ListarCategorias() ([]string, error) {
	var categorias []string
	err := r.db.Model(&domain.Evento{}).
		Where("status = ? AND categoria <> ''", domain.StatusEventoPublicado).
		Distinct().
		Order("categoria").
		Pluck("categoria", &categorias).Error
	return categorias, err
}
