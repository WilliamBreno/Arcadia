package service

import (
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// validarComSessoes: ingresso do evento todo entra uma vez em cada sessão;
// tipo restrito a uma sessão só entra nela. Fora da janela de qualquer
// sessão ativa: fora_da_sessao.
func (s *CheckinService) validarComSessoes(item *domain.ItemPedido, tipo *domain.TipoIngresso, sessoes []domain.Sessao) (*ResultadoCheckin, error) {
	base := ResultadoCheckin{Item: item, TipoIngressoNome: tipo.Nome, MeiaEntrada: tipo.MeiaEntrada}

	if item.Status != domain.StatusItemPago && item.Status != domain.StatusItemUtilizado {
		r := base
		r.Resultado = ResultadoCancelado
		return &r, nil
	}
	agora := time.Now()
	atual := SessaoAtual(sessoes, agora)
	if atual == nil || !tipoValeNaSessao(tipo.SessaoIDs, atual.ID) {
		r := base
		r.Resultado = ResultadoForaDaSessao
		return &r, nil
	}

	inseriu, err := s.sessoes.RegistrarCheckin(item.ID, atual.ID)
	if err != nil {
		return nil, err
	}
	if !inseriu {
		r := base
		r.Resultado = ResultadoJaUtilizado
		if quando, errH := s.sessoes.HoraCheckin(item.ID, atual.ID); errH == nil {
			copia := *item
			copia.UtilizadoEm = &quando
			r.Item = &copia
		}
		return &r, nil
	}
	if item.Status == domain.StatusItemPago {
		_, _ = s.itensPedido.MarcarUtilizadoAtomico(item.ID)
	}
	copia := *item
	copia.UtilizadoEm = &agora
	r := base
	r.Resultado, r.Item = ResultadoValido, &copia
	return &r, nil
}
