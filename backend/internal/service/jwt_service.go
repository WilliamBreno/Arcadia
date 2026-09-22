package service

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
)

// ClaimsAcesso são as claims do access token (JWT curto, seção 4 do plano).
type ClaimsAcesso struct {
	UsuarioID       int64                  `json:"usuario_id"`
	PapelPlataforma domain.PapelPlataforma `json:"papel_plataforma"`
	jwt.RegisteredClaims
}

type JWTService struct {
	segredo []byte
	ttl     time.Duration
}

func NovoJWTService(segredo string, ttlMinutos int) *JWTService {
	return &JWTService{segredo: []byte(segredo), ttl: time.Duration(ttlMinutos) * time.Minute}
}

func (s *JWTService) GerarAccessToken(u *domain.Usuario) (string, error) {
	agora := time.Now()
	claims := ClaimsAcesso{
		UsuarioID:       u.ID,
		PapelPlataforma: u.PapelPlataforma,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(agora),
			ExpiresAt: jwt.NewNumericDate(agora.Add(s.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.segredo)
}

func (s *JWTService) ValidarAccessToken(tokenStr string) (*ClaimsAcesso, error) {
	claims := &ClaimsAcesso{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return s.segredo, nil
	})
	if err != nil || !token.Valid {
		if err != nil {
			return nil, err
		}
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
