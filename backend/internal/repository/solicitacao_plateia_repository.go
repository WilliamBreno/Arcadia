package repository

import (
	"gorm.io/gorm"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

type SolicitacaoPlateiaRepository struct {
	db *gorm.DB
}

func NovoSolicitacaoPlateiaRepository(db *gorm.DB) *SolicitacaoPlateiaRepository {
	return &SolicitacaoPlateiaRepository{db: db}
}

func (r *SolicitacaoPlateiaRepository) Criar(s *domain.SolicitacaoPlateia) error {
	return r.db.Create(s).Error
}

func (r *SolicitacaoPlateiaRepository) Salvar(s *domain.SolicitacaoPlateia) error {
	return r.db.Save(s).Error
}

func (r *SolicitacaoPlateiaRepository) BuscarPorID(id int64) (*domain.SolicitacaoPlateia, error) {
	var s domain.SolicitacaoPlateia
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SolicitacaoPlateiaRepository) BuscarPorEventoEUsuario(eventoID, usuarioID int64) (*domain.SolicitacaoPlateia, error) {
	var s domain.SolicitacaoPlateia
	if err := r.db.First(&s, "evento_id = ? AND usuario_id = ?", eventoID, usuarioID).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

type SolicitacaoDetalhada struct {
	domain.SolicitacaoPlateia
	UsuarioNome  string
	UsuarioEmail string
}

func (r *SolicitacaoPlateiaRepository) ListarPorEvento(eventoID int64) ([]SolicitacaoDetalhada, error) {
	var linhas []SolicitacaoDetalhada
	err := r.db.Table("solicitacoes_plateia").
		Select("solicitacoes_plateia.*, usuarios.nome as usuario_nome, usuarios.email as usuario_email").
		Joins("JOIN usuarios ON usuarios.id = solicitacoes_plateia.usuario_id").
		Where("solicitacoes_plateia.evento_id = ?", eventoID).
		Order("solicitacoes_plateia.id").
		Scan(&linhas).Error
	return linhas, err
}

// ListarListaEspera devolve a fila na ordem em que o organizador aprovou.
func (r *SolicitacaoPlateiaRepository) ListarListaEspera(eventoID int64) ([]domain.SolicitacaoPlateia, error) {
	var lista []domain.SolicitacaoPlateia
	err := r.db.Where("evento_id = ? AND status = ?", eventoID, domain.SolicitacaoListaEspera).
		Order("decidido_em, id").Find(&lista).Error
	return lista, err
}

// ContarAprovadasSemCompra são as vagas "prometidas": aprovadas que ainda
// não têm nenhum item reservado/pago/utilizado no evento.
func (r *SolicitacaoPlateiaRepository) ContarAprovadasSemCompra(eventoID int64) (int64, error) {
	var total int64
	err := r.db.Raw(`SELECT COUNT(*) FROM solicitacoes_plateia s
		WHERE s.evento_id = ? AND s.status = 'aprovada'
		AND NOT EXISTS (
			SELECT 1 FROM itens_pedido ip
			JOIN pedidos p ON p.id = ip.pedido_id
			WHERE p.usuario_id = s.usuario_id AND p.evento_id = s.evento_id
			AND ip.cortesia = false AND ip.status IN ('reservado', 'pago', 'utilizado')
		)`, eventoID).Scan(&total).Error
	return total, err
}

// VagasLivres soma, nos tipos ativos, quantidade menos itens que ocupam vaga.
func (r *SolicitacaoPlateiaRepository) VagasLivres(eventoID int64) (int64, error) {
	var livres int64
	err := r.db.Raw(`SELECT COALESCE(SUM(GREATEST(t.quantidade - (
			SELECT COUNT(*) FROM itens_pedido ip
			WHERE ip.tipo_ingresso_id = t.id AND ip.status IN ('reservado', 'pago', 'utilizado')
		), 0)), 0) FROM tipos_ingresso t WHERE t.evento_id = ? AND t.ativo`, eventoID).Scan(&livres).Error
	return livres, err
}
