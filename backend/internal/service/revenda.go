package service

import (
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// Revenda oficial (item 7.4), ligada à lista de espera do item 7.3.
//
// O dono de um ingresso pago põe o ingresso à venda (só quando aquele tipo
// esgotou). O anúncio vira uma vaga a mais à venda; se há fila, a vaga é
// guardada para a primeira pessoa (7.3). O comprador compra normalmente,
// pelo preço de tabela e com a taxa normal. Quando o pagamento dele é
// aprovado, o ingresso do vendedor deixa de valer na hora e o vendedor
// recebe de volta o preço que pagou pelo ingresso (sem taxa extra; a taxa
// de serviço e a garantia da compra dele não voltam), pelo estorno do
// próprio pagamento, com nova tentativa se o Mercado Pago falhar.
// O organizador não perde nem ganha: a venda nova substitui a antiga.

const (
	RevendaAVenda    = "a_venda"
	RevendaReservada = "reservada"
	RevendaVendida   = "vendida"
	RevendaRetirada  = "retirada"
	RevendaEncerrada = "encerrada"

	MotivoRevendido = "Revendido na revenda oficial"
)

var (
	ErrRevendaNaoPermitida  = errors.New("este ingresso não pode ser revendido")
	ErrRevendaSemEsgotar    = errors.New("a revenda só abre quando este ingresso esgota; enquanto houver ingresso à venda, quem quiser compra direto")
	ErrRevendaJaAnunciada   = errors.New("este ingresso já está na revenda")
	ErrRevendaEmAndamento   = errors.New("alguém está pagando por este ingresso na revenda agora; aguarde alguns minutos")
	ErrRevendaNaoEncontrada = errors.New("este ingresso não está na revenda")
)

type Revenda struct {
	ID                    int64 `gorm:"primaryKey"`
	ItemID                int64
	EventoID              int64
	TipoIngressoID        int64
	VendedorID            int64
	ValorDevolverCentavos int64
	Status                string
	ItemCompradorID       *int64
	VendidaEm             *time.Time
	CriadoEm              time.Time
	AtualizadoEm          time.Time
}

func (Revenda) TableName() string { return "revendas" }

// --- Estoque: anúncios contam como vaga à venda ---

// RevendasAVenda conta os anúncios livres de um tipo.
func RevendasAVenda(tx *gorm.DB, tipoID int64) (int64, error) {
	var n int64
	err := tx.Model(&Revenda{}).Where("tipo_ingresso_id = ? AND status = ?", tipoID, RevendaAVenda).Count(&n).Error
	return n, err
}

// RevendasAVendaPorTipos é a versão da página pública.
func RevendasAVendaPorTipos(db *gorm.DB, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var linhas []struct {
		TipoIngressoID int64
		Total          int64
	}
	err := db.Model(&Revenda{}).Select("tipo_ingresso_id, COUNT(*) AS total").
		Where("tipo_ingresso_id IN ? AND status = ?", ids, RevendaAVenda).Group("tipo_ingresso_id").Scan(&linhas).Error
	for _, l := range linhas {
		out[l.TipoIngressoID] = l.Total
	}
	return out, err
}

// VagasParaComprar: o que alguém pode comprar agora = estoque livre do
// organizador + ingressos na revenda − vagas guardadas para a fila (de
// outras pessoas). Testada.
func VagasParaComprar(livresOrganizador, naRevenda, guardadas int64) int64 {
	return max(max(livresOrganizador, 0)+naRevenda-guardadas, 0)
}

// QuantasDaRevenda: de uma compra de `quantidade`, quantas saem da revenda
// (o estoque do organizador vende primeiro). Testada.
func QuantasDaRevenda(quantidade int, estoqueOrganizador int64) int {
	if estoqueOrganizador < 0 {
		estoqueOrganizador = 0
	}
	if int64(quantidade) <= estoqueOrganizador {
		return 0
	}
	return quantidade - int(estoqueOrganizador)
}

// reivindicarRevendas liga itens novos do comprador aos anúncios mais
// antigos do tipo (travados, sem esperar quem já está travado).
func reivindicarRevendas(tx *gorm.DB, tipoID int64, itensComprador []int64) error {
	if len(itensComprador) == 0 {
		return nil
	}
	var anuncios []Revenda
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("tipo_ingresso_id = ? AND status = ?", tipoID, RevendaAVenda).
		Order("criado_em, id").Limit(len(itensComprador)).Find(&anuncios).Error; err != nil {
		return err
	}
	if len(anuncios) < len(itensComprador) {
		return ErrEstoqueInsuficiente
	}
	agora := time.Now()
	for i, a := range anuncios {
		if err := tx.Model(&Revenda{}).Where("id = ?", a.ID).
			Updates(map[string]any{"status": RevendaReservada, "item_comprador_id": itensComprador[i], "atualizado_em": agora}).Error; err != nil {
			return err
		}
	}
	return nil
}

// liberarRevendas devolve à venda os anúncios de compradores cuja reserva
// expirou.
func liberarRevendas(tx *gorm.DB, itensComprador []int64) error {
	if len(itensComprador) == 0 {
		return nil
	}
	return tx.Model(&Revenda{}).Where("item_comprador_id IN ? AND status = ?", itensComprador, RevendaReservada).
		Updates(map[string]any{"status": RevendaAVenda, "item_comprador_id": nil, "atualizado_em": time.Now()}).Error
}

// fecharVendas: o pagamento do comprador foi aprovado. Marca os anúncios
// como vendidos e invalida na hora os ingressos dos vendedores (o QR antigo
// para de entrar). Devolve os anúncios para o estorno, que sai depois.
func fecharVendas(tx *gorm.DB, itensComprador []int64) ([]int64, error) {
	if len(itensComprador) == 0 {
		return nil, nil
	}
	var anuncios []Revenda
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("item_comprador_id IN ? AND status = ?", itensComprador, RevendaReservada).Find(&anuncios).Error; err != nil {
		return nil, err
	}
	agora := time.Now()
	ids := make([]int64, 0, len(anuncios))
	for _, a := range anuncios {
		if err := tx.Model(&Revenda{}).Where("id = ?", a.ID).
			Updates(map[string]any{"status": RevendaVendida, "vendida_em": agora, "atualizado_em": agora}).Error; err != nil {
			return nil, err
		}
		res := tx.Model(&domain.ItemPedido{}).Where("id = ? AND status = ?", a.ItemID, domain.StatusItemPago).
			Updates(map[string]any{"status": domain.StatusItemCancelado, "cancelado_em": agora, "motivo_cancelamento": MotivoRevendido})
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			// O vendedor entrou no evento antes (check-in offline): não há o
			// que cancelar nem devolver; fica registrado para o admin.
			slog.Error("revenda: ingresso do vendedor não estava pago ao fechar a venda", "revenda_id", a.ID, "item_id", a.ItemID)
			continue
		}
		ids = append(ids, a.ID)
	}
	return ids, nil
}

// --- Serviço ---

type RevendaService struct {
	db           *gorm.DB
	itens        *repository.ItemPedidoRepository
	cancelamento *CancelamentoService
	fila         *FilaEsperaService
}

func NovoRevendaService(db *gorm.DB, itens *repository.ItemPedidoRepository, cancelamento *CancelamentoService, fila *FilaEsperaService) *RevendaService {
	return &RevendaService{db: db, itens: itens, cancelamento: cancelamento, fila: fila}
}

// PodeRevender confere as regras do item (sem olhar o estoque). Testada.
func PodeRevender(item *domain.ItemPedido, evento *domain.Evento, agora time.Time) error {
	switch {
	case item.Status != domain.StatusItemPago || item.FichaID != nil: // 8.4: inscrição não se revende
		return ErrRevendaNaoPermitida
	case item.Cortesia || item.PrecoCentavos <= 0:
		return errors.Join(ErrRevendaNaoPermitida, errors.New("ingresso gratuito ou cortesia: é só cancelar a inscrição"))
	case evento.Status != domain.StatusEventoPublicado:
		return ErrRevendaNaoPermitida
	case evento.InicioEm != nil && !agora.Before(*evento.InicioEm):
		return errors.Join(ErrRevendaNaoPermitida, errors.New("a revenda fecha no início do evento"))
	}
	return nil
}

// Anunciar põe o ingresso à venda.
func (s *RevendaService) Anunciar(usuarioID, itemID int64) (*Revenda, error) {
	var anuncio *Revenda
	var eventoID int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var item domain.ItemPedido
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, itemID).Error; err != nil {
			return ErrItemNaoEncontrado
		}
		var pedido domain.Pedido
		if err := tx.First(&pedido, item.PedidoID).Error; err != nil || pedido.UsuarioID != usuarioID {
			return ErrItemNaoPertenceAoUsuario
		}
		var evento domain.Evento
		if err := tx.First(&evento, pedido.EventoID).Error; err != nil {
			return ErrItemNaoEncontrado
		}
		if err := PodeRevender(&item, &evento, time.Now()); err != nil {
			return err
		}
		tipo, err := s.itens.BuscarTipoIngressoParaAtualizar(tx, item.TipoIngressoID)
		if err != nil {
			return err
		}
		ativos, err := s.itens.ContarAtivosPorTipo(tx, tipo.ID)
		if err != nil {
			return err
		}
		guardadas, err := VagasGuardadas(tx, tipo.ID, 0)
		if err != nil {
			return err
		}
		if int64(tipo.Quantidade)-ativos-guardadas > 0 {
			return ErrRevendaSemEsgotar
		}
		var ja int64
		if err := tx.Model(&Revenda{}).Where("item_id = ? AND status IN ?", item.ID, []string{RevendaAVenda, RevendaReservada}).Count(&ja).Error; err != nil {
			return err
		}
		if ja > 0 {
			return ErrRevendaJaAnunciada
		}
		agora := time.Now()
		anuncio = &Revenda{
			ItemID: item.ID, EventoID: evento.ID, TipoIngressoID: tipo.ID, VendedorID: usuarioID,
			ValorDevolverCentavos: item.PrecoCentavos, Status: RevendaAVenda, CriadoEm: agora, AtualizadoEm: agora,
		}
		eventoID = evento.ID
		return tx.Create(anuncio).Error
	})
	if err != nil {
		return nil, err
	}
	// Se há fila de espera, a vaga é guardada para a primeira pessoa (7.3).
	if s.fila != nil {
		s.fila.PromoverListaEspera(eventoID)
	}
	return anuncio, nil
}

// Retirar tira o ingresso da revenda (se ninguém estiver pagando por ele).
func (s *RevendaService) Retirar(usuarioID, itemID int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var a Revenda
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("item_id = ? AND vendedor_id = ? AND status IN ?", itemID, usuarioID, []string{RevendaAVenda, RevendaReservada}).First(&a).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRevendaNaoEncontrada
		}
		if err != nil {
			return err
		}
		if a.Status == RevendaReservada {
			return ErrRevendaEmAndamento
		}
		return tx.Model(&a).Updates(map[string]any{"status": RevendaRetirada, "atualizado_em": time.Now()}).Error
	})
}

// AnuncioAtivo informa se o item está na revenda (para bloquear cancelar,
// transferir e o check-in enquanto alguém paga por ele).
func AnuncioAtivo(db *gorm.DB, itemID int64) (*Revenda, error) {
	var a Revenda
	err := db.Where("item_id = ? AND status IN ?", itemID, []string{RevendaAVenda, RevendaReservada}).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

// retirarSeLivre: o dono vai usar o ingresso (entrar, cancelar). Se o
// anúncio está livre, sai da revenda; se alguém está pagando, bloqueia.
func retirarSeLivre(db *gorm.DB, itemID int64) error {
	a, err := AnuncioAtivo(db, itemID)
	if err != nil || a == nil {
		return err
	}
	res := db.Model(&Revenda{}).Where("id = ? AND status = ?", a.ID, RevendaAVenda).
		Updates(map[string]any{"status": RevendaRetirada, "atualizado_em": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrRevendaEmAndamento
	}
	return nil
}

// MinhaRevenda é o que "Meus ingressos" mostra.
type MinhaRevenda struct {
	ItemID int64  `json:"item_id"`
	Status string `json:"status"`
	Valor  int64  `json:"valor_devolver_centavos"`
}

func (s *RevendaService) Minhas(usuarioID int64) ([]MinhaRevenda, error) {
	var out []MinhaRevenda
	err := s.db.Model(&Revenda{}).Select("item_id, status, valor_devolver_centavos AS valor").
		Where("vendedor_id = ? AND status IN ?", usuarioID, []string{RevendaAVenda, RevendaReservada, RevendaVendida}).
		Order("id").Scan(&out).Error
	if out == nil {
		out = []MinhaRevenda{}
	}
	return out, err
}

// Liquidar devolve ao vendedor o preço do ingresso revendido, pelo estorno
// do pagamento dele (fila de reembolsos: se o MP falhar, o job tenta de novo).
func (s *RevendaService) Liquidar(revendaIDs []int64) {
	for _, id := range revendaIDs {
		var a Revenda
		if err := s.db.First(&a, id).Error; err != nil {
			continue
		}
		if err := s.cancelamento.devolverRevenda(&a); err != nil {
			slog.Warn("revenda: estorno ao vendedor ficou para nova tentativa", "revenda_id", a.ID, "erro", err)
		}
	}
}

// Encerrar fecha os anúncios livres de eventos que começaram ou foram
// cancelados (o ingresso continua valendo com o dono). Roda no job.
func (s *RevendaService) Encerrar(agora time.Time) (int64, error) {
	res := s.db.Exec(`UPDATE revendas r SET status = ?, atualizado_em = ? FROM eventos e
		WHERE r.evento_id = e.id AND r.status = ? AND (e.status <> ? OR e.inicio_em <= ?)`,
		RevendaEncerrada, agora, RevendaAVenda, domain.StatusEventoPublicado, agora)
	return res.RowsAffected, res.Error
}
