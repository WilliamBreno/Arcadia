package service

import "testing"

func TestCorpoCampanha(t *testing.T) {
	got := CorpoCampanha("Oi <b>gente</b>!\nLinha 2\n\nVejam https://evve.app/e/x?a=1&b=2 agora\n\n\n javascript:alert(1) http://inseguro.com")
	quer := `<p>Oi &lt;b&gt;gente&lt;/b&gt;!<br>Linha 2</p><p>Vejam <a href="https://evve.app/e/x?a=1&amp;b=2">https://evve.app/e/x?a=1&amp;b=2</a> agora</p><p>javascript:alert(1) http://inseguro.com</p>`
	if got != quer {
		t.Errorf("\n got %s\nquer %s", got, quer)
	}
}
