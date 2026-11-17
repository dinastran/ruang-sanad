package services

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newTasbihTestDB(t *testing.T) (*sql.DB, *TasbihService) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))
	return db, NewTasbihService(queries.NewQuerier(db))
}

var tasbihUserSeq int

func seedUser(t *testing.T, db *sql.DB, role string) int64 {
	t.Helper()
	tasbihUserSeq++
	res, err := db.Exec("INSERT INTO users (email, name, role) VALUES (?, ?, ?)",
		fmt.Sprintf("tasbih%d@test.local", tasbihUserSeq), "User Tasbih", role)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestTasbihSeed(t *testing.T) {
	db, _ := newTasbihTestDB(t)
	require.EqualValues(t, 12, countRows(t, db, "SELECT COUNT(*) FROM tasbih_dzikir WHERE user_id IS NULL"))
	require.EqualValues(t, 0, countRows(t, db, "SELECT COUNT(*) FROM tasbih_dzikir WHERE user_id IS NULL AND (teks_arab = '' OR arti = '' OR rujukan = '')"))
	require.EqualValues(t, 3, countRows(t, db, "SELECT COUNT(*) FROM tasbih_paket"))
	require.EqualValues(t, 4, countRows(t, db, "SELECT COUNT(*) FROM tasbih_paket_item i JOIN tasbih_paket p ON p.id = i.paket_id WHERE p.slug = 'badah-shalat'"))
	require.EqualValues(t, 1, countRows(t, db, "SELECT i.target FROM tasbih_paket_item i JOIN tasbih_paket p ON p.id = i.paket_id WHERE p.slug = 'badah-shalat' AND i.urutan = 4"))
	require.EqualValues(t, 34, countRows(t, db, "SELECT i.target FROM tasbih_paket_item i JOIN tasbih_paket p ON p.id = i.paket_id WHERE p.slug = 'tasbih-fatimah' AND i.urutan = 3"))
	require.EqualValues(t, 0, countRows(t, db, "SELECT COUNT(*) FROM tasbih_paket_item WHERE dzikir_id IS NULL"))
}
