package repository

import "github.com/WilliamBreno/Arcadia/backend/internal/domain"

func (r *ReembolsoRepository) BuscarPorID(id int64) (*domain.Reembolso, error) {
	var reembolso domain.Reembolso
	if err := r.db.First(&reembolso, id).Error; err != nil {
		return nil, err
	}
	return &reembolso, nil
}

type ReembolsoDetalhado struct {
	domain.Reembolso
	EventoTitulo string
	Codigo       string
	TitularNome  string
}

func (r *ReembolsoRepository) ListarDetalhados(status string) ([]ReembolsoDetalhado, error) {
	var linhas []ReembolsoDetalhado
	q := r.db.Table("reembolsos").
		Select("reembolsos.*, eventos.titulo as evento_titulo, itens_pedido.codigo as codigo, itens_pedido.titular_nome as titular_nome").
		Joins("JOIN itens_pedido ON itens_pedido.id = reembolsos.item_id").
		Joins("JOIN pedidos ON pedidos.id = itens_pedido.pedido_id").
		Joins("JOIN eventos ON eventos.id = pedidos.evento_id")
	if status != "" {
		q = q.Where("reembolsos.status = ?", status)
	}
	err := q.Order("reembolsos.criado_em DESC").Scan(&linhas).Error
	return linhas, err
}

type ResumoReembolso struct {
	Tipo          string
	Status        string
	Quantidade    int64
	ValorCentavos int64
}

func (r *ReembolsoRepository) Resumo() ([]ResumoReembolso, error) {
	var linhas []ResumoReembolso
	err := r.db.Table("reembolsos").
		Select("tipo, status, COUNT(*) as quantidade, COALESCE(SUM(valor_centavos),0) as valor_centavos").
		Group("tipo, status").Scan(&linhas).Error
	return linhas, err
}
