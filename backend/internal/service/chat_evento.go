package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrChatAcesso = errors.New("você não tem acesso ao chat da equipe deste evento")
var ErrChatDados = errors.New("dados do chat inválidos")
var ErrChatNaoEncontrado = errors.New("registro do chat não encontrado")
var ErrChatLimite = errors.New("muitas mensagens: aguarde um minuto antes de enviar novamente")

// Jurado confirmado nunca participa do chat, mesmo acumulando outro papel.
const chatParticipantesSQL = `SELECT DISTINCT ON (u.id) e.id AS evento_id, u.id AS usuario_id, u.nome, p.papel,
 (p.papel IN ('organizador','gestor','coprodutor')) AS pode_fixar
 FROM eventos e JOIN (
 SELECT id AS organizador_id, usuario_id, 'organizador' AS papel, 1 AS prioridade, NULL::bigint AS evento_id FROM organizadores
 UNION ALL SELECT organizador_id, usuario_id, papel, 2, NULL::bigint FROM organizador_membros
 UNION ALL SELECT NULL::bigint, o.usuario_id, 'coprodutor', 3, c.evento_id FROM coproducoes c JOIN organizadores o ON o.id=c.organizador_id WHERE c.status='ativa'
 UNION ALL SELECT NULL::bigint, usuario_id, 'staff', 4, evento_id FROM papeis_evento WHERE papel='staff' AND status='confirmado'
 ) p ON (p.evento_id=e.id OR (p.evento_id IS NULL AND p.organizador_id=e.organizador_id))
 JOIN usuarios u ON u.id=p.usuario_id
 WHERE e.id=? AND NOT EXISTS (SELECT 1 FROM papeis_evento j WHERE j.evento_id=e.id AND j.usuario_id=u.id AND j.papel='jurado' AND j.status='confirmado')
 ORDER BY u.id, p.prioridade`

type ChatParticipante struct {
	EventoID  int64  `json:"-"`
	UsuarioID int64  `json:"usuario_id"`
	Nome      string `json:"nome"`
	Papel     string `json:"papel"`
	PodeFixar bool   `json:"pode_fixar"`
	LidoAte   int64  `json:"lido_ate"`
}
type ChatMensagem struct {
	ID        int64     `json:"id"`
	UsuarioID int64     `json:"usuario_id"`
	Nome      string    `json:"nome"`
	Texto     string    `json:"texto"`
	TemFoto   bool      `json:"tem_foto"`
	Fixada    bool      `json:"fixada"`
	CriadoEm  time.Time `json:"criado_em"`
}
type ChatAtencao struct {
	ID         int64      `json:"id"`
	JuradoID   int64      `json:"jurado_id"`
	Nome       string     `json:"nome"`
	MensagemID int64      `json:"mensagem_id"`
	CriadoEm   time.Time  `json:"criado_em"`
	AtendidoEm *time.Time `json:"atendido_em"`
}
type ChatEventoService struct {
	db          *gorm.DB
	frontendURL string
}

func NovoChatEventoService(db *gorm.DB, frontendURL string) *ChatEventoService {
	return &ChatEventoService{db, strings.TrimRight(frontendURL, "/")}
}

func chatParticipantes(db *gorm.DB, eventoID int64) ([]ChatParticipante, error) {
	p := []ChatParticipante{}
	err := db.Raw(chatParticipantesSQL, eventoID).Scan(&p).Error
	return p, err
}
func chatAcesso(db *gorm.DB, usuarioID, eventoID int64) (*ChatParticipante, error) {
	p, err := chatParticipantes(db, eventoID)
	if err != nil {
		return nil, err
	}
	for i := range p {
		if p[i].UsuarioID == usuarioID {
			return &p[i], nil
		}
	}
	return nil, ErrChatAcesso
}
func (s *ChatEventoService) Acesso(usuarioID, eventoID int64) error {
	_, err := chatAcesso(s.db, usuarioID, eventoID)
	return err
}

type ChatCanal struct {
	EventoID int64  `json:"evento_id"`
	Titulo   string `json:"titulo"`
	NaoLidas int64  `json:"nao_lidas"`
}

func (s *ChatEventoService) Canais(usuarioID int64) ([]ChatCanal, error) {
	// Seleciona só eventos nos quais o usuário tem um vínculo potencial.
	var ids []int64
	err := s.db.Raw(`SELECT DISTINCT e.id FROM eventos e WHERE
 e.organizador_id IN (SELECT id FROM organizadores WHERE usuario_id=? UNION SELECT organizador_id FROM organizador_membros WHERE usuario_id=?)
 OR e.id IN (SELECT c.evento_id FROM coproducoes c JOIN organizadores o ON o.id=c.organizador_id WHERE o.usuario_id=? AND c.status='ativa')
 OR e.id IN (SELECT evento_id FROM papeis_evento WHERE usuario_id=? AND papel='staff' AND status='confirmado')`, usuarioID, usuarioID, usuarioID, usuarioID).Scan(&ids).Error
	out := []ChatCanal{}
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if _, err := chatAcesso(s.db, usuarioID, id); errors.Is(err, ErrChatAcesso) {
			continue
		} else if err != nil {
			return nil, err
		}
		var c ChatCanal
		err = s.db.Raw(`SELECT e.id AS evento_id,e.titulo,(SELECT count(*) FROM chat_mensagens m WHERE m.evento_id=e.id AND m.usuario_id<>? AND m.id>COALESCE(l.lido_ate,0)) AS nao_lidas FROM eventos e LEFT JOIN chat_leituras l ON l.evento_id=e.id AND l.usuario_id=? WHERE e.id=?`, usuarioID, usuarioID, id).Scan(&c).Error
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
func chatMensagens(db *gorm.DB, eventoID, antes int64, fixadas bool) ([]ChatMensagem, error) {
	out := []ChatMensagem{}
	q := db.Table("chat_mensagens m").Select("m.id,m.usuario_id,u.nome,m.texto,m.foto IS NOT NULL AS tem_foto,m.fixada,m.criado_em").Joins("JOIN usuarios u ON u.id=m.usuario_id").Where("m.evento_id=?", eventoID)
	if antes > 0 {
		q = q.Where("m.id<?", antes)
	}
	if fixadas {
		q = q.Where("m.fixada=true")
	}
	err := q.Order("m.id DESC").Limit(50).Scan(&out).Error
	return out, err
}

type ChatPagina struct {
	Titulo        string             `json:"titulo"`
	Mensagens     []ChatMensagem     `json:"mensagens"`
	Fixadas       []ChatMensagem     `json:"fixadas"`
	Participantes []ChatParticipante `json:"participantes"`
	Atencoes      []ChatAtencao      `json:"atencoes"`
	LidoAte       int64              `json:"lido_ate"`
	EmailAtivo    bool               `json:"email_ativo"`
	PodeFixar     bool               `json:"pode_fixar"`
}

func (s *ChatEventoService) Pagina(usuarioID, eventoID, antes int64) (*ChatPagina, error) {
	a, err := chatAcesso(s.db, usuarioID, eventoID)
	if err != nil {
		return nil, err
	}
	out := &ChatPagina{EmailAtivo: true, PodeFixar: a.PodeFixar, Atencoes: []ChatAtencao{}}
	if err = s.db.Table("eventos").Where("id=?", eventoID).Pluck("titulo", &out.Titulo).Error; err != nil {
		return nil, err
	}
	if out.Mensagens, err = chatMensagens(s.db, eventoID, antes, false); err != nil {
		return nil, err
	}
	if out.Fixadas, err = chatMensagens(s.db, eventoID, 0, true); err != nil {
		return nil, err
	}
	if out.Participantes, err = chatParticipantes(s.db, eventoID); err != nil {
		return nil, err
	}
	var leituras []struct {
		UsuarioID, LidoAte int64
		EmailAtivo         bool
	}
	if err = s.db.Table("chat_leituras").Where("evento_id=?", eventoID).Find(&leituras).Error; err != nil {
		return nil, err
	}
	for _, l := range leituras {
		if l.UsuarioID == usuarioID {
			out.LidoAte = l.LidoAte
			out.EmailAtivo = l.EmailAtivo
		}
		for i := range out.Participantes {
			if out.Participantes[i].UsuarioID == l.UsuarioID {
				out.Participantes[i].LidoAte = l.LidoAte
			}
		}
	}
	err = s.db.Table("chat_atencoes a").Select("a.*,u.nome").Joins("JOIN usuarios u ON u.id=a.jurado_id").Where("a.evento_id=? AND a.atendido_em IS NULL", eventoID).Order("a.id").Scan(&out.Atencoes).Error
	return out, err
}
func validarMensagemChat(texto, chave string, foto []byte) error {
	if utf8.RuneCountInString(texto) > 3000 || (texto == "" && len(foto) == 0) || len(chave) < 8 || len(chave) > 80 {
		return ErrChatDados
	}
	return nil
}
func (s *ChatEventoService) Enviar(usuarioID, eventoID int64, texto, chave string, foto []byte) (int64, error) {
	texto = strings.TrimSpace(texto)
	if err := validarMensagemChat(texto, chave, foto); err != nil {
		return 0, err
	}
	var id int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Serializa criação, fixação e pedidos de atenção por evento.
		var e domain.Evento
		if err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).First(&e, eventoID).Error; err != nil {
			return ErrChatAcesso
		}
		if _, err := chatAcesso(tx, usuarioID, eventoID); err != nil {
			return err
		}
		if err := tx.Table("chat_mensagens").Where("evento_id=? AND usuario_id=? AND chave=?", eventoID, usuarioID, chave).Pluck("id", &id).Error; err != nil {
			return err
		}
		if id > 0 {
			return nil
		}
		var recentes int64
		if err := tx.Table("chat_mensagens").Where("evento_id=? AND usuario_id=? AND criado_em>?", eventoID, usuarioID, time.Now().Add(-time.Minute)).Count(&recentes).Error; err != nil {
			return err
		}
		if recentes >= 30 {
			return ErrChatLimite
		}
		if err := tx.Raw(`INSERT INTO chat_mensagens(evento_id,usuario_id,texto,chave,foto) VALUES (?,?,?,?,?) ON CONFLICT(evento_id,usuario_id,chave) DO UPDATE SET chave=excluded.chave RETURNING id`, eventoID, usuarioID, texto, chave, foto).Scan(&id).Error; err != nil {
			return err
		}
		return prepararLeiturasChat(tx, eventoID)
	})
	return id, err
}
func prepararLeiturasChat(tx *gorm.DB, eventoID int64) error {
	p, err := chatParticipantes(tx, eventoID)
	if err != nil {
		return err
	}
	for _, u := range p {
		if err = tx.Exec(`INSERT INTO chat_leituras(evento_id,usuario_id) VALUES (?,?) ON CONFLICT DO NOTHING`, eventoID, u.UsuarioID).Error; err != nil {
			return err
		}
	}
	return nil
}
func (s *ChatEventoService) Ler(usuarioID, eventoID, ate int64, email *bool) error {
	if _, err := chatAcesso(s.db, usuarioID, eventoID); err != nil {
		return err
	}
	if ate < 0 {
		return ErrChatDados
	}
	var maxID int64
	if err := s.db.Raw(`SELECT COALESCE(MAX(id),0) FROM chat_mensagens WHERE evento_id=?`, eventoID).Scan(&maxID).Error; err != nil {
		return err
	}
	if ate > maxID {
		return ErrChatDados
	}
	if err := s.db.Exec(`INSERT INTO chat_leituras(evento_id,usuario_id,lido_ate) VALUES (?,?,?) ON CONFLICT(evento_id,usuario_id) DO UPDATE SET lido_ate=GREATEST(chat_leituras.lido_ate,excluded.lido_ate)`, eventoID, usuarioID, ate).Error; err != nil {
		return err
	}
	if email != nil {
		return s.db.Exec(`UPDATE chat_leituras SET email_ativo=? WHERE evento_id=? AND usuario_id=?`, *email, eventoID, usuarioID).Error
	}
	return nil
}
func (s *ChatEventoService) Fixar(usuarioID, eventoID, mensagemID int64, fixar bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var e domain.Evento
		if err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).First(&e, eventoID).Error; err != nil {
			return ErrChatAcesso
		}
		a, err := chatAcesso(tx, usuarioID, eventoID)
		if err != nil {
			return err
		}
		if !a.PodeFixar {
			return ErrChatAcesso
		}
		if fixar {
			var n int64
			if err := tx.Table("chat_mensagens").Where("evento_id=? AND fixada AND id<>?", eventoID, mensagemID).Count(&n).Error; err != nil {
				return err
			}
			if n >= 5 {
				return fmt.Errorf("%w: no máximo cinco avisos fixados", ErrChatDados)
			}
		}
		res := tx.Table("chat_mensagens").Where("id=? AND evento_id=?", mensagemID, eventoID).Update("fixada", fixar)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrChatNaoEncontrado
		}
		return nil
	})
}
func (s *ChatEventoService) Foto(usuarioID, eventoID, mensagemID int64) ([]byte, error) {
	if _, err := chatAcesso(s.db, usuarioID, eventoID); err != nil {
		return nil, err
	}
	var r struct{ Foto []byte }
	res := s.db.Table("chat_mensagens").Select("foto").Where("evento_id=? AND id=? AND foto IS NOT NULL", eventoID, mensagemID).Take(&r)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, ErrChatNaoEncontrado
	}
	return r.Foto, res.Error
}
