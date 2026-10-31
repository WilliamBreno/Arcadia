package service

import "testing"

func TestOfertaArtista(t *testing.T) {
	valor := int64(150099)
	o := ArtistaOferta{PrecoTipo: "fixo", ValorCentavos: &valor, Integrantes: 3, DuracaoMinutos: 45}
	if _, e := normalizarOferta(o); e != nil {
		t.Fatal(e)
	}
	for _, alterar := range []func(*ArtistaOferta){func(o *ArtistaOferta) { o.ValorCentavos = nil }, func(o *ArtistaOferta) { o.Integrantes = 0 }, func(o *ArtistaOferta) { o.PrecoTipo = "desconto" }, func(o *ArtistaOferta) { v := int64(-1); o.ValorCentavos = &v }} {
		ruim := o
		alterar(&ruim)
		if _, e := normalizarOferta(ruim); e == nil {
			t.Fatal("oferta inválida aceita")
		}
	}
	o.PrecoTipo = "consulta"
	normal, e := normalizarOferta(o)
	if e != nil || normal.ValorCentavos != nil {
		t.Fatal("sob consulta não deve publicar valor antigo")
	}
}
func TestPedidoArtista(t *testing.T) {
	p := ArtistaOrcamento{Inicio: "2026-11-01", Fim: "2026-11-03", DestinoID: 2704302, TipoEvento: "Festival", DuracaoMinutos: 30}
	if _, e := normalizarPedido(p); e != nil {
		t.Fatal(e)
	}
	p.Fim = "2026-10-31"
	if _, e := normalizarPedido(p); e == nil {
		t.Fatal("período invertido")
	}
	p.Fim = "2026-11-03"
	p.Inicio = "2026-02-30"
	if _, e := normalizarPedido(p); e == nil {
		t.Fatal("dia inexistente")
	}
}
func TestCalcularViagemArtista(t *testing.T) {
	onibus, aviao := int64(50), int64(30)
	minimo, limiar := 50, 500
	c := ArtistaViagemConfig{Habilitada: true, OnibusCentavosKm: &onibus, AviaoCentavosKm: &aviao, DistanciaMinKm: &minimo, AviaoDesdeKm: &limiar, IdaVolta: true}
	for _, caso := range []struct {
		km      int64
		pessoas int
		modo    string
		valor   int64
	}{{49, 3, "sem_estimativa", 0}, {50, 3, "onibus", 15000}, {499, 3, "onibus", 149700}, {500, 3, "aviao", 90000}, {1000, 1, "aviao", 60000}} {
		m, v, e := calcularViagemArtista(c, caso.km, caso.pessoas)
		if e != nil || m != caso.modo || v != caso.valor {
			t.Fatalf("%+v => %s %d %v", caso, m, v, e)
		}
	}
	c.IdaVolta = false
	m, v, e := calcularViagemArtista(c, 500, 3)
	if e != nil || m != "aviao" || v != 45000 {
		t.Fatalf("só ida: %s %d %v", m, v, e)
	}
	c.Habilitada = false
	if _, _, e := calcularViagemArtista(c, 500, 3); e == nil {
		t.Fatal("config desabilitada")
	}
	c.Habilitada = true
	c.AviaoCentavosKm = nil
	if validarViagemConfig(c) == nil {
		t.Fatal("config incompleta")
	}
	c.Habilitada = false
	if validarViagemConfig(c) != nil {
		t.Fatal("deve poder ficar desativada sem parâmetros")
	}
}
