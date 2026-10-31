package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestAgendaArtistaValidacao(t *testing.T) {
	base := AgendaArtista{Tipo: "apresentacao", Titulo: " Show ", Inicio: "2028-02-29", Fim: "2028-02-29"}
	if d, e := NormalizarAgenda(base); e != nil || d.Titulo != "Show" || d.Publico {
		t.Fatalf("agenda privada: %+v %v", d, e)
	}
	for _, caso := range []struct {
		nome  string
		mudar func(*AgendaArtista)
	}{
		{"data inexistente", func(d *AgendaArtista) { d.Inicio = "2027-02-29" }},
		{"período invertido", func(d *AgendaArtista) { d.Fim = "2028-02-28" }},
		{"período excessivo", func(d *AgendaArtista) { d.Fim = "2030-01-01" }},
		{"tipo inválido", func(d *AgendaArtista) { d.Tipo = "outro" }},
		{"apresentação sem título", func(d *AgendaArtista) { d.Titulo = "  " }},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			d := base
			caso.mudar(&d)
			if _, e := NormalizarAgenda(d); e == nil {
				t.Fatal("deveria recusar")
			}
		})
	}
	base.Tipo, base.Titulo = "bloqueio", ""
	if _, e := NormalizarAgenda(base); e != nil {
		t.Fatal(e)
	}
}

func TestImagemKitRestringeArmazenamento(t *testing.T) {
	base := "https://imagens.evve.test/publico"
	if got := nomeImagemKit(base, base+"/abc-123.jpg"); got != "abc-123.jpg" {
		t.Fatal(got)
	}
	for _, u := range []string{"http://127.0.0.1/admin.jpg", "https://imagens.evve.test.evil/publico/x.jpg", base + "/../privado/x.jpg", base + "/sub/x.jpg", base + "/x.jpg?token=privado", base + "/%2e%2e.jpg", base + "/x.svg", "https://usuario:senha@imagens.evve.test/publico/x.jpg"} {
		if nomeImagemKit(base, u) != "" {
			t.Fatalf("aceitou %s", u)
		}
	}
}

func TestMidiaKitPDFComFoto(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.White)
	var b bytes.Buffer
	if e := png.Encode(&b, img); e != nil {
		t.Fatal(e)
	}
	foto := normalizarFotoKit(b.Bytes())
	if len(foto) == 0 {
		t.Fatal("foto válida recusada")
	}
	if normalizarFotoKit([]byte("arquivo inválido")) != nil {
		t.Fatal("arquivo inválido aceito")
	}
	p := &GrupoPublico{Nome: "Dança e ação", TipoNome: "Dança", LogoURL: "foto-publica", Integrantes: []IntegrantePublico{{NomeArtistico: "Artista", FotoURL: "foto-autorizada"}}}
	chamadas := []string{}
	pdf, e := gerarMidiaKit(p, "https://evve.test", "Evve", func(u string) []byte { chamadas = append(chamadas, u); return foto })
	if e != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) || !bytes.Contains(pdf, []byte("/Subtype /Image")) {
		t.Fatalf("PDF com foto inválido: %v", e)
	}
	if len(chamadas) != 2 || chamadas[1] != "foto-autorizada" {
		t.Fatal(chamadas)
	}
}
