package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestReduzirParaJPEG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3000, 1000))
	for x := 0; x < 3000; x++ {
		c := color.RGBA{0, 0, 0, 255}
		if x >= 1500 {
			c = color.RGBA{255, 255, 255, 255}
		}
		for y := 0; y < 1000; y++ {
			src.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	out, err := ReduzirParaJPEG(buf.Bytes(), 2048)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 2048 || b.Dy() != 682 {
		t.Errorf("tamanho %dx%d, esperado 2048x682", b.Dx(), b.Dy())
	}
	if r, _, _, _ := img.At(100, 300).RGBA(); r > 0x2000 {
		t.Errorf("lado esquerdo devia ser escuro")
	}
	if r, _, _, _ := img.At(1900, 300).RGBA(); r < 0xd000 {
		t.Errorf("lado direito devia ser claro")
	}
	// JPEG regravado não leva segmento EXIF (APP1).
	if bytes.Contains(out, []byte("Exif")) {
		t.Error("saída com EXIF")
	}
	// Pequena não muda de tamanho.
	peq := image.NewRGBA(image.Rect(0, 0, 400, 300))
	buf.Reset()
	_ = png.Encode(&buf, peq)
	out, _ = ReduzirParaJPEG(buf.Bytes(), 2048)
	img, _ = jpeg.Decode(bytes.NewReader(out))
	if img.Bounds().Dx() != 400 {
		t.Error("foto pequena não devia mudar")
	}
	if _, err := ReduzirParaJPEG([]byte("não é imagem"), 2048); err == nil {
		t.Error("aceitou arquivo inválido")
	}
}
