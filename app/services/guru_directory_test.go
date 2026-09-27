package services

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/maulanashalihin/laju-go/app/queries"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestGuruDirectoryDetailIncludesLinkedUserIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	res, err := db.Exec(
		"INSERT INTO users (email, name, role) VALUES (?, ?, ?)",
		"guru@login.test", "Ustadz Login", "guru",
	)
	require.NoError(t, err)
	userID, err := res.LastInsertId()
	require.NoError(t, err)

	res, err = db.Exec(
		"INSERT INTO guru (nama, jenis_kelamin, user_id) VALUES (?, ?, ?)",
		"Ustadz Direktori", "L", userID,
	)
	require.NoError(t, err)
	guruID, err := res.LastInsertId()
	require.NoError(t, err)

	service := NewGuruService(queries.NewQuerier(db))
	guru, err := service.GetDirectoryByID(guruID)
	require.NoError(t, err)
	require.NotNil(t, guru.UserID)
	require.Equal(t, userID, *guru.UserID)
	require.Equal(t, "Ustadz Login", guru.LinkedUserName)
	require.Equal(t, "guru@login.test", guru.LinkedUserEmail)
}
