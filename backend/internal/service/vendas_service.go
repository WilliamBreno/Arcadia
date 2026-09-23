package service

import (
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

type ResumoVendasTipo struct {
	TipoIngressoID   int64
	TipoIngressoNome string
	Quantidade       int
	ReceitaCentavos  int64
}

type ResumoVendas struct {
	TotalVendido    int
	ReceitaCentavos int64
	PorTipo         []ResumoVendasTipo
}

type VendasEResumo struct {
	Resumo ResumoVendas
	Itens  []repository.ItemVenda
}

// VendasService é o "painel básico" da seção 1.11: quantidade vendida e
// receita bruta (preço do ingresso, sem taxa da plataforma) por evento e
// por tipo de ingresso. O detalhamento financeiro completo (taxa do
// processador, líquido, repasses) fica para o painel financeiro da
// Fase 2 (item 2.2).
type VendasService struct {
	eventos     *repository.EventoRepository
	itensPedido *repository.ItemPedidoRepository
}

func NovoVendasService(eventos *repository.EventoRepository, itensPedido *repository.ItemPedidoRepository) *VendasService {
	return &VendasService{eventos: eventos, itensPedido: itensPedido}
}

func (s *VendasService) Listar(organizadorID, eventoID int64) (*VendasEResumo, error) {
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, err
	}
	if evento.OrganizadorID != organizadorID {
		return nil, ErrEventoNaoPertenceAoOrganizador
	}

	itens, err := s.itensPedido.ListarVendasPorEvento(eventoID)
	if err != nil {
		return nil, err
	}

	porTipo := map[int64]*ResumoVendasTipo{}
	var ordem []int64
	resumo := ResumoVendas{}
	for _, item := range itens {
		resumo.TotalVendido++
		resumo.ReceitaCentavos += item.PrecoCentavos

		linha, ok := porTipo[item.TipoIngressoID]
		if !ok {
			linha = &ResumoVendasTipo{TipoIngressoID: item.TipoIngressoID, TipoIngressoNome: item.TipoIngressoNome}
			porTipo[item.TipoIngressoID] = linha
			ordem = append(ordem, item.TipoIngressoID)
		}
		linha.Quantidade++
		linha.ReceitaCentavos += item.PrecoCentavos
	}
	for _, id := range ordem {
		resumo.PorTipo = append(resumo.PorTipo, *porTipo[id])
	}

	return &VendasEResumo{Resumo: resumo, Itens: itens}, nil
}
