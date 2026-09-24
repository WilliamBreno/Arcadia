package handler

import "testing"

func TestAssinaturaDeAudio(t *testing.T) {
	casos := []struct {
		cabeca []byte
		tipo   string
		want   bool
	}{
		{[]byte("ID3\x03\x00\x00\x00\x00\x00\x00\x00\x00"), "audio/mpeg", true},
		{[]byte{0xFF, 0xFB, 0x90, 0x00, 0, 0, 0, 0, 0, 0, 0, 0}, "audio/mpeg", true},
		{[]byte("RIFF\x00\x00\x00\x00WAVE"), "audio/wav", true},
		{[]byte("\x00\x00\x00\x18ftypM4A "), "audio/mp4", true},
		{[]byte("<html><script>"), "audio/mpeg", false},
		{[]byte("RIFF\x00\x00\x00\x00WAVE"), "audio/mpeg", false},
		{[]byte("MZ\x90\x00"), "audio/wav", false},
	}
	for _, c := range casos {
		if got := assinaturaDeAudio(c.cabeca, c.tipo); got != c.want {
			t.Errorf("assinaturaDeAudio(%q, %s) = %v, want %v", c.cabeca, c.tipo, got, c.want)
		}
	}
}
