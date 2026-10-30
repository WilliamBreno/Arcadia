package service

import (
	"errors"
	"testing"
	"time"
)

func TestQRParticipante(t *testing.T) {
	p := PayloadParticipante("segredo", 42)
	if id, err := LerPayloadParticipante("segredo", p); err != nil || id != 42 {
		t.Fatalf("válido: %d %v", id, err)
	}
	ruins := []string{
		"EVVEP:43:" + tokenParticipante("segredo", 42), // trocou a ficha
		PayloadParticipante("outro-segredo", 42),
		"ABC123:tokenqualquer", // QR de ingresso
		"EVVEP:42",
		"",
	}
	for _, r := range ruins {
		if _, err := LerPayloadParticipante("segredo", r); !errors.Is(err, ErrQRParticipante) {
			t.Errorf("deveria recusar %q", r)
		}
	}
}

func TestProximoDaFila(t *testing.T) {
	fila := []ItemFila{
		{FichaID: 1, Status: PalcoApresentou},
		{FichaID: 2, Status: PalcoNoPalco},
		{FichaID: 3, Status: PalcoAusente},
		{FichaID: 4, Status: PalcoAguardando},
		{FichaID: 5, Status: PalcoPresente},
	}
	atual := int64(2)
	if p := ProximoDaFila(fila, &atual); p == nil || p.FichaID != 4 {
		t.Errorf("segue a ordem, pulando quem apresentou, faltou e o atual: %+v", p)
	}
	if p := ProximoDaFila([]ItemFila{{FichaID: 1, Status: PalcoApresentou}}, nil); p != nil {
		t.Error("ninguém mais na fila")
	}
}

func TestFotosNoTelao(t *testing.T) {
	adulto := time.Now().AddDate(-25, 0, 0)
	menor := time.Now().AddDate(-15, 0, 0)
	dados := []byte(`{"personagem": "Nezuko", "foto_referencia": "https://cdn/ref.jpg"}`)
	a := (&linhaFicha{ID: 1, Nome: "Ana Silva", NomeArtistico: "AnaCos", DataNascimento: &adulto, Dados: dados, FotoURL: "https://cdn/ana.jpg"}).item()
	if a.FotoParticipante == "" || a.FotoReferencia == "" || a.Nome != "AnaCos" {
		t.Errorf("adulto: %+v", a)
	}
	m := (&linhaFicha{ID: 2, Nome: "Mini Souza", DataNascimento: &menor, Dados: dados, FotoURL: "https://cdn/mini.jpg"}).item()
	if m.FotoParticipante != "" || m.FotoReferencia == "" || !m.Menor || m.Nome != "Mini" {
		t.Errorf("menor: sem foto própria, com referência e só primeiro nome: %+v", m)
	}
	x := (&linhaFicha{ID: 3, Dados: []byte(`{"foto_referencia": "javascript:alert(1)"}`)}).item()
	if x.FotoReferencia != "" {
		t.Error("referência que não é http é descartada")
	}
}
