package service

import (
	"errors"
	"testing"
	"time"
)

var hojeTalento = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func nasc(anos int, diasAMais int) time.Time {
	return hojeTalento.AddDate(-anos, 0, diasAMais)
}

func TestIdade(t *testing.T) {
	if Idade(nasc(18, 0), hojeTalento) != 18 || Idade(nasc(18, 1), hojeTalento) != 17 || Idade(nasc(12, 0), hojeTalento) != 12 {
		t.Error("idade em anos completos")
	}
}

func TestSituacaoPorIdade(t *testing.T) {
	agora := time.Now()
	autorizado := &agora
	casos := []struct {
		nome       string
		p          PerfilTalento
		n          time.Time
		visivel    bool
		menor      bool
		indexar    bool
		mostraFoto bool
	}{
		{"adulto", PerfilTalento{Publico: true, FotoURL: "https://x/f.jpg"}, nasc(25, 0), true, false, true, true},
		{"adulto oculto", PerfilTalento{Publico: false}, nasc(25, 0), false, false, false, false},
		{"15 sem autorização", PerfilTalento{Publico: true, FotoURL: "https://x/f.jpg"}, nasc(15, 0), false, true, false, false},
		{"15 autorizado sem foto", PerfilTalento{Publico: true, FotoURL: "https://x/f.jpg", AutorizadoEm: autorizado}, nasc(15, 0), true, true, false, false},
		{"15 autorizado com foto", PerfilTalento{Publico: true, FotoURL: "https://x/f.jpg", AutorizadoEm: autorizado, AutorizaFoto: true}, nasc(15, 0), true, true, false, true},
		{"11 mesmo autorizado", PerfilTalento{Publico: true, AutorizadoEm: autorizado}, nasc(11, 0), false, true, false, false},
		{"faz 18 hoje", PerfilTalento{Publico: true}, nasc(18, 0), true, false, true, false},
	}
	for _, c := range casos {
		s := Situacao(&c.p, c.n, hojeTalento)
		if s.Visivel != c.visivel || s.Menor != c.menor || s.Indexar != c.indexar || s.MostraFoto != c.mostraFoto {
			t.Errorf("%s: %+v", c.nome, s)
		}
	}
}

func TestNormalizarPerfil(t *testing.T) {
	d, _, err := NormalizarPerfil(DadosPerfil{NomeArtistico: " Kira  Cos ", DataNascimento: nasc(20, 0).Format("2006-01-02"),
		Instagram: "https://www.instagram.com/kira.cos/?hl=pt", TikTok: "@kira_cos", UF: "al", ResponsavelEmail: "x@y.z"}, hojeTalento)
	if err != nil || d.NomeArtistico != "Kira Cos" || d.Instagram != "kira.cos" || d.TikTok != "kira_cos" || d.UF != "AL" || d.ResponsavelEmail != "" {
		t.Errorf("adulto: %+v %v", d, err)
	}
	d, _, err = NormalizarPerfil(DadosPerfil{NomeArtistico: "Mini Cos", DataNascimento: nasc(14, 0).Format("2006-01-02"),
		Instagram: "mini.cos", Cidade: "Maceió", ResponsavelNome: "Maria Souza", ResponsavelEmail: "Maria@Exemplo.com"}, hojeTalento)
	if err != nil || d.Instagram != "" || d.Cidade != "" || d.ResponsavelEmail != "maria@exemplo.com" {
		t.Errorf("menor: redes e cidade descartadas, e-mail do responsável normalizado: %+v %v", d, err)
	}
	if _, _, err := NormalizarPerfil(DadosPerfil{NomeArtistico: "Mini Cos", DataNascimento: nasc(14, 0).Format("2006-01-02")}, hojeTalento); !errors.Is(err, ErrPerfilInvalido) {
		t.Error("menor sem responsável deve falhar")
	}
	if _, _, err := NormalizarPerfil(DadosPerfil{NomeArtistico: "Bebê", DataNascimento: nasc(10, 0).Format("2006-01-02")}, hojeTalento); !errors.Is(err, ErrMenorDe12) {
		t.Error("menor de 12 não cria perfil")
	}
	if _, _, err := NormalizarPerfil(DadosPerfil{NomeArtistico: "Kira", DataNascimento: nasc(20, 0).Format("2006-01-02"), Instagram: "kira<script>"}, hojeTalento); !errors.Is(err, ErrPerfilInvalido) {
		t.Error("@ com caracteres estranhos deve falhar")
	}
}

func TestMontarEscondeDadosDeMenor(t *testing.T) {
	p := &PerfilTalento{Slug: "mini", NomeArtistico: "Mini", FotoURL: "https://x/f.jpg", Cidade: "Maceió", UF: "AL", Instagram: "mini.cos"}
	pub := Montar(p, SituacaoPerfil{Visivel: true, Menor: true}, nil)
	if pub.Cidade != "" || pub.Instagram != "" || pub.FotoURL != "" || pub.Indexar || pub.Participacoes == nil {
		t.Errorf("menor: %+v", pub)
	}
	pub = Montar(p, SituacaoPerfil{Visivel: true, Indexar: true, MostraFoto: true}, nil)
	if pub.Cidade != "Maceió" || pub.Instagram != "mini.cos" || pub.FotoURL == "" || !pub.Indexar {
		t.Errorf("adulto: %+v", pub)
	}
}
