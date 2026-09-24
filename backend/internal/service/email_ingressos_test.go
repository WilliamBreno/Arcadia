package service

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

func TestCorpoEmailIngressos(t *testing.T) {
	inicio := time.Date(2026, 12, 5, 20, 0, 0, 0, time.UTC)
	evento := &domain.Evento{Titulo: `Festa <script>alert(1)</script>`, InicioEm: &inicio}
	itens := []domain.ItemPedido{
		{TitularNome: `Ana <b>`, Codigo: "AAAA1111", QRToken: "tok1"},
		{TitularNome: "Beto", Codigo: "BBBB2222", QRToken: "tok2"},
	}

	corpo, imagens := corpoEmailIngressos("Olá!", "Pago.", evento, itens, "seg", "https://evve.app")
	if len(imagens) != 2 {
		t.Fatalf("esperava 1 imagem por ingresso, veio %d", len(imagens))
	}
	for _, img := range imagens {
		if !bytes.HasPrefix(img.PNG, []byte("\x89PNG\r\n\x1a\n")) {
			t.Errorf("QR %s não é PNG válido", img.ContentID)
		}
		if !strings.Contains(corpo, `src="cid:`+img.ContentID+`"`) {
			t.Errorf("corpo não referencia cid:%s", img.ContentID)
		}
	}
	if strings.Contains(corpo, "<script>") || strings.Contains(corpo, "Ana <b>") {
		t.Error("texto de usuário/organizador deve ser escapado")
	}
	if !strings.Contains(corpo, "https://evve.app/ingresso/AAAA1111/tok1") {
		t.Error("deve trazer o link do ingresso online")
	}

	evento.QRRotativo = true
	corpo, imagens = corpoEmailIngressos("Olá!", "Pago.", evento, itens, "seg", "https://evve.app")
	if len(imagens) != 0 || strings.Contains(corpo, "cid:") {
		t.Error("evento com QR rotativo não pode levar QR fixo no e-mail")
	}
	if !strings.Contains(corpo, "muda a cada 30 segundos") {
		t.Error("deve explicar o QR dinâmico")
	}
}
