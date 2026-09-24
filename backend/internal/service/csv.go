package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/WilliamBreno/Arcadia/backend/internal/domain"
	"github.com/WilliamBreno/Arcadia/backend/internal/repository"
)

// celulaCSV neutraliza CSV/formula injection: células que começam com
// = + - @ (ou tab/CR) ganham um apóstrofo, senão o Excel executa.
func celulaCSV(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func reais(centavos int64) string {
	return fmt.Sprintf("%d,%02d", centavos/100, centavos%100)
}

func formatarQuando(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(time.Local).Format("02/01/2006 15:04")
}

// ExportarCompradores escreve o CSV de compradores/titulares (item 2.5).
func (s *CortesiaService) ExportarCompradores(w io.Writer, organizadorID, eventoID int64) error {
	if _, err := s.eventos.BuscarDoOrganizador(organizadorID, eventoID); err != nil {
		return err
	}
	itens, err := s.itensPedido.ListarParaExportacao(eventoID)
	if err != nil {
		return err
	}
	return escreverCompradoresCSV(w, itens)
}

func escreverCompradoresCSV(w io.Writer, itens []repository.ItemVenda) error {
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	_ = cw.Write([]string{"nome", "email", "tipo_ingresso", "codigo", "status", "cortesia", "preco", "desconto", "taxa", "garantia", "total", "criado_em", "utilizado_em"})
	for _, i := range itens {
		cortesia := "nao"
		if i.Cortesia {
			cortesia = "sim"
		}
		_ = cw.Write([]string{
			celulaCSV(i.TitularNome), celulaCSV(i.TitularEmail), celulaCSV(i.TipoIngressoNome), i.Codigo, string(i.Status), cortesia,
			reais(i.PrecoCentavos), reais(i.DescontoCentavos), reais(i.TaxaPlataformaCentavos), reais(i.GarantiaCentavos), reais(i.TotalCentavos),
			formatarQuando(&i.CriadoEm), formatarQuando(i.UtilizadoEm),
		})
	}
	cw.Flush()
	return cw.Error()
}

// EscreverParticipantesCSV: só dados de identificação/artísticos; telefone,
// data de nascimento e dados do responsável (menores) ficam de fora de
// propósito (dados pessoais sensíveis, ver seção 14).
func EscreverParticipantesCSV(w io.Writer, fichas []domain.FichaParticipacao) error {
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	_ = cw.Write([]string{"nome", "nome_artistico", "instagram", "papel", "status", "tipo_apresentacao"})
	for _, f := range fichas {
		tipo := ""
		if f.TipoApresentacao != nil {
			tipo = string(*f.TipoApresentacao)
		}
		_ = cw.Write([]string{celulaCSV(f.Nome), celulaCSV(f.NomeArtistico), celulaCSV(f.Instagram), string(f.Papel), string(f.Status), tipo})
	}
	cw.Flush()
	return cw.Error()
}
