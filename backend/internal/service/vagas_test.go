package service

import (
	"errors"
	"testing"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func ptr(n int) *int { return &n }

func TestConferirVagas(t *testing.T) {
	cos := domain.TipoApresentacaoCosplay
	dan := domain.TipoApresentacaoDanca
	v := VagasEvento{Jurados: ptr(3), Participantes: ptr(10), Cosplay: ptr(4)}
	o := OcupacaoVagas{Jurados: 2, Participantes: 9, PorTipo: map[domain.TipoApresentacao]int{cos: 3, dan: 6}}

	casos := []struct {
		nome   string
		o      OcupacaoVagas
		papel  domain.Papel
		tipo   *domain.TipoApresentacao
		troca  bool
		espera error
	}{
		{"jurado com vaga", o, domain.PapelJurado, nil, false, nil},
		{"jurado lotado", OcupacaoVagas{Jurados: 3}, domain.PapelJurado, nil, false, ErrVagasJurados},
		{"participante com vaga no total e na modalidade", o, domain.PapelParticipante, &cos, false, nil},
		{"participante sem modalidade", o, domain.PapelParticipante, nil, false, nil},
		{"modalidade sem limite", o, domain.PapelParticipante, &dan, false, nil},
		{"total lotado", OcupacaoVagas{Participantes: 10, PorTipo: map[domain.TipoApresentacao]int{}}, domain.PapelParticipante, &dan, false, ErrVagasParticipantes},
		{"modalidade lotada", OcupacaoVagas{Participantes: 5, PorTipo: map[domain.TipoApresentacao]int{cos: 4}}, domain.PapelParticipante, &cos, false, ErrVagasTipo},
		{"troca de modalidade com total cheio passa", OcupacaoVagas{Participantes: 10, PorTipo: map[domain.TipoApresentacao]int{cos: 1}}, domain.PapelParticipante, &cos, true, nil},
		{"troca para modalidade lotada", OcupacaoVagas{Participantes: 10, PorTipo: map[domain.TipoApresentacao]int{cos: 4}}, domain.PapelParticipante, &cos, true, ErrVagasTipo},
	}
	for _, c := range casos {
		err := conferirVagas(v, c.o, c.papel, c.tipo, c.troca)
		if !errors.Is(err, c.espera) && !(err == nil && c.espera == nil) {
			t.Errorf("%s: obtido %v, esperado %v", c.nome, err, c.espera)
		}
	}
	if err := conferirVagas(VagasEvento{}, OcupacaoVagas{Jurados: 999, Participantes: 999}, domain.PapelJurado, nil, false); err != nil {
		t.Errorf("sem limite deveria caber: %v", err)
	}
	if !OcupaVaga(domain.StatusFichaPendente) || OcupaVaga(domain.StatusFichaListaEspera) || OcupaVaga(domain.StatusFichaRejeitado) {
		t.Error("OcupaVaga errado")
	}
}
