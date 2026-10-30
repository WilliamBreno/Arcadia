package service

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestOrdenarVotosComEmpate(t *testing.T) {
	ls := OrdenarVotos([]LinhaVoto{{Nome: "C", Votos: 3}, {Nome: "A", Votos: 5}, {Nome: "B", Votos: 3}, {Nome: "D", Votos: 0}})
	got := ""
	for _, l := range ls {
		got += l.Nome + string(rune('0'+l.Posicao))
	}
	if got != "A1B2C2D4" {
		t.Errorf("ordem/posição: %s", got)
	}
}

func TestVisibilidadeDaVotacao(t *testing.T) {
	agora := time.Now()
	e := &domain.Evento{}
	if VotacaoAberta(e) || PremioPublicoVisivel(e) {
		t.Error("nada aberto no começo")
	}
	e.VotacaoAbertaEm = &agora
	if !VotacaoAberta(e) {
		t.Error("aberta")
	}
	e.ResultadoLiberadoEm = &agora
	if PremioPublicoVisivel(e) {
		t.Error("com a votação aberta, a contagem não aparece")
	}
	e.VotacaoFechadaEm = &agora
	if VotacaoAberta(e) || !PremioPublicoVisivel(e) {
		t.Error("fechada e com resultado liberado: aparece")
	}
	e.ResultadoLiberadoEm = nil
	if PremioPublicoVisivel(e) {
		t.Error("sem resultado liberado, não aparece")
	}
}

func TestApelidosJuradosEstaveis(t *testing.T) {
	ap := apelidosJurados([]domain.Avaliacao{{JuradoUsuarioID: 90}, {JuradoUsuarioID: 12}, {JuradoUsuarioID: 90}, {JuradoUsuarioID: 50}})
	if ap[12] != "Jurado 1" || ap[50] != "Jurado 2" || ap[90] != "Jurado 3" || len(ap) != 3 {
		t.Errorf("apelidos: %v", ap)
	}
}
