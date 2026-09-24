package service

import (
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

const (
	folgaAntesSessao    = 3 * time.Hour  // portaria abre até 3h antes
	duracaoPadraoSessao = 12 * time.Hour // sem fim_em: vale até 12h após o início
)

func fimEfetivoSessao(s *domain.Sessao) time.Time {
	if s.FimEm != nil {
		return *s.FimEm
	}
	return s.InicioEm.Add(duracaoPadraoSessao)
}

// SessaoAtual devolve a sessão ATIVA em andamento (janela: 3h antes do início
// até o fim). Se duas se sobrepõem, a que começa mais perto de agora.
func SessaoAtual(sessoes []domain.Sessao, agora time.Time) *domain.Sessao {
	var melhor *domain.Sessao
	for i := range sessoes {
		s := &sessoes[i]
		if s.Status != domain.SessaoAtiva {
			continue
		}
		if agora.Before(s.InicioEm.Add(-folgaAntesSessao)) || agora.After(fimEfetivoSessao(s)) {
			continue
		}
		if melhor == nil || absDur(agora.Sub(s.InicioEm)) < absDur(agora.Sub(melhor.InicioEm)) {
			melhor = s
		}
	}
	return melhor
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// PeriodoDoEvento é o intervalo do evento derivado das sessões ativas
// (início da primeira, fim da última) — usado para manter evento.inicio_em/
// fim_em coerentes; o repasse sai depois do fim da última sessão.
func PeriodoDoEvento(sessoes []domain.Sessao) (inicio time.Time, fim time.Time, ok bool) {
	for i := range sessoes {
		s := &sessoes[i]
		if s.Status != domain.SessaoAtiva {
			continue
		}
		f := s.InicioEm
		if s.FimEm != nil {
			f = *s.FimEm
		}
		if !ok || s.InicioEm.Before(inicio) {
			inicio = s.InicioEm
		}
		if !ok || f.After(fim) {
			fim = f
		}
		ok = true
	}
	return
}
