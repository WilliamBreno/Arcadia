package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

var ErrIBGEArtista = errors.New("não foi possível consultar os municípios agora; tente novamente")

type MunicipioArtista struct {
	ID        int      `json:"id" gorm:"primaryKey"`
	Nome      string   `json:"nome"`
	UF        string   `json:"uf" gorm:"column:uf"`
	Latitude  *float64 `json:"-"`
	Longitude *float64 `json:"-"`
}

func (MunicipioArtista) TableName() string { return "municipios_artistas" }

// Municípios e malhas simplificadas: APIs oficiais do IBGE. Dados em cache no banco.
const ibgeArtistasURL = "https://servicodados.ibge.gov.br"

func buscarIBGEArtista(caminho string, out any) error {
	c := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := c.Get(ibgeArtistasURL + caminho)
	if e != nil {
		return ErrIBGEArtista
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ErrIBGEArtista
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
	if e != nil || len(b) > 4*1024*1024 {
		return ErrIBGEArtista
	}
	if json.Unmarshal(b, out) != nil {
		return ErrIBGEArtista
	}
	return nil
}
func (s *GruposService) Municipios(uf string) ([]MunicipioArtista, error) {
	uf = strings.ToUpper(strings.TrimSpace(uf))
	if !strings.Contains("|AC|AL|AP|AM|BA|CE|DF|ES|GO|MA|MT|MS|MG|PA|PB|PR|PE|PI|RJ|RN|RS|RO|RR|SC|SP|SE|TO|", "|"+uf+"|") || len(uf) != 2 {
		return nil, ErrContratacao
	}
	out := []MunicipioArtista{}
	if e := s.db.Where("uf = ?", uf).Order("nome").Find(&out).Error; e != nil {
		return nil, e
	}
	if len(out) > 0 {
		return out, nil
	}
	var dados []struct {
		ID   int    `json:"id"`
		Nome string `json:"nome"`
	}
	if e := buscarIBGEArtista("/api/v1/localidades/estados/"+uf+"/municipios?orderBy=nome", &dados); e != nil {
		return nil, e
	}
	for _, d := range dados {
		if d.ID > 0 && d.Nome != "" {
			out = append(out, MunicipioArtista{ID: d.ID, Nome: d.Nome, UF: uf})
		}
	}
	if len(out) == 0 {
		return nil, ErrIBGEArtista
	}
	return out, s.db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&out, 200).Error
}

func (s *GruposService) coordenadaMunicipio(id int) (*MunicipioArtista, error) {
	var m MunicipioArtista
	if s.db.First(&m, id).Error != nil {
		return nil, ErrContratacao
	}
	if m.Latitude != nil && m.Longitude != nil {
		return &m, nil
	}
	var geo struct {
		Features []struct {
			Geometry struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if e := buscarIBGEArtista(fmt.Sprintf("/api/v3/malhas/municipios/%d", id)+"?formato="+url.QueryEscape("application/vnd.geo+json")+"&qualidade=minima", &geo); e != nil {
		return nil, e
	}
	minX, minY, maxX, maxY := 180.0, 90.0, -180.0, -90.0
	pontos := 0
	var percorrer func(any)
	percorrer = func(v any) {
		a, ok := v.([]any)
		if !ok {
			return
		}
		if len(a) >= 2 {
			if x, ok := a[0].(float64); ok {
				if y, ok := a[1].(float64); ok && x >= -180 && x <= 180 && y >= -90 && y <= 90 {
					minX = math.Min(minX, x)
					maxX = math.Max(maxX, x)
					minY = math.Min(minY, y)
					maxY = math.Max(maxY, y)
					pontos++
					return
				}
			}
		}
		for _, b := range a {
			percorrer(b)
		}
	}
	for _, f := range geo.Features {
		var v any
		if json.Unmarshal(f.Geometry.Coordinates, &v) == nil {
			percorrer(v)
		}
	}
	if pontos == 0 {
		return nil, ErrIBGEArtista
	}
	// Centro da extensão municipal: aproximação geográfica, não distância rodoviária nem tarifa real.
	lat, lon := (minY+maxY)/2, (minX+maxX)/2
	if e := s.db.Model(&m).Updates(map[string]any{"latitude": lat, "longitude": lon}).Error; e != nil {
		return nil, e
	}
	m.Latitude, m.Longitude = &lat, &lon
	return &m, nil
}

type ArtistaViagemConfig struct {
	ID               int       `json:"-" gorm:"primaryKey"`
	Habilitada       bool      `json:"habilitada"`
	OnibusCentavosKm *int64    `json:"onibus_centavos_km"`
	AviaoCentavosKm  *int64    `json:"aviao_centavos_km"`
	DistanciaMinKm   *int      `json:"distancia_min_km"`
	AviaoDesdeKm     *int      `json:"aviao_desde_km"`
	IdaVolta         bool      `json:"ida_volta"`
	AtualizadoPor    *int64    `json:"-"`
	AtualizadoEm     time.Time `json:"atualizado_em"`
}

func (ArtistaViagemConfig) TableName() string { return "artista_viagem_config" }
func validarViagemConfig(c ArtistaViagemConfig) error {
	for _, v := range []*int64{c.OnibusCentavosKm, c.AviaoCentavosKm} {
		if v != nil && (*v <= 0 || *v > 1000000) {
			return ErrContratacao
		}
	}
	for _, v := range []*int{c.DistanciaMinKm, c.AviaoDesdeKm} {
		if v != nil && (*v < 0 || *v > 40000) {
			return ErrContratacao
		}
	}
	if c.Habilitada && (c.OnibusCentavosKm == nil || c.AviaoCentavosKm == nil || c.DistanciaMinKm == nil || c.AviaoDesdeKm == nil) {
		return ErrContratacao
	}
	return nil
}
func (s *GruposService) ViagemConfig() (*ArtistaViagemConfig, error) {
	var c ArtistaViagemConfig
	e := s.db.First(&c, 1).Error
	return &c, e
}
func (s *GruposService) SalvarViagemConfig(usuarioID int64, c ArtistaViagemConfig) error {
	if e := validarViagemConfig(c); e != nil {
		return e
	}
	c.ID, c.AtualizadoPor, c.AtualizadoEm = 1, &usuarioID, time.Now()
	return s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&c).Error
}

type EstimativaArtista struct {
	Disponivel      bool   `json:"disponivel"`
	PassagemInclusa bool   `json:"passagem_inclusa"`
	DistanciaKm     int64  `json:"distancia_km"`
	Transporte      string `json:"transporte"`
	ViagemCentavos  *int64 `json:"viagem_centavos"`
	TotalCentavos   *int64 `json:"total_centavos"`
	PrecoTipo       string `json:"preco_tipo"`
	IdaVolta        bool   `json:"ida_volta"`
	Aviso           string `json:"aviso"`
}

func calcularViagemArtista(c ArtistaViagemConfig, km int64, pessoas int) (string, int64, error) {
	if validarViagemConfig(c) != nil || !c.Habilitada || km < 0 || km > 40000 || pessoas < 1 || pessoas > 1000 {
		return "", 0, ErrContratacao
	}
	if km < int64(*c.DistanciaMinKm) {
		return "sem_estimativa", 0, nil
	}
	multiplicador := int64(pessoas)
	if c.IdaVolta {
		multiplicador *= 2
	}
	onibus := km * *c.OnibusCentavosKm * multiplicador
	if km >= int64(*c.AviaoDesdeKm) {
		aviao := km * *c.AviaoCentavosKm * multiplicador
		if aviao < onibus {
			return "aviao", aviao, nil
		}
	}
	return "onibus", onibus, nil
}
func (s *GruposService) EstimarViagem(slug string, destinoID int) (*EstimativaArtista, error) {
	if _, e := s.Publico(slug); e != nil {
		return nil, e
	}
	var g Grupo
	if s.db.Where("slug = ?", slug).First(&g).Error != nil {
		return nil, ErrArtistaNaoEncontrado
	}
	// Não inferir a cidade de saída de artista solo menor.
	if g.Solo && s.situacaoSolo(&g, time.Now()).Menor {
		return &EstimativaArtista{Aviso: "Combine o deslocamento diretamente na conversa."}, nil
	}
	o, e := s.oferta(g.ID)
	if e != nil {
		return nil, e
	}
	if !o.Habilitada {
		return nil, ErrArtistaNaoEncontrado
	}
	r := &EstimativaArtista{PassagemInclusa: o.PassagemInclusa, PrecoTipo: o.PrecoTipo, Aviso: "Estimativa de deslocamento desativada. Combine os detalhes na conversa."}
	if o.PassagemInclusa {
		r.Aviso = "Passagem inclusa na oferta. Confirme as condições na proposta."
		r.TotalCentavos = o.ValorCentavos
		return r, nil
	}
	c, e := s.ViagemConfig()
	if e != nil {
		return nil, e
	}
	if !c.Habilitada {
		return r, nil
	}
	if o.OrigemID == nil {
		r.Aviso = "Cidade de saída não informada. Combine o deslocamento na conversa."
		return r, nil
	}
	origem, e := s.coordenadaMunicipio(*o.OrigemID)
	if e != nil {
		return nil, e
	}
	destino, e := s.coordenadaMunicipio(destinoID)
	if e != nil {
		return nil, e
	}
	lat1, lat2 := *origem.Latitude*math.Pi/180, *destino.Latitude*math.Pi/180
	deltaLat, deltaLon := (lat2 - lat1), (*destino.Longitude-*origem.Longitude)*math.Pi/180
	h := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	km := int64(math.Ceil(6371 * 2 * math.Asin(math.Sqrt(math.Min(1, h)))))
	modo, valor, e := calcularViagemArtista(*c, km, o.Integrantes)
	if e != nil {
		return nil, e
	}
	r.DistanciaKm, r.Transporte, r.IdaVolta = km, modo, c.IdaVolta
	if modo == "sem_estimativa" {
		r.Aviso = "Distância abaixo do mínimo configurado. Combine o deslocamento na conversa."
		return r, nil
	}
	r.Disponivel = true
	r.ViagemCentavos = &valor
	if o.ValorCentavos != nil {
		total := *o.ValorCentavos + valor
		r.TotalCentavos = &total
	}
	r.Aviso = "Estimativa, não cotação: distância em linha reta entre centros das malhas municipais do IBGE; valores por km e pessoa definidos pelo admin. Confirme passagem e valor final com o artista."
	return r, nil
}
