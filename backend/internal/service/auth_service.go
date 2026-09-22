package service

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

var (
	ErrCredenciaisInvalidas = errors.New("email ou senha inválidos")
	ErrEmailJaCadastrado    = errors.New("e-mail já cadastrado")
	ErrTokenInvalido        = errors.New("token inválido ou expirado")
)

type AuthService struct {
	usuarios      *repository.UsuarioRepository
	refreshTokens *repository.RefreshTokenRepository
	jwt           *JWTService
	refreshTTL    time.Duration
}

func NovoAuthService(usuarios *repository.UsuarioRepository, refreshTokens *repository.RefreshTokenRepository, jwt *JWTService, refreshTTLDias int) *AuthService {
	return &AuthService{
		usuarios:      usuarios,
		refreshTokens: refreshTokens,
		jwt:           jwt,
		refreshTTL:    time.Duration(refreshTTLDias) * 24 * time.Hour,
	}
}

func (s *AuthService) RefreshTTL() time.Duration {
	return s.refreshTTL
}

func (s *AuthService) Cadastrar(nome, email, senha string) (*domain.Usuario, error) {
	if _, err := s.usuarios.BuscarPorEmail(email); err == nil {
		return nil, ErrEmailJaCadastrado
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	senhaHash := string(hash)

	usuario := &domain.Usuario{
		Nome:            nome,
		Email:           email,
		SenhaHash:       &senhaHash,
		PapelPlataforma: domain.PapelUsuario,
		CriadoEm:        time.Now(),
	}
	if err := s.usuarios.Criar(usuario); err != nil {
		return nil, err
	}
	return usuario, nil
}

func (s *AuthService) Login(email, senha string) (*domain.Usuario, error) {
	usuario, err := s.usuarios.BuscarPorEmail(email)
	if err != nil || usuario.SenhaHash == nil {
		return nil, ErrCredenciaisInvalidas
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*usuario.SenhaHash), []byte(senha)); err != nil {
		return nil, ErrCredenciaisInvalidas
	}
	return usuario, nil
}

// LoginOuCriarComGoogle vincula pelo google_id; se não existir, tenta casar
// por e-mail (conta criada antes por senha) e vincula; senão cria uma
// conta nova já com e-mail verificado (a Google já validou o e-mail).
func (s *AuthService) LoginOuCriarComGoogle(googleID, email, nome, avatarURL string) (*domain.Usuario, error) {
	if usuario, err := s.usuarios.BuscarPorGoogleID(googleID); err == nil {
		return usuario, nil
	}

	if usuario, err := s.usuarios.BuscarPorEmail(email); err == nil {
		usuario.GoogleID = &googleID
		if err := s.usuarios.Salvar(usuario); err != nil {
			return nil, err
		}
		return usuario, nil
	}

	agora := time.Now()
	usuario := &domain.Usuario{
		Nome:              nome,
		Email:             email,
		GoogleID:          &googleID,
		AvatarURL:         avatarURL,
		EmailVerificadoEm: &agora,
		PapelPlataforma:   domain.PapelUsuario,
		CriadoEm:          agora,
	}
	if err := s.usuarios.Criar(usuario); err != nil {
		return nil, err
	}
	return usuario, nil
}

// EmitirTokens gera o access token (JWT) e um refresh token opaco novo,
// persistindo apenas o hash do refresh token.
func (s *AuthService) EmitirTokens(usuario *domain.Usuario) (accessToken, refreshTokenPlano string, err error) {
	accessToken, err = s.jwt.GerarAccessToken(usuario)
	if err != nil {
		return "", "", err
	}

	refreshTokenPlano, err = GerarTokenOpaco()
	if err != nil {
		return "", "", err
	}

	registro := &domain.RefreshToken{
		UsuarioID: usuario.ID,
		TokenHash: HashToken(refreshTokenPlano),
		ExpiraEm:  time.Now().Add(s.refreshTTL),
		CriadoEm:  time.Now(),
	}
	if err := s.refreshTokens.Criar(registro); err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenPlano, nil
}

// RenovarTokens valida o refresh token, revoga-o (rotação) e emite um par novo.
func (s *AuthService) RenovarTokens(refreshTokenPlano string) (usuario *domain.Usuario, novoAccessToken, novoRefreshTokenPlano string, err error) {
	registro, err := s.refreshTokens.BuscarValidoPorHash(HashToken(refreshTokenPlano))
	if err != nil {
		return nil, "", "", ErrTokenInvalido
	}

	usuario, err = s.usuarios.BuscarPorID(registro.UsuarioID)
	if err != nil {
		return nil, "", "", ErrTokenInvalido
	}

	if err := s.refreshTokens.Revogar(registro); err != nil {
		return nil, "", "", err
	}

	novoAccessToken, novoRefreshTokenPlano, err = s.EmitirTokens(usuario)
	if err != nil {
		return nil, "", "", err
	}

	return usuario, novoAccessToken, novoRefreshTokenPlano, nil
}

func (s *AuthService) Logout(refreshTokenPlano string) error {
	registro, err := s.refreshTokens.BuscarValidoPorHash(HashToken(refreshTokenPlano))
	if err != nil {
		return nil
	}
	return s.refreshTokens.Revogar(registro)
}

func (s *AuthService) SolicitarVerificacaoEmail(usuario *domain.Usuario) (string, error) {
	token, err := GerarTokenOpaco()
	if err != nil {
		return "", err
	}
	hash := HashToken(token)
	expira := time.Now().Add(24 * time.Hour)
	usuario.EmailVerificacaoTokenHash = &hash
	usuario.EmailVerificacaoExpiraEm = &expira
	if err := s.usuarios.Salvar(usuario); err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) VerificarEmail(token string) error {
	usuario, err := s.usuarios.BuscarPorTokenVerificacao(HashToken(token))
	if err != nil {
		return ErrTokenInvalido
	}
	agora := time.Now()
	usuario.EmailVerificadoEm = &agora
	usuario.EmailVerificacaoTokenHash = nil
	usuario.EmailVerificacaoExpiraEm = nil
	return s.usuarios.Salvar(usuario)
}

// SolicitarRedefinicaoSenha não falha se o e-mail não existir (evita
// enumeração de contas) — o handler decide a resposta genérica.
func (s *AuthService) SolicitarRedefinicaoSenha(email string) (usuario *domain.Usuario, token string, err error) {
	usuario, err = s.usuarios.BuscarPorEmail(email)
	if err != nil {
		return nil, "", nil
	}

	token, err = GerarTokenOpaco()
	if err != nil {
		return nil, "", err
	}
	hash := HashToken(token)
	expira := time.Now().Add(1 * time.Hour)
	usuario.SenhaResetTokenHash = &hash
	usuario.SenhaResetExpiraEm = &expira
	if err := s.usuarios.Salvar(usuario); err != nil {
		return nil, "", err
	}
	return usuario, token, nil
}

func (s *AuthService) RedefinirSenha(token, novaSenha string) error {
	usuario, err := s.usuarios.BuscarPorTokenReset(HashToken(token))
	if err != nil {
		return ErrTokenInvalido
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(novaSenha), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	senhaHash := string(hash)
	usuario.SenhaHash = &senhaHash
	usuario.SenhaResetTokenHash = nil
	usuario.SenhaResetExpiraEm = nil
	if err := s.usuarios.Salvar(usuario); err != nil {
		return err
	}

	return s.refreshTokens.RevogarTodosDoUsuario(usuario.ID)
}
