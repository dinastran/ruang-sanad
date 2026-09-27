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

func setupManagedUserTest(t *testing.T) (*sql.DB, *UserService, int64, int64, int64) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(db, filepath.Join("..", "..", "migrations")))

	res, err := db.Exec("INSERT INTO users (email, name, role) VALUES (?, ?, ?)", "admin@example.com", "Super Admin", "super_admin")
	require.NoError(t, err)
	adminID, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = db.Exec("INSERT INTO users (email, name, role) VALUES (?, ?, ?)", "target@example.com", "Target User", "cs")
	require.NoError(t, err)
	targetID, err := res.LastInsertId()
	require.NoError(t, err)
	res, err = db.Exec("INSERT INTO users (email, name, role) VALUES (?, ?, ?)", "cs@example.com", "CS User", "cs")
	require.NoError(t, err)
	csID, err := res.LastInsertId()
	require.NoError(t, err)

	q := queries.NewQuerier(db)
	return db, NewUserService(q), adminID, targetID, csID
}

func TestDeleteManagedUserAuthorizationAndSelfProtection(t *testing.T) {
	_, service, adminID, targetID, csID := setupManagedUserTest(t)

	err := service.DeleteManagedUser(csID, targetID)
	require.ErrorContains(t, err, "hanya Super Admin")

	err = service.DeleteManagedUser(adminID, adminID)
	require.ErrorContains(t, err, "sedang digunakan")

	require.NoError(t, service.DeleteManagedUser(adminID, targetID))
	_, err = service.GetProfile(targetID)
	require.Error(t, err)
}
