package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// Descoberta (item 6.2): cidades, categorias, "Em alta" e coleções. Tudo
// só com eventos publicados, públicos e que ainda não terminaram.

// condFuturo: o evento ainda não terminou (fim, ou 12 h após o início sem
// fim — mesma regra de venda e check-in). Sem data, conta como futuro.
const condFuturo = "(eventos.inicio_em IS NULL OR COALESCE(eventos.fim_em, eventos.inicio_em + interval '12 hours') >= now())"

func (r *EventoRepository) publicosFuturos() *gorm.DB {
	return r.db.Model(&domain.Evento{}).
		Joins("LEFT JOIN locais ON locais.id = eventos.local_id").
		Where("eventos.status = ? AND eventos.visibilidade = ?", domain.StatusEventoPublicado, domain.VisibilidadePublico).
		Where(condFuturo)
}

func filtrarCidade(q *gorm.DB, cidade, uf string) *gorm.DB {
	if cidade != "" {
		q = q.Where("locais.cidade ILIKE ?", cidade)
	}
	if uf != "" {
		q = q.Where("locais.uf = ?", uf)
	}
	return q
}

// CidadeContagem: cidade com eventos futuros.
type CidadeContagem struct {
	Cidade  string
	UF      string
	Eventos int64
}

func (r *EventoRepository) ListarCidadesComEventos() ([]CidadeContagem, error) {
	var linhas []CidadeContagem
	err := r.publicosFuturos().
		Select("locais.cidade, locais.uf, COUNT(*) as eventos").
		Where("locais.cidade <> ''").
		Group("locais.cidade, locais.uf").
		Order("eventos DESC, locais.cidade").
		Scan(&linhas).Error
	return linhas, err
}

// CategoriaContagem: categoria com eventos futuros (opcionalmente numa cidade).
type CategoriaContagem struct {
	Categoria string
	Eventos   int64
}

func (r *EventoRepository) ListarCategoriasComEventos(cidade, uf string) ([]CategoriaContagem, error) {
	var linhas []CategoriaContagem
	err := filtrarCidade(r.publicosFuturos(), cidade, uf).
		Select("eventos.categoria, COUNT(*) as eventos").
		Where("eventos.categoria <> ''").
		Group("eventos.categoria").
		Order("eventos DESC, eventos.categoria").
		Scan(&linhas).Error
	return linhas, err
}

// EmAlta: mais ingressos vendidos (pagos ou usados, sem cortesia) desde `desde`.
func (r *EventoRepository) EmAlta(cidade, uf string, desde time.Time, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	vendas := r.db.Table("itens_pedido").
		Select("tipos_ingresso.evento_id, COUNT(*) as vendidos").
		Joins("JOIN tipos_ingresso ON tipos_ingresso.id = itens_pedido.tipo_ingresso_id").
		Where("itens_pedido.status IN ? AND itens_pedido.cortesia = false AND itens_pedido.ficha_id IS NULL AND itens_pedido.expositor_id IS NULL AND itens_pedido.criado_em >= ?",
			[]domain.StatusItemPedido{domain.StatusItemPago, domain.StatusItemUtilizado}, desde).
		Group("tipos_ingresso.evento_id")
	err := filtrarCidade(r.publicosFuturos(), cidade, uf).
		Select("eventos.*").
		Joins("JOIN (?) AS vendas ON vendas.evento_id = eventos.id", vendas).
		Order("vendas.vendidos DESC, eventos.inicio_em").
		Limit(limite).Find(&eventos).Error
	return eventos, err
}

// QuaseEsgotando: 80% ou mais das vagas ativas vendidas, mas ainda com vaga.
func (r *EventoRepository) QuaseEsgotando(cidade, uf string, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	ocupacao := r.db.Table("tipos_ingresso").
		Select(`tipos_ingresso.evento_id, SUM(tipos_ingresso.quantidade) as vagas,
			(SELECT COUNT(*) FROM itens_pedido ip JOIN tipos_ingresso t2 ON t2.id = ip.tipo_ingresso_id
			 WHERE t2.evento_id = tipos_ingresso.evento_id AND t2.ativo AND ip.status IN ('reservado','pago','utilizado')) as ocupadas`).
		Where("tipos_ingresso.ativo").
		Group("tipos_ingresso.evento_id")
	err := filtrarCidade(r.publicosFuturos(), cidade, uf).
		Select("eventos.*").
		Joins("JOIN (?) AS oc ON oc.evento_id = eventos.id", ocupacao).
		Where("oc.vagas > 0 AND oc.ocupadas < oc.vagas AND oc.ocupadas >= oc.vagas * 0.8").
		Order("(oc.vagas - oc.ocupadas) ASC, eventos.inicio_em").
		Limit(limite).Find(&eventos).Error
	return eventos, err
}

// Novos: publicados desde `desde`.
func (r *EventoRepository) Novos(cidade, uf string, desde time.Time, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	err := filtrarCidade(r.publicosFuturos(), cidade, uf).
		Select("eventos.*").
		Where("eventos.publicado_em >= ?", desde).
		Order("eventos.publicado_em DESC").
		Limit(limite).Find(&eventos).Error
	return eventos, err
}

// --- Coleções ---

func (r *EventoRepository) ListarColecoes(somenteAtivas bool) ([]domain.Colecao, error) {
	var cs []domain.Colecao
	q := r.db.Order("ordem, criado_em")
	if somenteAtivas {
		q = q.Where("ativa")
	}
	return cs, q.Find(&cs).Error
}

func (r *EventoRepository) BuscarColecao(slug string) (*domain.Colecao, error) {
	var c domain.Colecao
	if err := r.db.First(&c, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *EventoRepository) BuscarColecaoPorID(id int64) (*domain.Colecao, error) {
	var c domain.Colecao
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *EventoRepository) SalvarColecao(c *domain.Colecao) error { return r.db.Save(c).Error }

func (r *EventoRepository) ExcluirColecao(id int64) error {
	return r.db.Delete(&domain.Colecao{}, id).Error
}

func (r *EventoRepository) SlugColecaoExiste(slug string, excetoID int64) (bool, error) {
	var n int64
	err := r.db.Model(&domain.Colecao{}).Where("slug = ? AND id <> ?", slug, excetoID).Count(&n).Error
	return n > 0, err
}

// EventosDaColecao: só publicados, públicos e futuros (o admin pode ter
// incluído um evento que depois terminou ou foi cancelado — some sozinho).
func (r *EventoRepository) EventosDaColecao(colecaoID int64, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	err := r.publicosFuturos().
		Select("eventos.*").
		Joins("JOIN colecao_eventos ce ON ce.evento_id = eventos.id AND ce.colecao_id = ?", colecaoID).
		Order("ce.ordem, eventos.inicio_em").
		Limit(limite).Find(&eventos).Error
	return eventos, err
}

// SlugsDaColecao: todos os eventos incluídos (para o admin, sem filtro).
func (r *EventoRepository) SlugsDaColecao(colecaoID int64) ([]string, error) {
	var slugs []string
	err := r.db.Table("colecao_eventos").
		Select("eventos.slug").
		Joins("JOIN eventos ON eventos.id = colecao_eventos.evento_id").
		Where("colecao_eventos.colecao_id = ?", colecaoID).
		Order("colecao_eventos.ordem").
		Pluck("eventos.slug", &slugs).Error
	return slugs, err
}

// DefinirEventosDaColecao troca a lista inteira, na ordem dada.
func (r *EventoRepository) DefinirEventosDaColecao(colecaoID int64, eventoIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM colecao_eventos WHERE colecao_id = ?", colecaoID).Error; err != nil {
			return err
		}
		for i, id := range eventoIDs {
			if err := tx.Exec("INSERT INTO colecao_eventos (colecao_id, evento_id, ordem) VALUES (?, ?, ?) ON CONFLICT DO NOTHING", colecaoID, id, i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// IDsPorSlugs resolve slugs de eventos publicados (para montar a coleção).
func (r *EventoRepository) IDsPorSlugs(slugs []string) (map[string]int64, error) {
	var linhas []struct {
		ID   int64
		Slug string
	}
	err := r.db.Model(&domain.Evento{}).Select("id, slug").
		Where("slug IN ? AND status = ?", slugs, domain.StatusEventoPublicado).Scan(&linhas).Error
	m := make(map[string]int64, len(linhas))
	for _, l := range linhas {
		m[l.Slug] = l.ID
	}
	return m, err
}
