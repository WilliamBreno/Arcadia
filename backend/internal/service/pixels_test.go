package service

import "testing"

func TestNormalizarPixels(t *testing.T) {
	p, err := NormalizarPixels(Pixels{Meta: " 1234567890123456 ", GA4: "g-abc123xyz9", TikTok: "c4abcdefghij12345678"})
	if err != nil {
		t.Fatalf("IDs válidos: %v", err)
	}
	if p.Meta != "1234567890123456" || p.GA4 != "G-ABC123XYZ9" || p.TikTok != "C4ABCDEFGHIJ12345678" {
		t.Errorf("normalização: %+v", p)
	}
	if _, err := NormalizarPixels(Pixels{}); err != nil {
		t.Errorf("tudo vazio é válido (desligado): %v", err)
	}
	ruins := []Pixels{
		{Meta: "123"},
		{Meta: "12345678901'><script>"},
		{GA4: "UA-12345-1"},
		{GA4: "G-ABC\";alert(1)//"},
		{TikTok: "abc"},
		{TikTok: "C4ABCDEFGHIJ1234<script>"},
	}
	for _, r := range ruins {
		if _, err := NormalizarPixels(r); err == nil {
			t.Errorf("deveria recusar %+v", r)
		}
	}
}
