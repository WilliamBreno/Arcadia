package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrOfflineIndisponivelRotativo = errors.New("o modo offline não está disponível em evento com QR rotativo (o QR muda a cada 30s e precisa do servidor)")
	ErrLoteOfflineInvalido         = errors.New("lote de entradas vazio ou grande demais (máx. 2000)")
)

// HashTokenQR é o SHA-256 (hex) do token estático. O pacote offline leva só
// o hash: o aparelho compara o hash do que leu, sem ter os tokens em claro.
func HashTokenQR(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type IngressoPacote struct {
	Codigo      string
	HashToken   string
	Nome        string
	TipoNome    string
	MeiaEntrada bool
	Status      string
	UtilizadoEm *time.Time
	SessaoIDs   []int64 // vazio = todas as sessões
}

type PacoteOffline struct {
	EventoID  int64
	GeradoEm  time.Time
	Sessoes   []domain.Sessao
	Ingressos []IngressoPacote
}

// Pacote monta os dados para validar sem internet (item da Fase 4).
func (s *CheckinService) Pacote(usuarioID, eventoID int64) (*PacoteOffline, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	evento, err := s.eventos.BuscarPorID(eventoID)
	if err != nil {
		return nil, err
	}
	if evento.QRRotativo {
		return nil, ErrOfflineIndisponivelRotativo
	}
	tipos, err := s.tiposIngresso.ListarPorEvento(eventoID)
	if err != nil {
		return nil, err
	}
	sessoesDoTipo := map[int64][]int64{}
	for _, t := range tipos {
		sessoesDoTipo[t.ID] = t.SessaoIDs
	}
	sessoes, err := s.sessoes.ListarPorEvento(eventoID)
	if err != nil {
		return nil, err
	}
	itens, err := s.itensPedido.ListarParaPacote(eventoID)
	if err != nil {
		return nil, err
	}
	pacote := &PacoteOffline{EventoID: eventoID, GeradoEm: time.Now(), Sessoes: sessoes, Ingressos: make([]IngressoPacote, 0, len(itens))}
	for _, i := range itens {
		pacote.Ingressos = append(pacote.Ingressos, IngressoPacote{
			Codigo: i.Codigo, HashToken: HashTokenQR(i.QRToken), Nome: i.TitularNome, TipoNome: i.TipoNome,
			MeiaEntrada: i.MeiaEntrada, Status: i.Status, UtilizadoEm: i.UtilizadoEm, SessaoIDs: sessoesDoTipo[i.TipoIngressoID],
		})
	}
	return pacote, nil
}

type EntradaOffline struct {
	EntradaID string
	Codigo    string
	SessaoID  *int64
	LidoEm    time.Time
}

type ResultadoOffline struct {
	EntradaID string
	Codigo    string
	Resultado string // aceito | conflito | cancelado | nao_encontrado | sessao_invalida
}

// Sincronizar aplica as entradas lidas offline. Idempotente por entrada_id.
// O QR já foi conferido no aparelho (hash); aqui o servidor só decide quem
// "chegou primeiro": UPDATE atômico (evento comum) ou UNIQUE item+sessão.
func (s *CheckinService) Sincronizar(usuarioID, eventoID int64, entradas []EntradaOffline) ([]ResultadoOffline, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	if len(entradas) == 0 || len(entradas) > 2000 {
		return nil, ErrLoteOfflineInvalido
	}

	sessoes, err := s.sessoes.ListarPorEvento(eventoID)
	if err != nil {
		return nil, err
	}
	sessaoPorID := map[int64]*domain.Sessao{}
	for i := range sessoes {
		sessaoPorID[sessoes[i].ID] = &sessoes[i]
	}

	resultados := make([]ResultadoOffline, 0, len(entradas))
	for _, e := range entradas {
		if anterior, ok := s.itensPedido.BuscarOfflinePorEntrada(eventoID, e.EntradaID); ok {
			resultados = append(resultados, ResultadoOffline{e.EntradaID, e.Codigo, anterior})
			continue
		}

		resultado, itemID := s.aplicarEntradaOffline(eventoID, e, sessaoPorID, len(sessoes) > 0)
		if err := s.itensPedido.GravarOffline(eventoID, repository.CheckinOffline{
			EntradaID: e.EntradaID, ItemID: itemID, Codigo: e.Codigo, SessaoID: e.SessaoID,
			StaffUsuarioID: usuarioID, LidoEm: e.LidoEm, Resultado: resultado,
		}); err != nil {
			return resultados, err
		}
		resultados = append(resultados, ResultadoOffline{e.EntradaID, e.Codigo, resultado})
	}
	return resultados, nil
}

func (s *CheckinService) aplicarEntradaOffline(eventoID int64, e EntradaOffline, sessaoPorID map[int64]*domain.Sessao, temSessoes bool) (string, *int64) {
	item, err := s.itensPedido.BuscarPorCodigo(e.Codigo)
	if err != nil {
		return "nao_encontrado", nil
	}
	tipo, err := s.tiposIngresso.BuscarPorID(item.TipoIngressoID)
	if err != nil || tipo.EventoID != eventoID {
		return "nao_encontrado", nil
	}
	id := item.ID
	if item.Status != domain.StatusItemPago && item.Status != domain.StatusItemUtilizado {
		return "cancelado", &id
	}

	if temSessoes {
		sessao := (*domain.Sessao)(nil)
		if e.SessaoID != nil {
			sessao = sessaoPorID[*e.SessaoID]
		}
		if sessao == nil || sessao.Status != domain.SessaoAtiva || !tipoValeNaSessao(tipo.SessaoIDs, sessao.ID) {
			return "sessao_invalida", &id
		}
		inseriu, err := s.sessoes.RegistrarCheckin(item.ID, sessao.ID)
		if err != nil || !inseriu {
			return "conflito", &id
		}
		if item.Status == domain.StatusItemPago {
			_, _ = s.itensPedido.MarcarUtilizadoAtomicoEm(item.ID, e.LidoEm)
		}
		return "aceito", &id
	}

	marcou, err := s.itensPedido.MarcarUtilizadoAtomicoEm(item.ID, e.LidoEm)
	if err != nil || !marcou {
		return "conflito", &id // já tinha entrado (online ou em outro aparelho)
	}
	return "aceito", &id
}

func (s *CheckinService) Conflitos(usuarioID, eventoID int64) ([]repository.ConflitoOffline, error) {
	if !s.TemAcesso(usuarioID, eventoID) {
		return nil, ErrSemAcessoCheckin
	}
	return s.itensPedido.ListarConflitosOffline(eventoID)
}
