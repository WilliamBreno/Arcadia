package service

import (
	"context"
	"errors"

	"google.golang.org/api/idtoken"
)

var ErrGoogleIDTokenInvalido = errors.New("id_token do Google inválido")

// GoogleAuthService valida o ID token que o frontend recebe do Google
// Identity Services e devolve os dados básicos do perfil. Exige
// GOOGLE_CLIENT_ID configurado (seção 4 do plano) — sem isso, o login
// com Google fica indisponível, mas e-mail/senha funciona normalmente.
type GoogleAuthService struct {
	clientID string
}

func NovoGoogleAuthService(clientID string) *GoogleAuthService {
	return &GoogleAuthService{clientID: clientID}
}

func (s *GoogleAuthService) Habilitado() bool {
	return s.clientID != ""
}

type PerfilGoogle struct {
	GoogleID  string
	Email     string
	Nome      string
	AvatarURL string
}

func (s *GoogleAuthService) ValidarIDToken(ctx context.Context, idTokenStr string) (*PerfilGoogle, error) {
	if !s.Habilitado() {
		return nil, ErrGoogleIDTokenInvalido
	}

	payload, err := idtoken.Validate(ctx, idTokenStr, s.clientID)
	if err != nil {
		return nil, ErrGoogleIDTokenInvalido
	}

	email, _ := payload.Claims["email"].(string)
	nome, _ := payload.Claims["name"].(string)
	avatarURL, _ := payload.Claims["picture"].(string)
	if email == "" {
		return nil, ErrGoogleIDTokenInvalido
	}

	return &PerfilGoogle{
		GoogleID:  payload.Subject,
		Email:     email,
		Nome:      nome,
		AvatarURL: avatarURL,
	}, nil
}
