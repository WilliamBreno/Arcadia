// Comando carga simula a abertura de um lote concorrido (item 5.5): N
// compradores tentam reservar ao mesmo tempo um ingresso com poucas vagas.
// Confere que ninguém vende além do estoque, que não há erro 500 e mede o
// tempo de resposta.
//
// Uso (contra a API LOCAL, nunca produção):
//
//	go run ./cmd/carga -compradores 400 -estoque 50
//
// Lê DATABASE_URL e JWT_SECRET do ambiente (o .env de desenvolvimento): cria
// os usuários de teste direto no banco e assina os tokens de acesso, para
// não depender do limite de tentativas do login.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

func main() {
	api := flag.String("api", "http://localhost:8080/api/v1", "URL da API local")
	compradores := flag.Int("compradores", 400, "quantos compradores simultâneos")
	estoque := flag.Int("estoque", 50, "vagas do lote")
	rampa := flag.Duration("rampa", 500*time.Millisecond, "intervalo em que as chegadas se espalham (0 = todos no mesmo instante; o Windows recusa conexões acima de ~200 simultâneas)")
	flag.Parse()

	if !strings.Contains(*api, "localhost") && !strings.Contains(*api, "127.0.0.1") {
		log.Fatal("o teste de carga só roda contra a API local")
	}
	dbURL, segredo := os.Getenv("DATABASE_URL"), os.Getenv("JWT_SECRET")
	if dbURL == "" || segredo == "" {
		log.Fatal("defina DATABASE_URL e JWT_SECRET (carregue o .env)")
	}
	db, err := repository.Conectar(dbURL, 5)
	if err != nil {
		log.Fatal(err)
	}
	jwt := service.NovoJWTService(segredo, 60)
	run := time.Now().Unix()

	novoUsuario := func(nome string) (*domain.Usuario, string) {
		agora := time.Now()
		u := &domain.Usuario{
			Nome: nome, Email: fmt.Sprintf("%s-%d@carga.local", nome, run),
			PapelPlataforma: domain.PapelUsuario, EmailVerificadoEm: &agora, CriadoEm: agora,
		}
		if err := db.Create(u).Error; err != nil {
			log.Fatalf("criar usuário: %v", err)
		}
		tok, err := jwt.GerarAccessToken(u)
		if err != nil {
			log.Fatal(err)
		}
		return u, tok
	}
	chamar := func(metodo, caminho, token string, corpo any) (int, []byte, time.Duration) {
		var leitor io.Reader
		if corpo != nil {
			b, _ := json.Marshal(corpo)
			leitor = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(metodo, *api+caminho, leitor)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		inicio := time.Now()
		resp, err := http.DefaultClient.Do(req)
		dur := time.Since(inicio)
		if err != nil {
			return 0, []byte(err.Error()), dur
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, b, dur
	}

	// Organizador, evento e lote.
	_, orgTok := novoUsuario("org-carga")
	chamar("POST", "/org/perfil", orgTok, map[string]any{"nome_publico": fmt.Sprintf("Carga %d", run), "tipo_pessoa": "pf", "documento": "12345678901"})
	chamar("PUT", "/org/perfil", orgTok, map[string]any{"nome_publico": fmt.Sprintf("Carga %d", run), "tipo_pessoa": "pf", "documento": "12345678901", "chave_pix": "carga@pix.local", "tipo_chave_pix": "email"})
	inicio := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	_, b, _ := chamar("POST", "/org/eventos", orgTok, map[string]any{"titulo": fmt.Sprintf("Lote concorrido %d", run), "inicio_em": inicio, "tipo_acesso": "ingresso", "categoria": "geek"})
	var ev struct {
		ID   int64  `json:"id"`
		Slug string `json:"slug"`
	}
	_ = json.Unmarshal(b, &ev)
	_, b, _ = chamar("POST", fmt.Sprintf("/org/eventos/%d/ingressos", ev.ID), orgTok, map[string]any{
		"nome": "Lote 1", "preco_centavos": 5000, "quantidade": *estoque, "min_por_pedido": 1, "max_por_pedido": 4, "ativo": true})
	var tipo struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(b, &tipo)
	if s, b, _ := chamar("POST", fmt.Sprintf("/org/eventos/%d/publicar", ev.ID), orgTok, nil); s != 200 {
		log.Fatalf("publicar: %d %s", s, b)
	}

	tokens := make([]string, *compradores)
	for i := range tokens {
		_, tokens[i] = novoUsuario(fmt.Sprintf("comprador%d", i))
	}
	fmt.Printf("Evento %q: %d vagas, %d compradores chegando em %v\n", ev.Slug, *estoque, *compradores, *rampa)

	// Todos disparam no mesmo instante.
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		largada  = make(chan struct{})
		status   = map[int]int{}
		duracoes []time.Duration
		erros500 []string
	)
	for i := range tokens {
		wg.Add(1)
		atraso := time.Duration(0)
		if *rampa > 0 {
			atraso = *rampa * time.Duration(i) / time.Duration(len(tokens))
		}
		go func(tok string) {
			defer wg.Done()
			<-largada
			time.Sleep(atraso)
			s, corpo, d := chamar("POST", "/eventos/"+ev.Slug+"/pedidos", tok, map[string]any{
				"itens": []map[string]any{{"tipo_ingresso_id": tipo.ID, "quantidade": 1}}})
			mu.Lock()
			status[s]++
			duracoes = append(duracoes, d)
			if s >= 500 || s == 0 {
				erros500 = append(erros500, string(corpo))
			}
			mu.Unlock()
		}(tokens[i])
	}
	t0 := time.Now()
	close(largada)
	wg.Wait()
	total := time.Since(t0)

	var reservados int64
	db.Table("itens_pedido").Where("tipo_ingresso_id = ? AND status IN ?", tipo.ID, []string{"reservado", "pago"}).Count(&reservados)

	slices.Sort(duracoes)
	p := func(q float64) time.Duration { return duracoes[int(float64(len(duracoes)-1)*q)] }
	fmt.Printf("Tempo total: %v\n", total.Round(time.Millisecond))
	fmt.Printf("Respostas: %v\n", status)
	fmt.Printf("Latência: p50 %v · p95 %v · p99 %v · máx %v\n", p(0.5).Round(time.Millisecond), p(0.95).Round(time.Millisecond), p(0.99).Round(time.Millisecond), duracoes[len(duracoes)-1].Round(time.Millisecond))
	fmt.Printf("Reservados no banco: %d de %d vagas\n", reservados, *estoque)

	ok := true
	if reservados > int64(*estoque) {
		fmt.Println("FALHA: vendeu além do estoque (overselling)")
		ok = false
	}
	if *compradores >= *estoque && reservados != int64(*estoque) {
		fmt.Println("FALHA: sobrou vaga mesmo com mais compradores que vagas")
		ok = false
	}
	if status[201] != int(reservados) {
		fmt.Printf("FALHA: %d respostas 201 mas %d reservas no banco\n", status[201], reservados)
		ok = false
	}
	if len(erros500) > 0 {
		fmt.Printf("FALHA: %d erros 5xx/conexão. Exemplo: %s\n", len(erros500), erros500[0])
		ok = false
	}
	if ok {
		fmt.Println("OK: sem venda além do estoque, sem erro 500, todas as vagas vendidas")
	} else {
		os.Exit(1)
	}
}
