package service

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrContratacao = errors.New("dados da contratação inválidos")
var ErrContratacaoEstado = errors.New("ação indisponível: o pedido mudou, a proposta está desatualizada ou a data está ocupada")

type ArtistaOferta struct {
	GrupoID            int64             `json:"-" gorm:"primaryKey"`
	Habilitada         bool              `json:"habilitada"`
	PrecoTipo          string            `json:"preco_tipo"`
	ValorCentavos      *int64            `json:"valor_centavos"`
	PassagemInclusa    bool              `json:"passagem_inclusa"`
	HospedagemInclusa  bool              `json:"hospedagem_inclusa"`
	AlimentacaoInclusa bool              `json:"alimentacao_inclusa"`
	Integrantes        int               `json:"integrantes"`
	DuracaoMinutos     int               `json:"duracao_minutos"`
	OrigemID           *int              `json:"origem_id"`
	Observacoes        string            `json:"observacoes"`
	Origem             *MunicipioArtista `json:"origem,omitempty" gorm:"-"`
}

func (ArtistaOferta) TableName() string { return "artista_ofertas" }

func normalizarOferta(o ArtistaOferta) (ArtistaOferta, error) {
	o.Observacoes = strings.TrimSpace(o.Observacoes)
	if (o.PrecoTipo != "fixo" && o.PrecoTipo != "a_partir" && o.PrecoTipo != "consulta") ||
		(o.PrecoTipo != "consulta" && o.ValorCentavos == nil) || (o.ValorCentavos != nil && (*o.ValorCentavos < 0 || *o.ValorCentavos > 1000000000000)) ||
		o.Integrantes < 1 || o.Integrantes > 1000 || o.DuracaoMinutos < 1 || o.DuracaoMinutos > 1440 || utf8.RuneCountInString(o.Observacoes) > 1500 {
		return o, ErrContratacao
	}
	if o.PrecoTipo == "consulta" {
		o.ValorCentavos = nil
	}
	return o, nil
}

func (s *GruposService) Oferta(usuarioID, grupoID int64) (*ArtistaOferta, error) {
	if _, e := s.doGestor(usuarioID, grupoID); e != nil {
		return nil, e
	}
	return s.oferta(grupoID)
}
func (s *GruposService) oferta(grupoID int64) (*ArtistaOferta, error) {
	var o ArtistaOferta
	e := s.db.First(&o, "grupo_id = ?", grupoID).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return &ArtistaOferta{GrupoID: grupoID, PrecoTipo: "consulta", Integrantes: 1, DuracaoMinutos: 30}, nil
	}
	if e != nil {
		return nil, e
	}
	if o.OrigemID != nil {
		var m MunicipioArtista
		if s.db.First(&m, *o.OrigemID).Error == nil {
			o.Origem = &m
		}
	}
	return &o, nil
}
func (s *GruposService) SalvarOferta(usuarioID, grupoID int64, o ArtistaOferta) error {
	o, e := normalizarOferta(o)
	if e != nil {
		return e
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		local := *s
		local.db = tx
		var g Grupo
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, grupoID).Error; e != nil {
			return ErrArtistaNaoEncontrado
		}
		if _, e := local.doGestor(usuarioID, grupoID); e != nil {
			return e
		}
		if o.OrigemID != nil {
			var m MunicipioArtista
			if tx.First(&m, *o.OrigemID).Error != nil {
				return ErrContratacao
			}
		}
		o.GrupoID = grupoID
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "grupo_id"}}, UpdateAll: true}).Create(&o).Error
	})
}

type ArtistaOrcamento struct {
	ID               int64             `json:"id" gorm:"primaryKey"`
	GrupoID          int64             `json:"grupo_id"`
	ClienteID        int64             `json:"-"`
	Inicio           string            `json:"inicio"`
	Fim              string            `json:"fim"`
	DestinoID        int               `json:"destino_id"`
	TipoEvento       string            `json:"tipo_evento"`
	DuracaoMinutos   int               `json:"duracao_minutos"`
	Observacoes      string            `json:"observacoes"`
	Status           string            `json:"status"`
	PropostaCentavos *int64            `json:"proposta_centavos"`
	PropostaTexto    string            `json:"proposta_texto"`
	Versao           int               `json:"versao"`
	Motivo           string            `json:"motivo"`
	CriadoEm         time.Time         `json:"criado_em"`
	AtualizadoEm     time.Time         `json:"atualizado_em"`
	AgendaID         *int64            `json:"-"`
	GrupoNome        string            `json:"grupo_nome" gorm:"-"`
	GrupoSlug        string            `json:"grupo_slug" gorm:"-"`
	ClienteNome      string            `json:"cliente_nome" gorm:"-"`
	Destino          *MunicipioArtista `json:"destino,omitempty" gorm:"-"`
	SouCliente       bool              `json:"sou_cliente" gorm:"-"`
}

func (ArtistaOrcamento) TableName() string { return "artista_orcamentos" }

type ArtistaMensagem struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	OrcamentoID int64     `json:"-"`
	UsuarioID   int64     `json:"-"`
	Texto       string    `json:"texto"`
	CriadoEm    time.Time `json:"criado_em"`
	Autor       string    `json:"autor" gorm:"->"`
	Minha       bool      `json:"minha" gorm:"->"`
}

func (ArtistaMensagem) TableName() string { return "artista_mensagens" }

func normalizarPedido(o ArtistaOrcamento) (ArtistaOrcamento, error) {
	o.TipoEvento = limpar(o.TipoEvento)
	o.Observacoes = strings.TrimSpace(o.Observacoes)
	if !diaDisponibilidadeValido(o.Inicio) || !diaDisponibilidadeValido(o.Fim) || o.Fim < o.Inicio || o.DestinoID <= 0 ||
		utf8.RuneCountInString(o.TipoEvento) < 2 || utf8.RuneCountInString(o.TipoEvento) > 100 || utf8.RuneCountInString(o.Observacoes) > 3000 || o.DuracaoMinutos < 1 || o.DuracaoMinutos > 1440 {
		return o, ErrContratacao
	}
	a, _ := time.Parse("2006-01-02", o.Inicio)
	b, _ := time.Parse("2006-01-02", o.Fim)
	if b.Sub(a) > 366*24*time.Hour {
		return o, ErrContratacao
	}
	return o, nil
}

func (s *GruposService) CriarOrcamento(usuarioID int64, slug string, d ArtistaOrcamento) (*ArtistaOrcamento, error) {
	d, e := normalizarPedido(d)
	if e != nil {
		return nil, e
	}
	// Também aplica a visibilidade e a proteção do perfil solo de menor.
	if _, e := s.Publico(slug); e != nil {
		return nil, e
	}
	var o *ArtistaOrcamento
	e = s.db.Transaction(func(tx *gorm.DB) error {
		var g Grupo
		if tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("slug = ? AND publico", slug).First(&g).Error != nil {
			return ErrArtistaNaoEncontrado
		}
		local := *s
		local.db = tx
		if _, e := local.doGestor(usuarioID, g.ID); e == nil {
			return ErrArtistaSemPermissao
		} else if !errors.Is(e, ErrArtistaSemPermissao) {
			return e
		}
		oferta, e := local.oferta(g.ID)
		if e != nil {
			return e
		}
		if !oferta.Habilitada {
			return ErrContratacaoEstado
		}
		var m MunicipioArtista
		if tx.First(&m, d.DestinoID).Error != nil {
			return ErrContratacao
		}
		now := time.Now()
		o = &ArtistaOrcamento{GrupoID: g.ID, ClienteID: usuarioID, Inicio: d.Inicio, Fim: d.Fim, DestinoID: d.DestinoID, TipoEvento: d.TipoEvento, DuracaoMinutos: d.DuracaoMinutos, Observacoes: d.Observacoes, Status: "novo", CriadoEm: now, AtualizadoEm: now}
		return tx.Create(o).Error
	})
	return o, e
}

func (s *GruposService) enriquecerOrcamento(o *ArtistaOrcamento, usuarioID int64) {
	var g Grupo
	if s.db.First(&g, o.GrupoID).Error == nil {
		o.GrupoNome, o.GrupoSlug = g.Nome, g.Slug
	}
	var u domain.Usuario
	if s.db.First(&u, o.ClienteID).Error == nil {
		o.ClienteNome = primeiroNome(u.Nome)
	}
	var m MunicipioArtista
	if s.db.First(&m, o.DestinoID).Error == nil {
		o.Destino = &m
	}
	o.SouCliente = o.ClienteID == usuarioID
}
func (s *GruposService) Orcamentos(usuarioID, grupoID int64, pagina int) ([]ArtistaOrcamento, error) {
	q := s.db.Model(&ArtistaOrcamento{})
	if grupoID > 0 {
		if _, e := s.doGestor(usuarioID, grupoID); e != nil {
			return nil, e
		}
		q = q.Where("grupo_id = ?", grupoID)
	} else {
		q = q.Where("cliente_id = ?", usuarioID)
	}
	out := []ArtistaOrcamento{}
	e := q.Select("*, to_char(inicio,'YYYY-MM-DD') AS inicio, to_char(fim,'YYYY-MM-DD') AS fim").Order("atualizado_em DESC, id DESC").Offset((max(pagina, 1) - 1) * 200).Limit(200).Find(&out).Error
	for i := range out {
		s.enriquecerOrcamento(&out[i], usuarioID)
	}
	return out, e
}
func (s *GruposService) carregarOrcamento(usuarioID, id int64, lock bool) (*ArtistaOrcamento, error) {
	var o ArtistaOrcamento
	q := s.db.Model(&ArtistaOrcamento{}).Select("*, to_char(inicio,'YYYY-MM-DD') AS inicio, to_char(fim,'YYYY-MM-DD') AS fim")
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if e := q.First(&o, id).Error; e != nil {
		return nil, ErrArtistaNaoEncontrado
	}
	if o.ClienteID != usuarioID {
		if _, e := s.doGestor(usuarioID, o.GrupoID); e != nil {
			return nil, e
		}
	}
	return &o, nil
}
func (s *GruposService) Conversa(usuarioID, id, antes int64) (*ArtistaOrcamento, []ArtistaMensagem, error) {
	o, e := s.carregarOrcamento(usuarioID, id, false)
	if e != nil {
		return nil, nil, e
	}
	s.enriquecerOrcamento(o, usuarioID)
	ms := []ArtistaMensagem{}
	q := s.db.Model(&ArtistaMensagem{}).Select("artista_mensagens.*, split_part(u.nome, ' ', 1) AS autor, (usuario_id = ?) AS minha", usuarioID).
		Joins("JOIN usuarios u ON u.id = artista_mensagens.usuario_id").Where("orcamento_id = ?", id)
	if antes > 0 {
		q = q.Where("artista_mensagens.id < ?", antes)
	}
	e = q.Order("artista_mensagens.id DESC").Limit(200).Scan(&ms).Error
	slices.Reverse(ms)
	return o, ms, e
}

func (s *GruposService) Mensagem(usuarioID, id int64, texto string) error {
	texto = strings.TrimSpace(texto)
	if texto == "" || utf8.RuneCountInString(texto) > 3000 {
		return ErrContratacao
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		local := *s
		local.db = tx
		o, e := local.carregarOrcamento(usuarioID, id, true)
		if e != nil {
			return e
		}
		if o.Status == "cancelado" || o.Status == "recusado" || o.Status == "realizado" {
			return ErrContratacaoEstado
		}
		if e := tx.Create(&ArtistaMensagem{OrcamentoID: id, UsuarioID: usuarioID, Texto: texto, CriadoEm: time.Now()}).Error; e != nil {
			return e
		}
		status := o.Status
		if status == "novo" {
			status = "em_conversa"
		}
		return tx.Model(o).Updates(map[string]any{"status": status, "atualizado_em": time.Now()}).Error
	})
}

type AcaoOrcamento struct {
	Acao          string `json:"acao"`
	ValorCentavos *int64 `json:"valor_centavos"`
	Texto         string `json:"texto"`
	Versao        int    `json:"versao"`
	Motivo        string `json:"motivo"`
}

func (s *GruposService) AcaoOrcamento(usuarioID, id int64, a AcaoOrcamento) error {
	a.Texto = strings.TrimSpace(a.Texto)
	a.Motivo = strings.TrimSpace(a.Motivo)
	if utf8.RuneCountInString(a.Texto) > 3000 || utf8.RuneCountInString(a.Motivo) > 1000 {
		return ErrContratacao
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		local := *s
		local.db = tx
		// Todos os aceites do grupo usam a mesma trava, inclusive de pedidos diferentes.
		var ref ArtistaOrcamento
		if tx.First(&ref, id).Error != nil {
			return ErrArtistaNaoEncontrado
		}
		var g Grupo
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, ref.GrupoID).Error; e != nil {
			return e
		}
		o, e := local.carregarOrcamento(usuarioID, id, true)
		if e != nil {
			return e
		}
		cliente := o.ClienteID == usuarioID
		terminal := o.Status == "cancelado" || o.Status == "recusado" || o.Status == "realizado"
		registrar := func(campos map[string]any, texto string) error {
			if e := tx.Model(o).Updates(campos).Error; e != nil {
				return e
			}
			return tx.Create(&ArtistaMensagem{OrcamentoID: id, UsuarioID: usuarioID, Texto: texto, CriadoEm: time.Now()}).Error
		}
		switch a.Acao {
		case "propor":
			if cliente {
				return ErrArtistaSemPermissao
			}
			if terminal || o.Status == "aceito" {
				return ErrContratacaoEstado
			}
			if a.ValorCentavos == nil || *a.ValorCentavos < 0 || *a.ValorCentavos > 1000000000000 || a.Texto == "" {
				return ErrContratacao
			}
			texto := fmt.Sprintf("Proposta %d — R$ %d,%02d\n%s", o.Versao+1, *a.ValorCentavos/100, *a.ValorCentavos%100, a.Texto)
			return registrar(map[string]any{"status": "proposta_enviada", "proposta_centavos": *a.ValorCentavos, "proposta_texto": a.Texto, "versao": gorm.Expr("versao + 1"), "atualizado_em": time.Now()}, texto)
		case "aceitar":
			if !cliente {
				return ErrArtistaSemPermissao
			}
			if o.Status == "aceito" && a.Versao == o.Versao {
				return nil
			}
			if o.Status != "proposta_enviada" || a.Versao != o.Versao {
				return ErrContratacaoEstado
			}
			var ocupados int64
			if e := tx.Model(&AgendaArtista{}).Where("grupo_id = ? AND inicio <= ?::date AND fim >= ?::date", o.GrupoID, o.Fim, o.Inicio).Count(&ocupados).Error; e != nil {
				return e
			}
			if ocupados > 0 {
				return ErrContratacaoEstado
			}
			if e := tx.Raw(`SELECT count(*) FROM atracoes a JOIN eventos e ON e.id=a.evento_id WHERE a.grupo_id=? AND a.status='confirmado' AND e.status<>'cancelado'
    AND (e.inicio_em AT TIME ZONE e.timezone)::date<=?::date AND (COALESCE(e.fim_em,e.inicio_em+interval '12 hours') AT TIME ZONE e.timezone)::date>=?::date`, o.GrupoID, o.Fim, o.Inicio).Scan(&ocupados).Error; e != nil {
				return e
			}
			if ocupados > 0 {
				return ErrContratacaoEstado
			}
			agenda := AgendaArtista{GrupoID: o.GrupoID, Tipo: "apresentacao", Titulo: "Contratação: " + o.TipoEvento, Inicio: o.Inicio, Fim: o.Fim, Publico: false}
			if e := tx.Create(&agenda).Error; e != nil {
				return e
			}
			if e := tx.Model(&AgendaArtista{}).Where("id = ?", agenda.ID).Update("orcamento_id", o.ID).Error; e != nil {
				return e
			}
			return registrar(map[string]any{"status": "aceito", "agenda_id": agenda.ID, "atualizado_em": time.Now()}, fmt.Sprintf("Proposta %d aceita. Data reservada de %s a %s.", o.Versao, o.Inicio, o.Fim))
		case "cancelar", "recusar":
			destino := "cancelado"
			if a.Acao == "recusar" {
				destino = "recusado"
				if cliente {
					return ErrArtistaSemPermissao
				}
				if o.Status == "aceito" {
					return ErrContratacaoEstado
				}
			}
			if o.Status == destino {
				return nil
			}
			if terminal {
				return ErrContratacaoEstado
			}
			if a.Motivo == "" {
				return ErrContratacao
			}
			if o.AgendaID != nil {
				agendaID := *o.AgendaID
				if e := tx.Model(o).Update("agenda_id", nil).Error; e != nil {
					return e
				}
				if e := tx.Where("id = ? AND orcamento_id = ?", agendaID, id).Delete(&AgendaArtista{}).Error; e != nil {
					return e
				}
			}
			return registrar(map[string]any{"status": destino, "motivo": a.Motivo, "atualizado_em": time.Now()}, "Pedido "+destino+". Motivo: "+a.Motivo)
		case "realizado":
			if cliente {
				return ErrArtistaSemPermissao
			}
			if o.Status == "realizado" {
				return nil
			}
			if o.Status != "aceito" {
				return ErrContratacaoEstado
			}
			// Dia no Brasil, sem marcar uma apresentação futura como concluída.
			loc, _ := time.LoadLocation("America/Fortaleza")
			if o.Fim >= time.Now().In(loc).Format("2006-01-02") {
				return ErrContratacaoEstado
			}
			return registrar(map[string]any{"status": "realizado", "atualizado_em": time.Now()}, "Apresentação marcada como realizada.")
		}
		return fmt.Errorf("%w: ação desconhecida", ErrContratacao)
	})
}
