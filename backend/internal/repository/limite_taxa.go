package repository

import (
	"math/rand/v2"
	"time"

	"gorm.io/gorm"
)

// LimiteTaxaRepository conta tentativas por chave numa janela fixa, no
// Postgres (item 5.5): vale para todas as instâncias da API ao mesmo
// tempo, ao contrário do limite em memória que existia antes.
type LimiteTaxaRepository struct {
	db *gorm.DB
}

func NovoLimiteTaxaRepository(db *gorm.DB) *LimiteTaxaRepository {
	return &LimiteTaxaRepository{db: db}
}

// Registrar soma uma tentativa e devolve quantas já houve na janela atual.
// Um único comando (INSERT ... ON CONFLICT) — atômico mesmo com muitas
// requisições simultâneas.
func (r *LimiteTaxaRepository) Registrar(chave string, janela time.Duration, agora time.Time) (int, error) {
	// janela guarda o instante (unix) de início da janela, para janelas de
	// tamanhos diferentes conviverem na mesma tabela.
	seg := int64(janela.Seconds())
	inicio := agora.Unix() / seg * seg
	var contagem int
	err := r.db.Raw(`
		INSERT INTO limites_taxa (chave, janela, contagem) VALUES (?, ?, 1)
		ON CONFLICT (chave, janela) DO UPDATE SET contagem = limites_taxa.contagem + 1
		RETURNING contagem`, chave, inicio).Scan(&contagem).Error
	if err == nil && rand.IntN(200) == 0 {
		// Limpeza ocasional de janelas antigas (a tabela é UNLOGGED e pequena).
		r.db.Exec("DELETE FROM limites_taxa WHERE janela < ?", agora.Add(-2*time.Hour).Unix())
	}
	return contagem, err
}
