package service

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizarGrupo(t *testing.T) {
	d, err := NormalizarGrupo(DadosGrupo{Nome: "  Cia   Kpop  ", Tipo: "danca", UF: "al", Instagram: "https://instagram.com/cia.kpop/?hl=pt"})
	if err != nil || d.Nome != "Cia Kpop" || d.UF != "AL" || d.Instagram != "cia.kpop" {
		t.Fatalf("normalização errada: %+v %v", d, err)
	}
	for _, ruim := range []DadosGrupo{
		{Nome: "A", Tipo: "danca"},
		{Nome: "Grupo", Tipo: "circo"},
		{Nome: "Grupo", Tipo: "danca", UF: "ALA"},
		{Nome: "Grupo", Tipo: "danca", Instagram: "nome com espaço"},
		{Nome: "Grupo", Tipo: "danca", LogoURL: "javascript:alert(1)"},
	} {
		if _, err := NormalizarGrupo(ruim); !errors.Is(err, ErrArtistaInvalido) {
			t.Errorf("deveria recusar %+v", ruim)
		}
	}
}

func TestNormalizarIntegrante(t *testing.T) {
	agora := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	ano := 2019
	d, err := NormalizarIntegrante(DadosIntegrante{Email: " Ana@Teste.COM ", NomeArtistico: "Ana", Funcao: "Coreógrafa", DesdeAno: &ano}, agora)
	if err != nil || d.Email != "ana@teste.com" {
		t.Fatalf("%+v %v", d, err)
	}
	futuro := 2027
	if _, err := NormalizarIntegrante(DadosIntegrante{Email: "a@b.com", NomeArtistico: "Ana", DesdeAno: &futuro}, agora); err == nil {
		t.Error("ano no futuro deveria falhar")
	}
	if _, err := NormalizarIntegrante(DadosIntegrante{Email: "sem-arroba", NomeArtistico: "Ana"}, agora); err == nil {
		t.Error("e-mail inválido deveria falhar")
	}
	if AnosDeExperiencia(&ano, agora) != 7 || AnosDeExperiencia(&futuro, agora) != 1 || AnosDeExperiencia(nil, agora) != 0 {
		t.Error("anos de experiência errados")
	}
}

func TestSituacaoIntegrante(t *testing.T) {
	agora := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	nasc := func(anos int) *time.Time { d := agora.AddDate(-anos, 0, -1); return &d }
	casos := []struct {
		nome                                string
		status                              string
		idade                               int
		autorizado, foto, tem               bool
		visivel, anonimo, mostraFoto, insta bool
	}{
		{"convidado não aparece", "convidado", 30, false, false, true, false, false, false, false},
		{"adulto aparece com foto e insta", "ativo", 30, false, false, true, true, false, true, true},
		{"menor sem autorização não aparece", "ativo", 15, false, false, true, false, false, false, false},
		{"menor autorizado sem foto", "ativo", 15, true, false, true, true, false, false, false},
		{"menor autorizado com foto", "ativo", 15, true, true, true, true, false, true, false},
		{"menor de 12 anônimo", "ativo", 10, false, false, true, true, true, false, false},
		{"saiu não aparece", "saiu", 30, false, false, true, false, false, false, false},
	}
	for _, c := range casos {
		s := CalcularSituacaoIntegrante(c.status, nasc(c.idade), c.autorizado, c.foto, c.tem, agora)
		if s.Visivel != c.visivel || s.Anonimo != c.anonimo || s.MostraFoto != c.mostraFoto || s.MostraInstagram != c.insta {
			t.Errorf("%s: %+v", c.nome, s)
		}
	}
}

func TestNormalizarVideo(t *testing.T) {
	casos := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=10": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtube.com/shorts/dQw4w9WgXcQ":           "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.instagram.com/reel/Cabc123/?igsh=x":   "https://www.instagram.com/reel/Cabc123/",
		"https://www.tiktok.com/@cia/video/123":            "https://tiktok.com/@cia/video/123",
	}
	for in, esperado := range casos {
		v, err := NormalizarVideo(in)
		if err != nil || v.URL != esperado {
			t.Errorf("%s: %+v %v", in, v, err)
		}
	}
	if v, _ := NormalizarVideo("https://youtu.be/dQw4w9WgXcQ"); v.Embed != "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ" {
		t.Error("embed do YouTube errado")
	}
	for _, ruim := range []string{"javascript:alert(1)", "https://evil.com/video", "https://www.youtube.com/watch?v=curto", "https://www.instagram.com/perfil"} {
		if _, err := NormalizarVideo(ruim); err == nil {
			t.Errorf("deveria recusar %s", ruim)
		}
	}
}

func TestNormalizarAceite(t *testing.T) {
	agora := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	d, _, err := normalizarAceite(DadosAceite{NomeArtistico: "Lu", DataNascimento: "2012-01-01", Instagram: "lu", FotoURL: "https://x/f.jpg", ResponsavelNome: "Maria Silva", ResponsavelEmail: "m@x.com"}, agora)
	if err != nil || d.Instagram != "" {
		t.Errorf("menor: instagram deveria ser descartado: %+v %v", d, err)
	}
	if _, _, err := normalizarAceite(DadosAceite{NomeArtistico: "Lu", DataNascimento: "2012-01-01"}, agora); err == nil {
		t.Error("menor de 12 a 17 sem responsável deveria falhar")
	}
	d, _, err = normalizarAceite(DadosAceite{NomeArtistico: "Kid", DataNascimento: "2018-01-01", FotoURL: "https://x/f.jpg"}, agora)
	if err != nil || d.FotoURL != "" {
		t.Errorf("menor de 12: sem foto e sem responsável: %+v %v", d, err)
	}
	if CodigoInstagram([]byte{0, 1, 2, 3, 4, 5, 6, 7}) != "EVVE-ABCDEFGH" {
		t.Error("código do Instagram")
	}
}
