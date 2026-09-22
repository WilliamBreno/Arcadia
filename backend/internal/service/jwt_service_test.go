package service_test

import (
	"testing"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/service"
)

func TestJWTServiceGerarEValidar(t *testing.T) {
	jwtService := service.NovoJWTService("segredo-de-teste", 15)
	usuario := &domain.Usuario{ID: 42, PapelPlataforma: domain.PapelAdminPlataforma}

	token, err := jwtService.GerarAccessToken(usuario)
	if err != nil {
		t.Fatalf("erro ao gerar token: %v", err)
	}

	claims, err := jwtService.ValidarAccessToken(token)
	if err != nil {
		t.Fatalf("erro ao validar token: %v", err)
	}
	if claims.UsuarioID != 42 {
		t.Errorf("UsuarioID = %d, esperado 42", claims.UsuarioID)
	}
	if claims.PapelPlataforma != domain.PapelAdminPlataforma {
		t.Errorf("PapelPlataforma = %s, esperado %s", claims.PapelPlataforma, domain.PapelAdminPlataforma)
	}
}

func TestJWTServiceRejeitaTokenExpirado(t *testing.T) {
	// TTL negativo força um token já expirado no momento da geração.
	jwtService := service.NovoJWTService("segredo-de-teste", -1)
	usuario := &domain.Usuario{ID: 1, PapelPlataforma: domain.PapelUsuario}

	token, err := jwtService.GerarAccessToken(usuario)
	if err != nil {
		t.Fatalf("erro ao gerar token: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if _, err := jwtService.ValidarAccessToken(token); err == nil {
		t.Fatal("esperava erro para token expirado, mas validou com sucesso")
	}
}

func TestJWTServiceRejeitaSegredoErrado(t *testing.T) {
	emissor := service.NovoJWTService("segredo-a", 15)
	validador := service.NovoJWTService("segredo-b", 15)
	usuario := &domain.Usuario{ID: 1, PapelPlataforma: domain.PapelUsuario}

	token, err := emissor.GerarAccessToken(usuario)
	if err != nil {
		t.Fatalf("erro ao gerar token: %v", err)
	}

	if _, err := validador.ValidarAccessToken(token); err == nil {
		t.Fatal("esperava erro para token assinado com outro segredo")
	}
}
