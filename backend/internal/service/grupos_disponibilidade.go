package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// A busca revela apenas disponibilidade; nunca devolve os motivos privados.
func filtrarDisponibilidadeArtista(q *gorm.DB, dia string) (*gorm.DB, error) {
	if dia == "" {
		return q, nil
	}
	if !diaDisponibilidadeValido(dia) {
		return nil, fmt.Errorf("%w: informe uma data válida em AAAA-MM-DD para buscar disponibilidade", ErrArtistaInvalido)
	}
	q = q.Where(`NOT EXISTS (SELECT 1 FROM grupo_agenda ga
 WHERE ga.grupo_id = g.id AND ga.inicio <= ?::date AND ga.fim >= ?::date)`, dia, dia)
	// Uma atração confirmada também é um compromisso, mesmo em evento privado
	// ou ainda não publicado. O dia segue o fuso do evento.
	q = q.Where(`NOT EXISTS (SELECT 1 FROM atracoes a JOIN eventos e ON e.id = a.evento_id
 WHERE a.grupo_id = g.id AND a.status = 'confirmado' AND e.status <> 'cancelado'
 AND (e.inicio_em AT TIME ZONE e.timezone)::date <= ?::date
 AND (COALESCE(e.fim_em,e.inicio_em + interval '12 hours') AT TIME ZONE e.timezone)::date >= ?::date)`, dia, dia)
	return q, nil
}

func diaDisponibilidadeValido(dia string) bool {
	d, err := time.Parse("2006-01-02", dia)
	return err == nil && d.Year() >= 1
}
