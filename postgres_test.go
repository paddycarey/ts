package ts

import (
	"database/sql"
	"errors"
	"testing"

	_ "github.com/lib/pq"
)

func checkPostgres(s Store) error {

	db, err := sql.Open("postgres", s.URL().String())
	if err != nil {
		return err
	}
	defer db.Close()

	r, err := db.Query("select 1")
	if err != nil {
		return err
	}
	defer r.Close()

	if !r.Next() {
		return errors.New("expected row")
	}

	return nil
}

func TestPostgres(t *testing.T) {
	testStore(t, NewPostgres, checkPostgres)
}
