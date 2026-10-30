package service

// Item 9.1: repasse automático por Pix (Mercado Pago Payouts). Decisões do
// dono (30/10/2026):
//   - organizador verificado recebe 1 dia útil após o fim do evento; os
//     demais, 3 (REPASSE_DIAS_UTEIS_VERIFICADO / REPASSE_DIAS_UTEIS);
//   - o sistema envia sozinho até REPASSE_AUTO_LIMITE_CENTAVOS (R$ 20 mil),
//     sem reembolso falhado, sem contestação em disputa, com organizador
//     ativo e chave Pix CPF/CNPJ igual ao documento dele; senão, espera o
//     admin aprovar (um clique, que envia pela mesma API);
//   - tarifa do envio, se houver, é custo da Evve (não sai do organizador).
//
// Segurança do dinheiro: a chave de idempotência é a referência do envio
// (EVVE-REP-<repasse>-<tentativa>), gravada ANTES de chamar o MP. Se a
// chamada cair no meio, a próxima rodada repete com a mesma chave e o MP
// devolve o mesmo envio (nunca dois Pix). Só uma recusa definitiva (4xx) ou
// uma transação com erro abre uma nova tentativa, e só pelo admin.

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/mail"
	"github.com/WilliamBreno/Arcadia/backend/internal/mercadopago"
)

const (
	EnvioManual              = "manual"
	EnvioNaFila              = "na_fila"
	EnvioAguardandoAprovacao = "aguardando_aprovacao"
	EnvioEnviando            = "enviando"
	EnvioProcessando         = "processando"
	EnvioEnviado             = "enviado"
	EnvioFalhou              = "falhou"

	ChaveRepasseDiasUteisVerificado = "REPASSE_DIAS_UTEIS_VERIFICADO"
	ChaveRepasseAutoLimiteCentavos  = "REPASSE_AUTO_LIMITE_CENTAVOS"
)

var (
	ErrEnvioDesligado   = errors.New("o envio automático por Pix não está ligado (falta configurar o Mercado Pago)")
	ErrEnvioEmAndamento = errors.New("este repasse está sendo enviado pelo Mercado Pago agora")
	ErrChaveNaoEnviavel = errors.New("a chave Pix do organizador é aleatória: envie o Pix manualmente e marque como pago")
	ErrRepasseSemChave  = errors.New("o organizador não tem chave Pix cadastrada")
)

// Payouter é o que o envio precisa do Mercado Pago (trocado em teste).
type Payouter interface {
	EnviarPayout(req mercadopago.PayoutRequest, chave string, assinatura ed25519.PrivateKey) (*mercadopago.Payout, error)
	TransacoesPayout(payoutID string) ([]mercadopago.TransacaoPayout, error)
	ModoTeste() bool
}

// DefinirEnvio liga o repasse automático. Sem chave privada fora do modo
// de teste, o envio fica desligado e o repasse segue manual.
func (s *RepasseService) DefinirEnvio(p Payouter, chave ed25519.PrivateKey, m *mail.Cliente, nomePlataforma string) {
	s.payout, s.chaveAssinatura, s.mailCliente, s.nomePlataforma = p, chave, m, nomePlataforma
}

// EnvioLigado: há cliente e (em produção) a chave de assinatura.
func (s *RepasseService) EnvioLigado() bool {
	return s.payout != nil && (s.chaveAssinatura != nil || s.payout.ModoTeste())
}

// tipoPixMP converte o tipo de chave do Evve para o do MP ("" = não enviável).
func tipoPixMP(tipo string) string {
	switch tipo {
	case PixCPF:
		return "CPF"
	case PixCNPJ:
		return "CNPJ"
	case PixEmail:
		return "EMAIL"
	case PixTelefone:
		return "PHONE"
	}
	return ""
}

// MotivoParaAprovar devolve por que o envio automático precisa do admin
// ("" = pode ir sozinho). Testada.
func MotivoParaAprovar(valor, limite int64, org *domain.Organizador, reembolsosFalhos, contestacoes int64) string {
	switch {
	case valor <= 0:
		return "não há valor a enviar agora"
	case org.Status != domain.StatusOrganizadorAtivo:
		return "organizador não está ativo"
	case org.ChavePix == "":
		return "organizador sem chave Pix"
	case (org.TipoChavePix != PixCPF && org.TipoChavePix != PixCNPJ) || somenteDigitos(org.ChavePix) != somenteDigitos(org.Documento):
		return "a chave Pix não é o CPF/CNPJ do próprio organizador"
	case limite > 0 && valor > limite:
		return fmt.Sprintf("valor acima do limite automático (%s)", formatarReais(limite))
	case reembolsosFalhos > 0:
		return "há reembolso com falha neste evento"
	case contestacoes > 0:
		return "há contestação de cartão em disputa neste evento"
	}
	return ""
}

func (s *RepasseService) pendencias(eventoID int64) (reembolsos, contestacoes int64) {
	s.db.Table("reembolsos r").Joins("JOIN itens_pedido i ON i.id = r.item_id").Joins("JOIN pedidos p ON p.id = i.pedido_id").
		Where("p.evento_id = ? AND r.status = ?", eventoID, domain.StatusReembolsoFalhou).Count(&reembolsos)
	s.db.Model(&domain.Contestacao{}).Where("evento_id = ? AND status = ?", eventoID, domain.ContestacaoEmDisputa).Count(&contestacoes)
	return
}

// diasParaRepasse: verificado recebe antes.
func (s *RepasseService) diasParaRepasse(organizadorID int64) int {
	dias, _ := s.config.BuscarInt64(domain.ChaveRepasseDiasUteis)
	if s.organizadores != nil {
		if org, err := s.organizadores.BuscarPorID(organizadorID); err == nil && org.VerificadoEm != nil {
			if d, err := s.config.BuscarInt64(ChaveRepasseDiasUteisVerificado); err == nil && d >= 0 {
				return int(d)
			}
		}
	}
	return int(dias)
}

// ResultadoEnvio resume uma rodada do job.
type ResultadoEnvio struct {
	Enviados, AguardandoAprovacao, Confirmados, Falhas int
}

// EnviarRepasses é o job: manda o que está na fila, repete o que ficou
// "enviando" (mesma chave) e confere os que estão processando.
func (s *RepasseService) EnviarRepasses() (ResultadoEnvio, error) {
	var r ResultadoEnvio
	if !s.EnvioLigado() {
		return r, nil
	}
	agora := time.Now()
	var fila []domain.Repasse
	if err := s.db.Where("status = ? AND liberar_em <= ? AND envio_status IN ?", domain.StatusRepassePendente, agora,
		[]string{EnvioNaFila, EnvioEnviando}).Order("id").Find(&fila).Error; err != nil {
		return r, err
	}
	limite, _ := s.config.BuscarInt64(ChaveRepasseAutoLimiteCentavos)
	for i := range fila {
		rep := &fila[i]
		if rep.EnvioStatus == EnvioNaFila {
			org, err := s.organizadores.BuscarPorID(rep.OrganizadorID)
			if err != nil {
				continue
			}
			ref, con := s.pendencias(rep.EventoID)
			if motivo := MotivoParaAprovar(ValorAPagarAgora(rep), limite, org, ref, con); motivo != "" {
				s.db.Model(rep).Where("envio_status = ?", EnvioNaFila).
					Updates(map[string]any{"envio_status": EnvioAguardandoAprovacao, "envio_motivo": motivo})
				r.AguardandoAprovacao++
				continue
			}
		}
		switch err := s.enviar(rep.ID, nil); {
		case err == nil:
			r.Enviados++
		default:
			slog.Warn("repasse automático", "repasse", rep.ID, "erro", err)
			r.Falhas++
		}
	}

	var processando []domain.Repasse
	s.db.Where("status = ? AND envio_status = ?", domain.StatusRepassePendente, EnvioProcessando).Find(&processando)
	for i := range processando {
		ok, err := s.conferir(&processando[i])
		if err != nil {
			slog.Warn("conferir repasse", "repasse", processando[i].ID, "erro", err)
		}
		if ok {
			r.Confirmados++
		}
	}
	return r, nil
}

// AprovarEnvio é o clique do admin: envia mesmo acima do limite ou com
// pendência (decisão dele). Depois de uma falha, abre nova tentativa.
func (s *RepasseService) AprovarEnvio(repasseID, adminID int64) (*domain.Repasse, error) {
	if !s.EnvioLigado() {
		return nil, ErrEnvioDesligado
	}
	if err := s.enviar(repasseID, &adminID); err != nil {
		return nil, err
	}
	return s.repasses.BuscarPorID(repasseID)
}

// enviar trava o repasse, grava a referência e chama o MP.
func (s *RepasseService) enviar(repasseID int64, adminID *int64) error {
	var rep domain.Repasse
	var org *domain.Organizador
	var referencia string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&rep, repasseID).Error; err != nil {
			return ErrRepasseNaoEncontrado
		}
		if rep.Status != domain.StatusRepassePendente {
			return ErrRepasseNaoPendente
		}
		if rep.EnvioStatus == EnvioProcessando || rep.EnvioStatus == EnvioEnviado {
			return ErrEnvioEmAndamento
		}
		o, err := s.organizadores.BuscarPorID(rep.OrganizadorID)
		if err != nil {
			return err
		}
		org = o
		if org.ChavePix == "" {
			return ErrRepasseSemChave
		}
		if tipoPixMP(org.TipoChavePix) == "" {
			return ErrChaveNaoEnviavel
		}
		if ValorAPagarAgora(&rep) <= 0 {
			return fmt.Errorf("%w: não há valor a enviar", ErrRepasseNaoPendente)
		}
		mud := map[string]any{"envio_status": EnvioEnviando, "envio_motivo": ""}
		if rep.EnvioStatus == EnvioEnviando && rep.EnvioReferencia != nil {
			referencia = *rep.EnvioReferencia // repetir a mesma (idempotente)
		} else {
			rep.EnvioTentativa++
			referencia = fmt.Sprintf("EVVE-REP-%d-%d", rep.ID, rep.EnvioTentativa)
			mud["envio_tentativa"] = rep.EnvioTentativa
			mud["envio_referencia"] = referencia
		}
		if adminID != nil {
			mud["envio_aprovado_por"] = *adminID
		}
		return tx.Model(&rep).Updates(mud).Error
	})
	if err != nil {
		return err
	}

	req := mercadopago.NovoPayoutPix(referencia, fmt.Sprintf("Repasse Evve evento %d", rep.EventoID),
		tipoPixMP(org.TipoChavePix), org.ChavePix, ValorAPagarAgora(&rep))
	var assinatura ed25519.PrivateKey
	if !s.payout.ModoTeste() {
		assinatura = s.chaveAssinatura
	}
	p, err := s.payout.EnviarPayout(req, referencia, assinatura)
	if err != nil {
		if errors.Is(err, mercadopago.ErrPayoutRecusado) {
			s.db.Model(&domain.Repasse{}).Where("id = ?", rep.ID).
				Updates(map[string]any{"envio_status": EnvioFalhou, "envio_motivo": resumirErro(err)})
			s.avisarFalha(&rep, org, resumirErro(err))
		}
		// Rede ou 5xx: fica "enviando"; a próxima rodada repete com a mesma chave.
		return err
	}
	return s.db.Model(&domain.Repasse{}).Where("id = ?", rep.ID).
		Updates(map[string]any{"envio_status": EnvioProcessando, "mp_payout_id": p.ID.String()}).Error
}

func resumirErro(err error) string {
	m := err.Error()
	if len(m) > 300 {
		m = m[:300]
	}
	return m
}

// conferir: Pix creditado = repasse pago (com o lançamento de sempre);
// erro ou cancelado = falhou (nenhum dinheiro saiu; o admin decide).
func (s *RepasseService) conferir(rep *domain.Repasse) (bool, error) {
	if rep.MPPayoutID == nil {
		return false, nil
	}
	ts, err := s.payout.TransacoesPayout(*rep.MPPayoutID)
	if err != nil || len(ts) == 0 {
		return false, err
	}
	t := ts[0]
	switch t.Status {
	case "success":
		pago, err := s.marcarPago(rep, "Mercado Pago payout "+*rep.MPPayoutID, "Pix automático ("+deref8(rep.EnvioReferencia)+")")
		if err != nil {
			return false, err
		}
		s.db.Model(&domain.Repasse{}).Where("id = ?", rep.ID).Update("envio_status", EnvioEnviado)
		s.avisarEnviado(pago)
		return true, nil
	case "error", "canceled":
		motivo := "o Mercado Pago não concluiu o Pix (" + t.Status
		if t.StatusDetail != "" {
			motivo += ": " + t.StatusDetail
		}
		motivo += ")"
		s.db.Model(&domain.Repasse{}).Where("id = ?", rep.ID).Updates(map[string]any{"envio_status": EnvioFalhou, "envio_motivo": motivo})
		if org, err := s.organizadores.BuscarPorID(rep.OrganizadorID); err == nil {
			s.avisarFalha(rep, org, motivo)
		}
	}
	return false, nil
}

func deref8(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *RepasseService) emailDoOrganizador(organizadorID int64) (string, string) {
	var linha struct{ Email, NomePublico string }
	s.db.Table("organizadores o").Select("u.email, o.nome_publico").Joins("JOIN usuarios u ON u.id = o.usuario_id").
		Where("o.id = ?", organizadorID).Scan(&linha)
	return linha.Email, linha.NomePublico
}

func (s *RepasseService) tituloEvento(eventoID int64) string {
	if e, err := s.eventos.BuscarPorID(eventoID); err == nil {
		return e.Titulo
	}
	return ""
}

func (s *RepasseService) avisarEnviado(rep *domain.Repasse) {
	if s.mailCliente == nil || rep == nil {
		return
	}
	email, nome := s.emailDoOrganizador(rep.OrganizadorID)
	if email == "" {
		return
	}
	corpo := fmt.Sprintf(`<p>Olá, %s!</p><p>O repasse de <strong>%s</strong> foi enviado por Pix: <strong>%s</strong>. Ele aparece na sua conta em instantes.</p>`,
		html.EscapeString(nome), html.EscapeString(s.tituloEvento(rep.EventoID)), formatarReais(ValorAPagarAgora(rep)))
	if rep.ReservaCentavos > 0 {
		corpo += fmt.Sprintf(`<p>A reserva de segurança (%s) é liberada depois, como combinado.</p>`, formatarReais(rep.ReservaCentavos))
	}
	_ = s.mailCliente.Enviar(email, "Repasse enviado — "+s.nomePlataforma, corpo)
}

func (s *RepasseService) avisarFalha(rep *domain.Repasse, org *domain.Organizador, motivo string) {
	if s.mailCliente == nil {
		return
	}
	email, nome := s.emailDoOrganizador(org.ID)
	if email == "" {
		return
	}
	corpo := fmt.Sprintf(`<p>Olá, %s!</p><p>Não conseguimos enviar o Pix do repasse de <strong>%s</strong>. Confira se a chave Pix cadastrada no seu perfil de organizador está certa. Nossa equipe vai tentar de novo e falar com você se precisar.</p>`,
		html.EscapeString(nome), html.EscapeString(s.tituloEvento(rep.EventoID)))
	slog.Warn("repasse não enviado", "repasse", rep.ID, "motivo", motivo)
	_ = s.mailCliente.Enviar(email, "Repasse: confira sua chave Pix — "+s.nomePlataforma, corpo)
}

// EnvioParaPainel é o texto do status para o admin e o organizador.
func RotuloEnvio(status string) string {
	return map[string]string{
		EnvioManual: "manual", EnvioNaFila: "na fila do Pix automático", EnvioAguardandoAprovacao: "aguardando aprovação do admin",
		EnvioEnviando: "enviando", EnvioProcessando: "Pix em processamento", EnvioEnviado: "Pix enviado", EnvioFalhou: "envio falhou",
	}[strings.TrimSpace(status)]
}
