package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
)

// Entrar com código no e-mail (item 7.2): a pessoa digita o e-mail, recebe
// 6 dígitos e entra sem senha, direto no checkout. Se a conta não existe,
// depois do código certo pedimos o nome e criamos a conta com o e-mail já
// verificado (acertar o código prova que o e-mail é dela).

const (
	ValidadeCodigoLogin  = 10 * time.Minute
	TentativasCodigo     = 5
	MaxNomeCodigoLogin   = 120
	digitosCodigoLogin   = 6
	limiteCodigoLoginMax = 1_000_000
)

var (
	ErrCodigoInvalido = errors.New("código inválido ou expirado")
	ErrPrecisaNome    = errors.New("informe seu nome para criar a conta")
)

type CodigoLogin struct {
	ID         int64 `gorm:"primaryKey"`
	Email      string
	CodigoHash string
	Tentativas int
	ExpiraEm   time.Time
	UsadoEm    *time.Time
	CriadoEm   time.Time
}

func (CodigoLogin) TableName() string { return "codigos_login" }

type CodigoLoginService struct {
	db      *gorm.DB
	mail    *mail.Cliente
	segredo string
	nome    string
}

func NovoCodigoLoginService(db *gorm.DB, m *mail.Cliente, segredo, nomePlataforma string) *CodigoLoginService {
	return &CodigoLoginService{db: db, mail: m, segredo: segredo, nome: nomePlataforma}
}

// NormalizarEmail: minúsculas e sem espaços. Testada.
func NormalizarEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

// hashCodigo: HMAC do e-mail com o código. Sem o segredo do servidor, quem
// lesse o banco não teria como testar os 1 milhão de códigos. Testada.
func hashCodigo(segredo, email, codigo string) string {
	m := hmac.New(sha256.New, []byte(segredo))
	fmt.Fprintf(m, "codigo-login:%s:%s", email, codigo)
	return hex.EncodeToString(m.Sum(nil))
}

func gerarCodigoLogin() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(limiteCodigoLoginMax))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digitosCodigoLogin, n.Int64()), nil
}

// Solicitar gera um código novo (os anteriores deixam de valer) e manda por
// e-mail. Responde igual exista a conta ou não, para não revelar quem tem
// conta no Evve.
func (s *CodigoLoginService) Solicitar(email string) error {
	email = NormalizarEmail(email)
	codigo, err := gerarCodigoLogin()
	if err != nil {
		return err
	}
	agora := time.Now()
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&CodigoLogin{}).Where("email = ? AND usado_em IS NULL", email).Update("usado_em", agora).Error; err != nil {
			return err
		}
		return tx.Create(&CodigoLogin{
			Email: email, CodigoHash: hashCodigo(s.segredo, email, codigo),
			ExpiraEm: agora.Add(ValidadeCodigoLogin), CriadoEm: agora,
		}).Error
	})
	if err != nil {
		return err
	}
	corpo := fmt.Sprintf(`<div style="font-family:Arial,sans-serif;max-width:480px;margin:auto;color:#111">`+
		`<p>Seu código para entrar no %s:</p>`+
		`<p style="font-size:32px;font-weight:bold;letter-spacing:8px;margin:16px 0">%s</p>`+
		`<p>Vale por 10 minutos. Se não foi você, ignore este e-mail: ninguém entra sem o código.</p>`+
		`<p style="color:#777;font-size:12px">Nunca pedimos este código por telefone ou mensagem.</p></div>`, s.nome, codigo)
	return s.mail.Enviar(email, codigo+" é o seu código de acesso — "+s.nome, corpo)
}

// Verificar confere o código. Conta nova sem nome: ErrPrecisaNome, e o
// código continua valendo para a pessoa mandar de novo com o nome.
func (s *CodigoLoginService) Verificar(email, codigo, nome string) (*domain.Usuario, error) {
	email = NormalizarEmail(email)
	codigo = strings.TrimSpace(codigo)
	nome = strings.Join(strings.Fields(nome), " ")
	if len([]rune(nome)) > MaxNomeCodigoLogin {
		nome = string([]rune(nome)[:MaxNomeCodigoLogin])
	}
	var usuario *domain.Usuario
	var errFinal error
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var c CodigoLogin
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("email = ? AND usado_em IS NULL AND expira_em > ?", email, time.Now()).
			Order("criado_em DESC").First(&c).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errFinal = ErrCodigoInvalido
			return nil
		}
		if err != nil {
			return err
		}
		if c.Tentativas >= TentativasCodigo {
			errFinal = ErrCodigoInvalido
			return nil
		}
		if !hmac.Equal([]byte(c.CodigoHash), []byte(hashCodigo(s.segredo, email, codigo))) {
			errFinal = ErrCodigoInvalido
			return tx.Model(&c).Update("tentativas", gorm.Expr("tentativas + 1")).Error
		}

		var u domain.Usuario
		err = tx.Where("lower(email) = ?", email).Order("id").First(&u).Error
		agora := time.Now()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if nome == "" {
				errFinal = ErrPrecisaNome
				return nil
			}
			u = domain.Usuario{Nome: nome, Email: email, EmailVerificadoEm: &agora, PapelPlataforma: domain.PapelUsuario, CriadoEm: agora}
			if err := tx.Create(&u).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		case u.EmailVerificadoEm == nil:
			u.EmailVerificadoEm = &agora
			if err := tx.Model(&u).Update("email_verificado_em", agora).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&c).Update("usado_em", agora).Error; err != nil {
			return err
		}
		usuario = &u
		return nil
	})
	if err != nil {
		return nil, err
	}
	if errFinal != nil {
		return nil, errFinal
	}
	return usuario, nil
}
