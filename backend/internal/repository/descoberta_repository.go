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
		Where("itens_pedido.status IN ? AND itens_pedido.cortesia = false AND itens_pedido.ficha_id IS NULL AND itens_pedido.expositor_id IS NULL AND NOT itens_pedido.adicional AND itens_pedido.criado_em >= ?",
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

// exprGratuito: o evento tem entrada gratuita à venda (adicional não conta).
const exprGratuito = "EXISTS (SELECT 1 FROM tipos_ingresso ti WHERE ti.evento_id = eventos.id AND ti.ativo AND NOT ti.adicional AND ti.preco_centavos = 0)"

// EventosDaColecao: só publicados, públicos e futuros (o admin pode ter
// incluído um evento que depois terminou ou foi cancelado — some sozinho).
// Item 11.7: coleção automática segue a regra; a manual junta os eventos
// do admin e os marcados pelo organizador (menos os que o admin tirou).
func (r *EventoRepository) EventosDaColecao(col *domain.Colecao, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	q := r.publicosFuturos().Select("eventos.*")
	switch col.Regra {
	case domain.RegraColecaoGratuitos:
		q = q.Where(exprGratuito).Order("eventos.inicio_em")
	case domain.RegraColecaoEstaSemana:
		q = q.Where("eventos.inicio_em < now() + interval '7 days'").Order("eventos.inicio_em")
	case domain.RegraColecaoOnline:
		q = q.Where("eventos.local_id IS NULL").Order("eventos.inicio_em")
	case domain.RegraColecaoCategoria:
		q = q.Where("eventos.categoria = ?", col.RegraValor).Order("eventos.inicio_em")
	default:
		q = q.Joins("JOIN colecao_eventos ce ON ce.evento_id = eventos.id AND ce.colecao_id = ? AND NOT ce.bloqueado", col.ID).
			Order("ce.origem, ce.ordem, eventos.inicio_em")
	}
	err := q.Limit(limite).Find(&eventos).Error
	return eventos, err
}

// MarcadoPeloOrganizador é um evento que o organizador pôs num tema.
type MarcadoPeloOrganizador struct {
	Slug      string `json:"slug"`
	Titulo    string `json:"titulo"`
	Bloqueado bool   `json:"bloqueado"`
}

// MarcadosPeloOrganizador lista os eventos que organizadores puseram na coleção.
func (r *EventoRepository) MarcadosPeloOrganizador(colecaoID int64) ([]MarcadoPeloOrganizador, error) {
	lista := []MarcadoPeloOrganizador{}
	err := r.db.Table("colecao_eventos").Select("eventos.slug, eventos.titulo, colecao_eventos.bloqueado").
		Joins("JOIN eventos ON eventos.id = colecao_eventos.evento_id").
		Where("colecao_eventos.colecao_id = ? AND colecao_eventos.origem = 'organizador'", colecaoID).
		Order("colecao_eventos.criado_em DESC").Scan(&lista).Error
	return lista, err
}

// BloquearNaColecao tira (ou devolve) um evento marcado pelo organizador.
func (r *EventoRepository) BloquearNaColecao(colecaoID int64, slug string, bloqueado bool) (bool, error) {
	res := r.db.Exec(`UPDATE colecao_eventos SET bloqueado = ? WHERE colecao_id = ? AND origem = 'organizador'
		AND evento_id = (SELECT id FROM eventos WHERE slug = ?)`, bloqueado, colecaoID, slug)
	return res.RowsAffected > 0, res.Error
}

// TemaDoEvento é a presença do evento numa coleção.
type TemaDoEvento struct {
	ColecaoID int64
	Origem    string // admin | organizador
	Bloqueado bool
}

// TemasDoEvento: coleções em que o evento está (pelo admin ou pelo
// organizador, com o bloqueio do admin, se houver).
func (r *EventoRepository) TemasDoEvento(eventoID int64) (map[int64]TemaDoEvento, error) {
	var linhas []TemaDoEvento
	err := r.db.Table("colecao_eventos").Select("colecao_id, origem, bloqueado").
		Where("evento_id = ?", eventoID).Scan(&linhas).Error
	m := map[int64]TemaDoEvento{}
	for _, l := range linhas {
		m[l.ColecaoID] = l
	}
	return m, err
}

// DefinirTemasDoEvento troca os temas que o organizador marcou. Linhas
// bloqueadas pelo admin ficam como estão.
func (r *EventoRepository) DefinirTemasDoEvento(eventoID int64, colecaoIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Trava a linha do evento: dois salvamentos ao mesmo tempo não
		// somam temas (o último vence).
		if err := tx.Exec("SELECT id FROM eventos WHERE id = ? FOR UPDATE", eventoID).Error; err != nil {
			return err
		}
		apagar := "DELETE FROM colecao_eventos WHERE evento_id = ? AND origem = 'organizador' AND NOT bloqueado"
		args := []any{eventoID}
		if len(colecaoIDs) > 0 {
			apagar += " AND colecao_id NOT IN ?"
			args = append(args, colecaoIDs)
		}
		if err := tx.Exec(apagar, args...).Error; err != nil {
			return err
		}
		for _, id := range colecaoIDs {
			if err := tx.Exec(`INSERT INTO colecao_eventos (colecao_id, evento_id, ordem, origem)
				VALUES (?, ?, 1000, 'organizador') ON CONFLICT DO NOTHING`, id, eventoID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SlugsDaColecao: eventos que o admin incluiu (para o admin, sem filtro).
func (r *EventoRepository) SlugsDaColecao(colecaoID int64) ([]string, error) {
	var slugs []string
	err := r.db.Table("colecao_eventos").
		Select("eventos.slug").
		Joins("JOIN eventos ON eventos.id = colecao_eventos.evento_id").
		Where("colecao_eventos.colecao_id = ? AND colecao_eventos.origem = 'admin'", colecaoID).
		Order("colecao_eventos.ordem").
		Pluck("eventos.slug", &slugs).Error
	return slugs, err
}

// DefinirEventosDaColecao troca a lista do admin, na ordem dada. Os
// eventos marcados pelo organizador (item 11.7) não são tocados.
func (r *EventoRepository) DefinirEventosDaColecao(colecaoID int64, eventoIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM colecao_eventos WHERE colecao_id = ? AND origem = 'admin'", colecaoID).Error; err != nil {
			return err
		}
		for i, id := range eventoIDs {
			// Se o organizador já tinha marcado, a linha passa a ser do admin
			// (com a ordem dele e sem bloqueio).
			if err := tx.Exec(`INSERT INTO colecao_eventos (colecao_id, evento_id, ordem) VALUES (?, ?, ?)
				ON CONFLICT (colecao_id, evento_id) DO UPDATE SET origem = 'admin', ordem = EXCLUDED.ordem, bloqueado = false`, colecaoID, id, i).Error; err != nil {
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

// ---------- item 10.5: recomendações e feeds ----------

// TambemCompraram: eventos futuros comprados por quem comprou este, só com
// pelo menos `minimo` pessoas em comum (ninguém é identificado).
func (r *EventoRepository) TambemCompraram(eventoID int64, minimo, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	comuns := r.db.Raw(`SELECT p2.evento_id, count(DISTINCT p2.usuario_id) AS pessoas
		FROM pedidos p1 JOIN itens_pedido i1 ON i1.pedido_id = p1.id AND i1.status IN ('pago', 'utilizado') AND NOT i1.adicional
		JOIN pedidos p2 ON p2.usuario_id = p1.usuario_id AND p2.evento_id <> p1.evento_id AND p2.canal = 'online'
		JOIN itens_pedido i2 ON i2.pedido_id = p2.id AND i2.status IN ('pago', 'utilizado') AND NOT i2.adicional
		WHERE p1.evento_id = ? AND p1.canal = 'online' AND NOT i1.cortesia AND NOT i2.cortesia
		GROUP BY p2.evento_id HAVING count(DISTINCT p2.usuario_id) >= ?`, eventoID, minimo)
	err := r.publicosFuturos().Select("eventos.*").
		Joins("JOIN (?) AS comuns ON comuns.evento_id = eventos.id", comuns).
		Order("comuns.pessoas DESC, eventos.inicio_em").Limit(limite).Find(&eventos).Error
	return eventos, err
}

// Parecidos: mesma categoria e mesma cidade (sem a cidade, só a categoria).
func (r *EventoRepository) Parecidos(e *domain.Evento, cidade, uf string, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	q := filtrarCidade(r.publicosFuturos(), cidade, uf).Select("eventos.*").
		Where("eventos.id <> ? AND eventos.categoria = ?", e.ID, e.Categoria)
	err := q.Order("eventos.inicio_em").Limit(limite).Find(&eventos).Error
	return eventos, err
}

// ParaVoce: de quem a pessoa segue e das categorias e cidades em que ela
// já comprou, sem os eventos que ela já tem.
func (r *EventoRepository) ParaVoce(usuarioID int64, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	err := r.publicosFuturos().Select("eventos.*").
		Where(`NOT EXISTS (SELECT 1 FROM pedidos p JOIN itens_pedido i ON i.pedido_id = p.id
			WHERE p.evento_id = eventos.id AND p.usuario_id = ? AND i.status IN ('reservado', 'pago', 'utilizado'))`, usuarioID).
		Where(`eventos.organizador_id IN (SELECT CAST(alvo AS BIGINT) FROM seguindo WHERE usuario_id = ? AND tipo = 'organizador' AND alvo ~ '^[0-9]+$')
			OR EXISTS (SELECT 1 FROM pedidos p JOIN itens_pedido i ON i.pedido_id = p.id AND i.status IN ('pago', 'utilizado')
				JOIN eventos ev ON ev.id = p.evento_id LEFT JOIN locais lo ON lo.id = ev.local_id
				WHERE p.usuario_id = ? AND ev.categoria = eventos.categoria AND ev.categoria <> ''
				AND (lo.cidade IS NULL OR lo.cidade = locais.cidade))`, usuarioID, usuarioID).
		Order("eventos.inicio_em").Limit(limite).Find(&eventos).Error
	return eventos, err
}

// FeedOrganizador / FeedCidade: eventos públicos futuros para os feeds.
func (r *EventoRepository) FeedOrganizador(organizadorID int64, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	err := r.publicosFuturos().Select("eventos.*").Where("eventos.organizador_id = ?", organizadorID).
		Order("eventos.inicio_em").Limit(limite).Find(&eventos).Error
	return eventos, err
}

func (r *EventoRepository) FeedCidade(cidade, uf string, limite int) ([]domain.Evento, error) {
	var eventos []domain.Evento
	err := filtrarCidade(r.publicosFuturos(), cidade, uf).Select("eventos.*").
		Order("eventos.inicio_em").Limit(limite).Find(&eventos).Error
	return eventos, err
}
