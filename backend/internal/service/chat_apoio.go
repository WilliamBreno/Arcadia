package service

import (
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChatApoio struct {
	Responsaveis []ChatParticipante `json:"responsaveis"`
	Pedido       *ChatAtencao       `json:"pedido"`
}

func chatEventoJurado(db *gorm.DB, usuarioID int64, slug string) (int64, error) {
	var id int64
	err := db.Raw(`SELECT e.id FROM eventos e JOIN papeis_evento p ON p.evento_id=e.id WHERE e.slug=? AND p.usuario_id=? AND p.papel='jurado' AND p.status='confirmado'`, slug, usuarioID).Scan(&id).Error
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, ErrChatAcesso
	}
	return id, nil
}
func (s *ChatEventoService) Apoio(usuarioID int64, slug string) (*ChatApoio, error) {
	id, err := chatEventoJurado(s.db, usuarioID, slug)
	if err != nil {
		return nil, err
	}
	p, err := chatParticipantes(s.db, id)
	if err != nil {
		return nil, err
	}
	out := &ChatApoio{Responsaveis: p}
	var pedidos []ChatAtencao
	err = s.db.Table("chat_atencoes").Where("evento_id=? AND jurado_id=?", id, usuarioID).Order("id DESC").Limit(1).Find(&pedidos).Error
	if len(pedidos) > 0 {
		out.Pedido = &pedidos[0]
	}
	return out, err
}
func (s *ChatEventoService) PedirAtencao(usuarioID int64, slug, texto string) error {
	texto = strings.TrimSpace(texto)
	if utf8.RuneCountInString(texto) > 500 {
		return ErrChatDados
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		id, err := chatEventoJurado(tx, usuarioID, slug)
		if err != nil {
			return err
		}
		var evento domain.Evento
		if err = tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).First(&evento, id).Error; err != nil {
			return err
		}
		if _, err = chatEventoJurado(tx, usuarioID, slug); err != nil {
			return err
		}
		var pendentes int64
		if err = tx.Table("chat_atencoes").Where("evento_id=? AND jurado_id=? AND atendido_em IS NULL", id, usuarioID).Count(&pendentes).Error; err != nil {
			return err
		}
		if pendentes > 0 {
			return nil
		}
		var recentes int64
		if err = tx.Table("chat_atencoes").Where("evento_id=? AND jurado_id=? AND criado_em>?", id, usuarioID, time.Now().Add(-2*time.Minute)).Count(&recentes).Error; err != nil {
			return err
		}
		if recentes > 0 {
			return fmt.Errorf("%w: aguarde dois minutos antes de pedir atenção novamente", ErrChatDados)
		}
		var mensagemID int64
		err = tx.Raw(`INSERT INTO chat_mensagens(evento_id,usuario_id,chave,texto) VALUES (?,?,?,?) RETURNING id`, id, usuarioID, fmt.Sprintf("atencao-%d", time.Now().UnixNano()), "Pedido de atenção do jurado. "+texto).Scan(&mensagemID).Error
		if err != nil {
			return err
		}
		if err = tx.Exec(`INSERT INTO chat_atencoes(evento_id,jurado_id,mensagem_id) VALUES (?,?,?)`, id, usuarioID, mensagemID).Error; err != nil {
			return err
		}
		return prepararLeiturasChat(tx, id)
	})
}
func (s *ChatEventoService) Atender(usuarioID, eventoID, atencaoID int64) error {
	if _, err := chatAcesso(s.db, usuarioID, eventoID); err != nil {
		return err
	}
	r := s.db.Table("chat_atencoes").Where("id=? AND evento_id=? AND atendido_em IS NULL", atencaoID, eventoID).Updates(map[string]any{"atendido_por": usuarioID, "atendido_em": time.Now()})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return ErrChatNaoEncontrado
	}
	return nil
}

// Apenas contagem e link: o conteúdo privado e as fotos não saem por e-mail.
// Fila e cursor são gravados juntos, para retries do job não duplicarem resumos.
func (s *ChatEventoService) Resumos(agora time.Time) (int, error) {
	total := 0
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var lock bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(731114)").Scan(&lock).Error; err != nil {
			return err
		}
		if !lock {
			return nil
		}
		var candidatos []struct {
			EventoID, UsuarioID, LidoAte, NotificadoAte int64
			Email, Titulo                               string
		}
		err := tx.Raw(`SELECT l.evento_id,l.usuario_id,l.lido_ate,l.notificado_ate,u.email,e.titulo FROM chat_leituras l JOIN usuarios u ON u.id=l.usuario_id JOIN eventos e ON e.id=l.evento_id WHERE l.email_ativo AND u.email_verificado_em IS NOT NULL AND (l.notificado_em IS NULL OR l.notificado_em<?) AND EXISTS(SELECT 1 FROM chat_mensagens m WHERE m.evento_id=l.evento_id AND m.usuario_id<>l.usuario_id AND m.id>GREATEST(l.lido_ate,l.notificado_ate) AND m.criado_em<?) ORDER BY l.evento_id,l.usuario_id LIMIT 200 FOR UPDATE OF l SKIP LOCKED`, agora.Add(-time.Hour), agora.Add(-5*time.Minute)).Scan(&candidatos).Error
		if err != nil {
			return err
		}
		for _, c := range candidatos {
			if _, err = chatAcesso(tx, c.UsuarioID, c.EventoID); err == ErrChatAcesso {
				continue
			} else if err != nil {
				return err
			}
			var lote struct{ N, Ultimo int64 }
			if err = tx.Raw(`SELECT COUNT(*) AS n,COALESCE(MAX(id),0) AS ultimo FROM chat_mensagens WHERE evento_id=? AND usuario_id<>? AND id>GREATEST(?::bigint,?::bigint) AND criado_em<?`, c.EventoID, c.UsuarioID, c.LidoAte, c.NotificadoAte, agora.Add(-5*time.Minute)).Scan(&lote).Error; err != nil {
				return err
			}
			if lote.N == 0 {
				continue
			}
			corpo := fmt.Sprintf(`<p>Há %d mensagem(ns) não lida(s) no chat da equipe de <strong>%s</strong>.</p><p><a href="%s/chat-evento/%d">Abrir conversa</a></p><p>Você pode desativar estes resumos no chat.</p>`, lote.N, html.EscapeString(c.Titulo), html.EscapeString(s.frontendURL), c.EventoID)
			if err = tx.Exec(`INSERT INTO emails_fila(destinatario,assunto,corpo_html) VALUES (?,?,?)`, c.Email, "Mensagens da equipe — "+c.Titulo, corpo).Error; err != nil {
				return err
			}
			if err = tx.Exec(`UPDATE chat_leituras SET notificado_ate=?,notificado_em=? WHERE evento_id=? AND usuario_id=?`, lote.Ultimo, agora, c.EventoID, c.UsuarioID).Error; err != nil {
				return err
			}
			total++
		}
		return nil
	})
	return total, err
}
