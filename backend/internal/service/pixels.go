package service

import (
	"errors"
	"regexp"
	"strings"
)

// Pixels de anúncio por evento (item 7.5). O organizador informa só os
// identificadores; o site carrega os scripts nas páginas daquele evento e
// apenas depois que o visitante aceita no aviso de cookies. Nenhum dado
// pessoal (nome, e-mail, CPF) é mandado às plataformas.

type Pixels struct {
	Meta   string `json:"meta"`
	GA4    string `json:"ga4"`
	TikTok string `json:"tiktok"`
}

var (
	ErrPixelMeta   = errors.New("ID do pixel da Meta: só números (de 10 a 20 dígitos), como aparece no Gerenciador de Eventos")
	ErrPixelGA4    = errors.New("ID do Google Analytics 4: começa com G- (ex.: G-ABC123XYZ9)")
	ErrPixelTikTok = errors.New("ID do pixel do TikTok: letras maiúsculas e números (de 15 a 30), como aparece no TikTok Ads")

	reMeta   = regexp.MustCompile(`^[0-9]{10,20}$`)
	reGA4    = regexp.MustCompile(`^G-[A-Z0-9]{4,15}$`)
	reTikTok = regexp.MustCompile(`^[A-Z0-9]{15,30}$`)
)

// NormalizarPixels limpa espaços e caixa e valida o formato de cada ID
// (vazio = desligado). O formato fechado impede injetar script pelo campo.
// Testada.
func NormalizarPixels(p Pixels) (Pixels, error) {
	p.Meta = strings.TrimSpace(p.Meta)
	p.GA4 = strings.ToUpper(strings.TrimSpace(p.GA4))
	p.TikTok = strings.ToUpper(strings.TrimSpace(p.TikTok))
	if p.Meta != "" && !reMeta.MatchString(p.Meta) {
		return p, ErrPixelMeta
	}
	if p.GA4 != "" && !reGA4.MatchString(p.GA4) {
		return p, ErrPixelGA4
	}
	if p.TikTok != "" && !reTikTok.MatchString(p.TikTok) {
		return p, ErrPixelTikTok
	}
	return p, nil
}
